package controllers

import (
	"container/list"
	"encoding/json"
	"html/template"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/TruthHun/BookStack/utils/html2md"
	"github.com/aliyun/aliyun-oss-go-sdk/oss"

	"image/png"

	"bytes"

	"fmt"

	"github.com/PuerkitoBio/goquery"
	"github.com/TruthHun/BookStack/conf"
	"github.com/TruthHun/BookStack/models"
	"github.com/TruthHun/BookStack/models/store"
	"github.com/TruthHun/BookStack/utils"
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/orm"
	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/qr"
)

var (
	videoBoxFmt = `<div class="video-box">
	<div class="video-main">
	  <div class="video-heading">
		<div class="video-title">%v</div>
		<div class="video-playbackrate">
		  <select>
			<option value="0.7">0.7 倍</option>
			<option value="1.0" selected>1.0 倍</option>
			<option value="1.2">1.2 倍</option>
			<option value="1.5">1.5 倍</option>
			<option value="2.0">2.0 倍</option>
		  </select>
		</div>
	  </div>
	  <video controls poster="%v" src="%v" controlslist="nodownload" preload="true">%v</video>
	</div>
  </div>
  `
)

// DocumentController struct.
type DocumentController struct {
	BaseController
}

func (this *DocumentController) Abort404(bookName, bookLink string) {
	this.Ctx.ResponseWriter.WriteHeader(404)
	this.Data["BookName"] = bookName
	this.Data["BookLink"] = bookLink
	this.TplName = "errors/404.html"
	b, err := this.RenderBytes()
	if err != nil {
		this.Abort("404")
	}
	this.Ctx.ResponseWriter.Write(b)
	this.StopRun()
}

// 解析并提取版本控制的commit内容
func parseGitCommit(str string) (cont, commit string) {
	var slice []string
	arr := strings.Split(str, "<bookstack-git>")
	if len(arr) > 1 {
		slice = append(slice, arr[0])
		str = strings.Join(arr[1:], "")
	}
	arr = strings.Split(str, "</bookstack-git>")
	if len(arr) > 1 {
		slice = append(slice, arr[1:]...)
		commit = arr[0]
	}
	if len(slice) > 0 {
		cont = strings.Join(slice, "")
	} else {
		cont = str
	}
	return
}

// 判断用户是否可以阅读文档.
func isReadable(identify, token string, this *DocumentController) *models.BookResult {
	book, err := models.NewBook().FindByFieldFirst("identify", identify)
	if err != nil {
		beego.Error(err)
		this.Abort("404")
	}

	//如果文档是私有的
	if book.PrivatelyOwned == 1 && !this.Member.IsAdministrator() {
		isOk := false
		if this.Member != nil {
			_, err := models.NewRelationship().FindForRoleId(book.BookId, this.Member.MemberId)
			if err == nil {
				isOk = true
			}
		}

		if book.PrivateToken != "" && !isOk {
			//如果有访问的Token，并且该书籍设置了访问Token，并且和用户提供的相匹配，则记录到Session中.
			//如果用户未提供Token且用户登录了，则判断用户是否参与了该书籍.
			//如果用户未登录，则从Session中读取Token.
			if token != "" && strings.EqualFold(token, book.PrivateToken) {
				this.SetSession(identify, token)
			} else if token, ok := this.GetSession(identify).(string); !ok || !strings.EqualFold(token, book.PrivateToken) {
				hasErr := ""
				if this.Ctx.Request.Method == "POST" {
					hasErr = "true"
				}
				this.Redirect(beego.URLFor("DocumentController.Index", ":key", identify)+"?with-password=true&err="+hasErr, 302)
				this.StopRun()
			}
		} else if !isOk {
			this.Abort("404")
		}
	}

	bookResult := book.ToBookResult()
	if this.Member != nil {
		rel, err := models.NewRelationship().FindByBookIdAndMemberId(bookResult.BookId, this.Member.MemberId)
		if err == nil {
			bookResult.MemberId = book.MemberId
			bookResult.RoleId = rel.RoleId
			bookResult.RelationshipId = rel.RelationshipId
		}
	}
	//判断是否需要显示评论框
	switch bookResult.CommentStatus {
	case "closed":
		bookResult.IsDisplayComment = false
	case "open":
		bookResult.IsDisplayComment = true
	case "group_only":
		bookResult.IsDisplayComment = bookResult.RelationshipId > 0
	case "registered_only":
		bookResult.IsDisplayComment = true
	}
	return bookResult
}

// 文档首页.
func (this *DocumentController) Index() {
	identify := this.Ctx.Input.Param(":key")
	if identify == "" {
		this.Abort("404")
	}

	token := this.GetString("token")
	if len(strings.TrimSpace(this.GetString("with-password"))) > 0 {
		this.indexWithPassword()
		return
	}

	tab := strings.ToLower(this.GetString("tab"))

	bookResult := isReadable(identify, token, this)
	if bookResult.BookId == 0 { //没有阅读权限
		this.Redirect(beego.URLFor("HomeController.Index"), 302)
		return
	}

	this.TplName = "document/intro.html"
	bookResult.Lang = utils.GetLang(bookResult.Lang)
	this.Data["Book"] = bookResult

	switch tab {
	case "comment", "score":
	default:
		tab = "default"
	}
	this.Data["ExistWeCode"] = strings.TrimSpace(models.GetOptionValue("DOWNLOAD_WECODE", "")) != ""
	this.Data["Qrcode"] = new(models.Member).GetQrcodeByUid(bookResult.MemberId)
	this.Data["MyScore"] = new(models.Score).BookScoreByUid(this.Member.MemberId, bookResult.BookId)
	this.Data["Tab"] = tab
	if beego.AppConfig.DefaultBool("showWechatCode", false) && bookResult.PrivatelyOwned == 0 {
		wechatCode := models.NewWechatCode()
		go wechatCode.CreateWechatCode(bookResult.BookId) //如果已经生成了小程序码，则不会再生成
		this.Data["Wxacode"] = wechatCode.GetCode(bookResult.BookId)
	}

	//当前默认展示1000条评论(暂时先默认1000条)
	comments, _ := new(models.Comments).Comments(1, 1000, models.CommentOpt{BookId: bookResult.BookId, Status: []int{1}, WithoutDocComment: true})

	this.Data["Comments"] = comments
	this.Data["Menu"], _ = new(models.Document).GetMenuTop(bookResult.BookId)
	title := "《" + bookResult.BookName + "》"
	if tab == "comment" {
		title = "点评 - " + title
	}
	this.GetSeoByPage("book_info", map[string]string{
		"title":       title,
		"keywords":    bookResult.Label,
		"description": bookResult.Description,
	})
	this.Data["RelateBooks"] = models.NewRelateBook().Lists(bookResult.BookId)
	this.Data["Versions"] = models.NewVersion().GetPublicVersionItems(bookResult.Identify)
}

// 文档首页.
func (this *DocumentController) indexWithPassword() {
	identify := this.Ctx.Input.Param(":key")
	if identify == "" {
		this.Abort("404")
	}
	this.TplName = "document/read-with-password.html"
	this.GetSeoByPage("book_info", map[string]string{
		"title":       "密码访问",
		"keywords":    "密码访问",
		"description": "密码访问",
	})
	this.Data["ShowErrTips"] = this.GetString("err") != ""
	this.Data["Identify"] = identify
}

// 阅读文档.
func (this *DocumentController) Read() {

	identify := this.Ctx.Input.Param(":key")
	token := this.GetString("token")
	id := this.GetString(":id")

	if identify == "" || id == "" {
		this.Abort("404")
	}

	//如果没有开启你们匿名则跳转到登录
	if !this.EnableAnonymous && this.Member == nil {
		this.Redirect(beego.URLFor("AccountController.Login"), 302)
		return
	}

	bookResult := isReadable(identify, token, this)
	bookName := bookResult.BookName
	bookLink := beego.URLFor("DocumentController.Index", ":key", bookResult.Identify)

	this.TplName = "document/" + bookResult.Theme + "_read.html"

	var err error

	doc := models.NewDocument()
	if docId, _ := strconv.Atoi(id); docId > 0 {
		doc, err = doc.Find(docId) //文档id
		if err != nil {
			beego.Error(err)
			this.Abort404(bookName, bookLink)
		}
	} else {
		//此处的id是字符串，标识文档标识，根据文档标识和文档所属的书的id作为key去查询
		doc, err = doc.FindByBookIdAndDocIdentify(bookResult.BookId, id) //文档标识
		if err != nil {
			if err != orm.ErrNoRows {
				beego.Error(err, docId, id, bookResult)
			}
			this.Abort404(bookName, bookLink)
		}
	}

	if doc.BookId != bookResult.BookId {
		this.Abort404(bookName, bookLink)
	}

	// 是否允许阅读
	isAllowRead := true
	percent := 100
	if this.Member.MemberId == 0 {
		isAllowRead, percent = doc.IsAllowReadChapter(doc.BookId, doc.DocumentId)
	}

	bodyText := ""
	authHTTPS := strings.ToLower(models.GetOptionValue("AUTO_HTTPS", "false")) == "true"
	if doc.Release != "" {
		query, err := goquery.NewDocumentFromReader(bytes.NewBufferString(doc.Release))
		if err != nil {
			beego.Error(err)
		} else {
			query.Find("img").Each(func(i int, contentSelection *goquery.Selection) {
				src, ok := contentSelection.Attr("src")
				if ok {
					if utils.StoreType == utils.StoreOss && !(strings.HasPrefix(src, "https://") || strings.HasPrefix(src, "http://")) {
						src = this.OssDomain + "/" + strings.TrimLeft(src, "./")
					}
				}
				if authHTTPS {
					if srcArr := strings.Split(src, "://"); len(srcArr) > 1 {
						src = "https://" + strings.Join(srcArr[1:], "://")
					}
				}
				contentSelection.SetAttr("data-original", src).SetAttr("src", "/static/images/loading.gif").AddClass("lazy")
				if alt, _ := contentSelection.Attr("alt"); alt == "" {
					contentSelection.SetAttr("alt", doc.DocumentName+" - 图"+fmt.Sprint(i+1))
				}
			})

			medias := []string{"video", "audio"}
			for _, item := range medias {
				query.Find(item).Each(func(idx int, sel *goquery.Selection) {
					title := strings.TrimSpace(sel.Text())
					poster, _ := sel.Attr("poster")
					src, _ := sel.Attr("src")
					if !(strings.HasPrefix(src, "https://") || strings.HasPrefix(src, "http://")) {
						sign, _ := utils.GenerateMediaSign(src, time.Now().UnixNano(), time.Duration(utils.MediaDuration)*time.Second)
						if strings.Contains(src, "?") {
							src = src + "&sign=" + sign
						} else {
							src = src + "?sign=" + sign
						}
					}
					if item == "video" {
						sel.BeforeHtml(fmt.Sprintf(videoBoxFmt, title, poster, src, title))
						sel.Remove()
					} else {
						sel.SetAttr("preload", "true")
						sel.SetAttr("controlslist", "nodownload")
						sel.SetAttr("src", src)
					}
				})
			}

			html, err := query.Find("body").Html()
			if err != nil {
				beego.Error(err)
			} else {
				doc.Release = html
			}
		}
		bodyText = query.Find(".markdown-toc").Text()
	}

	attach, err := models.NewAttachment().FindListByDocumentId(doc.DocumentId)
	if err == nil {
		doc.AttachList = attach
		if len(attach) > 0 {
			content := bytes.NewBufferString("<div class=\"attach-list\"><strong>附件</strong><ul>")
			for _, item := range attach {
				li := fmt.Sprintf("<li><a href=\"%s\" target=\"_blank\" title=\"%s\">%s</a></li>", item.HttpPath, item.FileName, item.FileName)
				content.WriteString(li)
			}
			content.WriteString("</ul></div>")
			doc.Release += content.String()
		}
	}

	//文档阅读人次+1
	if err := models.SetIncreAndDecre("md_documents", "vcnt",
		fmt.Sprintf("document_id=%v", doc.DocumentId),
		true, 1,
	); err != nil {
		beego.Error(err.Error())
	}
	//书籍阅读人次+1
	if err := models.SetIncreAndDecre("md_books", "vcnt",
		fmt.Sprintf("book_id=%v", doc.BookId),
		true, 1,
	); err != nil {
		beego.Error(err.Error())
	}

	if this.Member.MemberId > 0 { //增加用户阅读记录
		if err := new(models.ReadRecord).Add(doc.DocumentId, this.Member.MemberId); err != nil {
			beego.Error(err.Error())
		}
	}
	parentTitle := doc.GetParentTitle(doc.ParentId)
	seo := map[string]string{
		"title":       doc.DocumentName + " - 《" + bookResult.BookName + "》",
		"keywords":    bookResult.Label,
		"description": beego.Substr(bodyText+" "+bookResult.Description, 0, 200),
	}

	if len(parentTitle) > 0 {
		seo["title"] = parentTitle + " - " + doc.DocumentName + " - 《" + bookResult.BookName + "》"
	}

	//SEO
	this.GetSeoByPage("book_read", seo)

	existBookmark := new(models.Bookmark).Exist(this.Member.MemberId, doc.DocumentId)

	doc.Vcnt = doc.Vcnt + 1

	models.NewBookCounter().Increase(bookResult.BookId, true)
	comments, _ := models.NewComments().Comments(1, 1000, models.CommentOpt{DocId: doc.DocumentId, Status: []int{1}})

	if !isAllowRead {
		doc.Release = ""
	}

	if this.IsAjax() {
		var data struct {
			Id          int                         `json:"doc_id"`
			DocTitle    string                      `json:"doc_title"`
			Body        string                      `json:"body"`
			Title       string                      `json:"title"`
			Bookmark    bool                        `json:"bookmark"` //是否已经添加了书签
			View        int                         `json:"view"`
			UpdatedAt   string                      `json:"updated_at"`
			Comments    []models.BookCommentsResult `json:"comments"`
			IsAllowRead bool                        `json:"is_allow_read"`
			Percent     int                         `json:"percent"`
		}
		data.DocTitle = doc.DocumentName
		data.Body = doc.Release
		data.Id = doc.DocumentId
		data.Title = this.Data["SeoTitle"].(string)
		data.Bookmark = existBookmark
		data.View = doc.Vcnt
		data.UpdatedAt = doc.ModifyTime.Format("2006-01-02 15:04:05")
		data.Comments = comments
		data.IsAllowRead = isAllowRead
		data.Percent = percent
		this.JsonResult(0, "ok", data)
	}

	tree, err := models.NewDocument().CreateDocumentTreeForHtml(bookResult.BookId, doc.DocumentId)

	if err != nil {
		beego.Error(err)
		this.Abort404(bookName, bookLink)
	}

	// 查询用户哪些文档阅读了
	var readMap = make(map[string]bool)
	if this.Member.MemberId > 0 {
		modelRecord := new(models.ReadRecord)
		lists, cnt, _ := modelRecord.List(this.Member.MemberId, bookResult.BookId)
		if cnt > 0 {
			for _, item := range lists {
				readMap[strconv.Itoa(item.DocId)] = true
			}
		}
	}

	this.Data["ToggleMenu"] = false
	if menuDoc, err := goquery.NewDocumentFromReader(strings.NewReader(tree)); err == nil {
		menuDoc.Find("li").Each(func(i int, selection *goquery.Selection) {
			if id, exist := selection.Attr("id"); exist {
				if _, ok := readMap[id]; ok {
					selection.AddClass("readed")
				}
			}
		})
		hide := models.GetOptionValue("COLLAPSE_HIDE", "true") == "true"
		menuDoc.Find("ul").Each(func(i int, selection *goquery.Selection) {
			if selection.Parent().Is("li") {
				selection.Parent().AddClass("collapse-node")
				selection.Parent().PrependHtml("<span></span>")
				this.Data["ToggleMenu"] = true
				if hide {
					selection.Parent().AddClass("collapse-hide")
				}
			}
		})
		menuDoc.Find(".jstree-clicked").Parents().RemoveClass("collapse-hide")
		tree, _ = menuDoc.Find("body").Html()
	}

	if beego.AppConfig.DefaultBool("showWechatCode", false) && bookResult.PrivatelyOwned == 0 {
		wechatCode := models.NewWechatCode()
		go wechatCode.CreateWechatCode(bookResult.BookId) //如果已经生成了小程序码，则不会再生成
		this.Data["Wxacode"] = wechatCode.GetCode(bookResult.BookId)
	}

	if wd := this.GetString("wd"); strings.TrimSpace(wd) != "" {
		this.Data["Keywords"] = models.NewElasticSearchClient().SegWords(wd)
	}
	this.Data["IsThemeDark"] = this.Ctx.GetCookie("theme-dark") == "true"
	this.Data["Bookmark"] = existBookmark
	this.Data["Model"] = bookResult
	this.Data["Book"] = bookResult //文档下载需要用到Book变量
	this.Data["Result"] = template.HTML(tree)
	this.Data["Title"] = doc.DocumentName
	this.Data["DocId"] = doc.DocumentId
	this.Data["Content"] = template.HTML(doc.Release)
	this.Data["View"] = doc.Vcnt
	this.Data["UpdatedAt"] = doc.ModifyTime.Format("2006-01-02 15:04:05")
	this.Data["ExistWeCode"] = strings.TrimSpace(models.GetOptionValue("DOWNLOAD_WECODE", "")) != ""
	this.Data["Comments"] = comments
	this.Data["IsAllowRead"] = isAllowRead
	this.Data["Percent"] = percent
	this.Data["Versions"] = models.NewVersion().GetPublicVersionItems(bookResult.Identify)
}

func (this *DocumentController) ReadBook() {
	identify := this.Ctx.Input.Param(":key")
	if identify == "" {
		this.Abort("404")
		return
	}

	book, err := models.NewBook().FindByIdentify(identify, "book_id")
	if err != nil || book.BookId == 0 {
		this.Abort("404")
		return
	}

	doc, err := models.NewDocument().FirstChapter(book.BookId, "document_id", "identify")
	if err != nil || doc.DocumentId == 0 {
		this.Abort("404")
		return
	}

	if sign := this.GetString("sign"); sign != "" {
		this.SetSession(book.BookId, sign)
	}

	this.Redirect(beego.URLFor("DocumentController.Read", ":key", identify, ":id", doc.Identify), 302)
}

// 编辑文档.
func (this *DocumentController) Edit() {
	docId := 0 // 文档id

	identify := this.Ctx.Input.Param(":key")
	if identify == "" {
		this.Abort("404")
	}

	bookResult := models.NewBookResult()

	var err error
	//如果是超级管理者，则不判断权限
	if this.Member.IsAdministrator() {
		book, err := models.NewBook().FindByFieldFirst("identify", identify)
		if err != nil {
			this.JsonResult(6002, "书籍不存在或权限不足")
		}
		bookResult = book.ToBookResult()
	} else {
		bookResult, err = models.NewBookResult().FindByIdentify(identify, this.Member.MemberId)
		if err != nil {
			beego.Error("DocumentController.Edit => ", err)
			this.Abort("404")
		}

		if bookResult.RoleId == conf.BookObserver {
			this.JsonResult(6002, "书籍不存在或权限不足")
		}
	}

	//根据不同编辑器类型加载编辑器【注：现在只支持markdown】
	this.TplName = "document/markdown_edit_template.html"

	this.Data["Model"] = bookResult
	r, _ := json.Marshal(bookResult)

	this.Data["ModelResult"] = template.JS(string(r))

	this.Data["Result"] = template.JS("[]")

	// 编辑的文档
	if id := this.GetString(":id"); id != "" {
		if num, _ := strconv.Atoi(id); num > 0 {
			docId = num
		} else { //字符串
			var doc = models.NewDocument()
			orm.NewOrm().QueryTable(doc).Filter("identify", id).Filter("book_id", bookResult.BookId).One(doc, "document_id")
			docId = doc.DocumentId
		}
	}

	trees, err := models.NewDocument().FindDocumentTree(bookResult.BookId, docId, true)
	if err != nil {
		beego.Error("FindDocumentTree => ", err)
	} else {
		if len(trees) > 0 {
			if jsTree, err := json.Marshal(trees); err == nil {
				this.Data["Result"] = template.JS(string(jsTree))
			}
		} else {
			this.Data["Result"] = template.JS("[]")
		}
	}
	this.Data["BaiDuMapKey"] = beego.AppConfig.DefaultString("baidumapkey", "")
	installedDependencies := utils.GetInstalledDependencies()
	for _, item := range installedDependencies {
		this.Data[item.Name+"_is_installed"] = item.IsInstalled
	}
}

// 创建一个文档.
func (this *DocumentController) Create() {
	identify := this.GetString("identify")        //书籍书籍标识
	docIdentify := this.GetString("doc_identify") //新建的文档标识
	docName := this.GetString("doc_name")
	parentId, _ := this.GetInt("parent_id", 0)
	docId, _ := this.GetInt("doc_id", 0)
	bookIdentify := strings.TrimSpace(this.GetString(":key"))

	o := orm.NewOrm()

	if identify == "" {
		this.JsonResult(6001, "参数错误")
	}
	if docName == "" {
		this.JsonResult(6004, "文档名称不能为空")
	}
	if docIdentify != "" {
		if ok, err := regexp.MatchString(`^[a-zA-Z0-9_\-\.]*$`, docIdentify); !ok || err != nil {
			this.JsonResult(6003, "文档标识只能是数字、字母，以及“-”、“_”和“.”等字符，并且不能是纯数字")
		}
		if num, _ := strconv.Atoi(docIdentify); docIdentify == "0" || strconv.Itoa(num) == docIdentify { //不能是纯数字
			this.JsonResult(6005, "文档标识只能是数字、字母，以及“-”、“_”和“.”等字符，并且不能是纯数字")
		}

		if bookIdentify == "" {
			this.JsonResult(1, "书籍参数不正确")
		}

		var book models.Book
		o.QueryTable("md_books").Filter("Identify", bookIdentify).One(&book, "BookId")
		if book.BookId == 0 {
			this.JsonResult(1, "书籍未创建")
		}

		d, _ := models.NewDocument().FindByBookIdAndDocIdentify(book.BookId, docIdentify)
		if d.DocumentId > 0 && d.DocumentId != docId {
			this.JsonResult(6006, "文档标识已被使用")
		}
	} else {
		docIdentify = fmt.Sprintf("date%v", time.Now().Format("20060102150405.md"))
	}

	bookId := 0
	//如果是超级管理员则不判断权限
	if this.Member.IsAdministrator() {
		book, err := models.NewBook().FindByFieldFirst("identify", identify)
		if err != nil {
			beego.Error(err)
			this.JsonResult(6002, "书籍不存在或权限不足")
		}
		bookId = book.BookId
	} else {
		bookResult, err := models.NewBookResult().FindByIdentify(identify, this.Member.MemberId)

		if err != nil || bookResult.RoleId == conf.BookObserver {
			beego.Error("FindByIdentify => ", err)
			this.JsonResult(6002, "书籍不存在或权限不足")
		}
		bookId = bookResult.BookId
	}

	if parentId > 0 {
		doc, err := models.NewDocument().Find(parentId)
		if err != nil || doc.BookId != bookId {
			this.JsonResult(6003, "父分类不存在")
		}
	}

	document, _ := models.NewDocument().Find(docId)

	document.MemberId = this.Member.MemberId
	document.BookId = bookId
	oldIdentify := document.Identify
	if docIdentify != "" {
		document.Identify = docIdentify
	}
	document.Version = time.Now().Unix()
	document.DocumentName = docName
	document.ParentId = parentId

	docIdInt64, err := document.InsertOrUpdate()
	if err != nil {
		beego.Error("InsertOrUpdate => ", err)
		this.JsonResult(6005, "保存失败")
	}

	ModelStore := new(models.DocumentStore)
	if ModelStore.GetFiledById(docIdInt64, "markdown") == "" {
		//因为创建和更新文档基本信息都调用的这个接口，先判断markdown是否有内容，没有内容则添加默认内容
		if err := ModelStore.InsertOrUpdate(models.DocumentStore{DocumentId: int(docIdInt64), Markdown: "[TOC]\n\r\n\r"}); err != nil {
			beego.Error(err)
		}
	}
	if docId > 0 && oldIdentify != docIdentify {
		go document.ReplaceIdentify(bookId, oldIdentify, docIdentify)
	}
	this.JsonResult(0, "ok", document)
}

// 批量创建文档
func (this *DocumentController) CreateMulti() {
	bookId, _ := this.GetInt("book_id")

	if !(this.Member.MemberId > 0 && bookId > 0) {
		this.JsonResult(1, "操作失败：只有书籍创始人才能批量添加")
	}

	var book models.Book
	o := orm.NewOrm()
	o.QueryTable("md_books").Filter("book_id", bookId).Filter("member_id", this.Member.MemberId).One(&book, "book_id")
	if book.BookId > 0 {
		content := this.GetString("content")
		slice := strings.Split(content, "\n")
		if len(slice) > 0 {
			ModelStore := new(models.DocumentStore)
			for _, row := range slice {
				if chapter := strings.Split(strings.TrimSpace(row), " "); len(chapter) > 1 {
					if ok, err := regexp.MatchString(`^[a-zA-Z0-9_\-\.]*$`, chapter[0]); ok && err == nil {
						i, _ := strconv.Atoi(chapter[0])
						if chapter[0] != "0" && strconv.Itoa(i) != chapter[0] { //不为纯数字
							doc := models.Document{
								DocumentName: strings.Join(chapter[1:], " "),
								Identify:     chapter[0],
								BookId:       bookId,
								//Markdown:     "[TOC]\n\r",
								MemberId: this.Member.MemberId,
							}
							if docId, err := doc.InsertOrUpdate(); err == nil {
								if err := ModelStore.InsertOrUpdate(models.DocumentStore{DocumentId: int(docId), Markdown: "[TOC]\n\r\n\r"}); err != nil {
									beego.Error(err.Error())
								}
							} else {
								beego.Error(err)
							}
						}

					}
				}
			}
		}
	}
	this.JsonResult(0, "添加成功")
}

// 上传附件或图片.
func (this *DocumentController) Upload() {
	identify := this.GetString("identify")
	docId, _ := this.GetInt("doc_id")
	fileType := this.GetString("type")

	if identify == "" {
		this.JsonResult(6001, "参数错误")
	}

	name := "editormd-file-file"
	if docId == 0 {
		if fileType != "" && !strings.Contains(fileType, "/") {
			name = "editormd-" + fileType + "-file"
		} else {
			fileType = strings.Split(fileType, "/")[0]
		}
	}

	file, moreFile, err := this.GetFile(name)
	if err == http.ErrMissingFile {
		name = "editormd-image-file"
		file, moreFile, err = this.GetFile(name)
		if err == http.ErrMissingFile {
			this.JsonResult(6003, "没有发现需要上传的文件")
		}
	}

	if err != nil {
		this.JsonResult(6002, err.Error())
	}

	defer file.Close()

	ext := filepath.Ext(moreFile.Filename)
	if ext == "" {
		this.JsonResult(6003, "无法解析文件的格式")
	}

	if !conf.IsAllowUploadFileExt(ext, fileType) {
		this.JsonResult(6004, "不允许的文件类型")
	}

	bookId := 0
	bookIdentify := ""
	//如果是超级管理员，则不判断权限
	if this.Member.IsAdministrator() {
		book, err := models.NewBook().FindByFieldFirst("identify", identify)
		if err != nil {
			this.JsonResult(6006, "文档不存在或权限不足")
		}
		bookId = book.BookId
		bookIdentify = book.Identify
	} else {
		book, err := models.NewBookResult().FindByIdentify(identify, this.Member.MemberId)
		if err != nil {
			beego.Error("DocumentController.Edit => ", err)
			if err == orm.ErrNoRows {
				this.JsonResult(6006, "权限不足")
			}
			this.JsonResult(6001, err.Error())
		}
		//如果没有编辑权限
		if book.RoleId != conf.BookEditor && book.RoleId != conf.BookAdmin && book.RoleId != conf.BookFounder {
			this.JsonResult(6006, "权限不足")
		}
		bookId = book.BookId
		bookIdentify = book.Identify
	}

	if docId > 0 {
		doc, err := models.NewDocument().Find(docId)
		if err != nil {
			this.JsonResult(6007, "文档不存在")
		}
		if doc.BookId != bookId {
			this.JsonResult(6008, "文档不属于指定的书籍")
		}
	} else {
		identify := filepath.Base(this.Ctx.Request.Header.Get("referer"))
		doc, err := models.NewDocument().FindByBookIdAndDocIdentify(bookId, identify)
		if err != nil {
			this.JsonResult(6007, "文档不存在")
		}
		docId = doc.DocumentId
	}

	fileName := strconv.FormatInt(time.Now().UnixNano(), 16)

	tmpPath := fmt.Sprintf("cache/%v-%v-%v", bookIdentify, time.Now().Format("200601"), fileName+ext)
	err = this.SaveToFile(name, tmpPath)
	if err != nil {
		beego.Error("SaveToFile => ", err)
		this.JsonResult(6005, "保存文件失败")
	}

	defer func() { os.Remove(tmpPath) }()

	prefix := "uploads"
	savePath := filepath.Join("projects", bookIdentify, time.Now().Format("200601"), fileName+ext)
	if utils.StoreType != utils.StoreOss {
		savePath = filepath.Join(prefix, savePath)
		os.MkdirAll(filepath.Dir(savePath), os.ModePerm)
		os.Rename(tmpPath, savePath)
	}

	savePath = strings.ReplaceAll(savePath, "\\", "/")

	attachment := models.NewAttachment()
	attachment.BookId = bookId
	attachment.FileName = moreFile.Filename
	attachment.CreateAt = this.Member.MemberId
	attachment.FileExt = ext
	attachment.FilePath = "/" + savePath
	attachment.HttpPath = attachment.FilePath
	attachment.DocumentId = docId

	// 非附件
	if name != "editormd-file-file" {
		attachment.DocumentId = 0
	}

	if fileInfo, err := os.Stat(savePath); err == nil {
		attachment.FileSize = float64(fileInfo.Size())
	}

	// 数据入库
	if err = attachment.Insert(); err != nil {
		os.Remove(savePath)
		beego.Error("Attachment Insert => ", err)
		this.JsonResult(6006, "文件保存失败")
	}

	if utils.StoreType == utils.StoreOss {
		if err := store.ModelStoreOss.MoveToOss(tmpPath, savePath, true, false); err != nil {
			beego.Error(err.Error())
		} else {
			if fileType == "video" || fileType == "audio" {
				if bucket, err := store.ModelStoreOss.GetBucket(); err == nil {
					bucket.SetObjectACL(savePath, oss.ACLPrivate)
				}
			}
		}
	}

	result := map[string]interface{}{
		"errcode":   0,
		"success":   1,
		"message":   "ok",
		"url":       attachment.HttpPath,
		"alt":       attachment.FileName,
		"is_attach": attachment.DocumentId > 0,
		"attach":    attachment,
	}
	this.Ctx.Output.JSON(result, true, false)
	this.StopRun()
}

// DownloadAttachment 下载附件.
func (this *DocumentController) DownloadAttachment() {
	identify := this.Ctx.Input.Param(":key")
	attachId, _ := strconv.Atoi(this.Ctx.Input.Param(":attach_id"))
	token := this.GetString("token")

	memberId := 0

	if this.Member != nil {
		memberId = this.Member.MemberId
	}
	bookId := 0

	//判断用户是否参与了书籍
	bookResult, err := models.NewBookResult().FindByIdentify(identify, memberId)

	if err != nil {
		//判断书籍公开状态
		book, err := models.NewBook().FindByFieldFirst("identify", identify)
		if err != nil {
			this.Abort("404")
		}
		//如果不是超级管理员则判断权限
		if this.Member == nil || this.Member.Role != conf.MemberSuperRole {
			//如果书籍是私有的，并且token不正确
			if (book.PrivatelyOwned == 1 && token == "") || (book.PrivatelyOwned == 1 && book.PrivateToken != token) {
				this.Abort("404")
			}
		}

		bookId = book.BookId
	} else {
		bookId = bookResult.BookId
	}
	//查找附件
	attachment, err := models.NewAttachment().Find(attachId)

	if err != nil {
		beego.Error("DownloadAttachment => ", err)
		if err == orm.ErrNoRows {
			this.Abort("404")
		} else {
			this.Abort("404")
		}
	}
	if attachment.BookId != bookId {
		this.Abort("404")
	}
	this.Ctx.Output.Download(strings.TrimLeft(attachment.FilePath, "./"), attachment.FileName)

	this.StopRun()
}

// 删除附件.
func (this *DocumentController) RemoveAttachment() {
	attachId, _ := this.GetInt("attach_id")
	if attachId <= 0 {
		this.JsonResult(6001, "参数错误")
	}

	attach, err := models.NewAttachment().Find(attachId)
	if err != nil {
		beego.Error(err)
		this.JsonResult(6002, "附件不存在")
	}

	document, err := models.NewDocument().Find(attach.DocumentId)
	if err != nil {
		beego.Error(err)
		this.JsonResult(6003, "文档不存在")
	}

	if this.Member.Role != conf.MemberSuperRole {
		rel, err := models.NewRelationship().FindByBookIdAndMemberId(document.BookId, this.Member.MemberId)
		if err != nil {
			beego.Error(err)
			this.JsonResult(6004, "权限不足")
		}
		if rel.RoleId == conf.BookObserver {
			this.JsonResult(6004, "权限不足")
		}
	}

	if err = attach.Delete(); err != nil {
		beego.Error(err)
		this.JsonResult(6005, "删除失败")
	}

	os.Remove(strings.TrimLeft(attach.FilePath, "./"))
	this.JsonResult(0, "ok", attach)
}

// 删除文档.
func (this *DocumentController) Delete() {

	identify := this.GetString("identify")
	docId, _ := this.GetInt("doc_id", 0)

	bookId := 0
	//如果是超级管理员则忽略权限判断
	if this.Member.IsAdministrator() {
		book, err := models.NewBook().FindByFieldFirst("identify", identify)
		if err != nil {
			beego.Error("FindByIdentify => ", err)
			this.JsonResult(6002, "书籍不存在或权限不足")
		}
		bookId = book.BookId
	} else {
		bookResult, err := models.NewBookResult().FindByIdentify(identify, this.Member.MemberId)
		if err != nil || bookResult.RoleId == conf.BookObserver {
			beego.Error("FindByIdentify => ", err)
			this.JsonResult(6002, "书籍不存在或权限不足")
		}
		bookId = bookResult.BookId
	}

	if docId <= 0 {
		this.JsonResult(6001, "参数错误")
	}

	doc, err := models.NewDocument().Find(docId)
	if err != nil {
		beego.Error("Delete => ", err)
		this.JsonResult(6003, "删除失败")
	}

	//如果文档所属书籍错误
	if doc.BookId != bookId {
		this.JsonResult(6004, "参数错误")
	}
	//递归删除书籍下的文档以及子文档
	err = doc.RecursiveDocument(doc.DocumentId)
	if err != nil {
		beego.Error(err.Error())
		this.JsonResult(6005, "删除失败")
	}

	//重置文档数量统计
	models.NewBook().ResetDocumentNumber(doc.BookId)

	go func() {
		// 删除文档的索引
		client := models.NewElasticSearchClient()
		if errDel := client.DeleteIndex(docId, false); errDel != nil && client.On {
			beego.Error(errDel.Error())
		}
	}()

	this.JsonResult(0, "ok")
}

// 获取或更新文档内容.
func (this *DocumentController) Content() {
	identify := this.Ctx.Input.Param(":key")
	docId, err := this.GetInt("doc_id")
	errMsg := "ok"
	if err != nil {
		docId, _ = strconv.Atoi(this.Ctx.Input.Param(":id"))
	}
	bookId := 0
	//如果是超级管理员，则忽略权限
	if this.Member.IsAdministrator() {
		book, err := models.NewBook().FindByFieldFirst("identify", identify)
		if err != nil {
			this.JsonResult(6002, "书籍不存在或权限不足")
		}
		bookId = book.BookId
	} else {
		bookResult, err := models.NewBookResult().FindByIdentify(identify, this.Member.MemberId)

		if err != nil || bookResult.RoleId == conf.BookObserver {
			beego.Error("FindByIdentify => ", err)
			this.JsonResult(6002, "书籍不存在或权限不足")
		}
		bookId = bookResult.BookId
	}

	if docId <= 0 {
		this.JsonResult(6001, "参数错误")
	}

	ModelStore := new(models.DocumentStore)

	if !this.Ctx.Input.IsPost() {
		doc, err := models.NewDocument().Find(docId)

		if err != nil {
			this.JsonResult(6003, "文档不存在")
		}
		attach, err := models.NewAttachment().FindListByDocumentId(doc.DocumentId)
		if err == nil {
			doc.AttachList = attach
		}

		//为了减少数据的传输量，这里Release和Content的内容置空，前端会根据markdown文本自动渲染
		//doc.Release = ""
		//doc.Content = ""
		doc.Markdown = ModelStore.GetFiledById(doc.DocumentId, "markdown")
		this.JsonResult(0, errMsg, doc)
	}

	//更新文档内容
	markdown := strings.TrimSpace(this.GetString("markdown", ""))
	content := this.GetString("html")

	// 文档拆分
	gq, err := goquery.NewDocumentFromReader(strings.NewReader(content))
	if err == nil {
		seg := gq.Find("bookstack-split").Text()
		if strings.Contains(seg, "#") {
			markdown = strings.Replace(markdown, fmt.Sprintf("<bookstack-split>%v</bookstack-split>", seg), "", -1)
			err := new(models.Document).SplitMarkdownAndStore(seg, markdown, docId)
			if err == nil {
				this.JsonResult(0, "true")
			} else {
				this.JsonResult(1, err.Error())
			}
		}
	}

	version, _ := this.GetInt64("version", 0)
	isCover := this.GetString("cover")

	doc, err := models.NewDocument().Find(docId)

	if err != nil {
		this.JsonResult(6003, "读取文档错误")
	}
	if doc.BookId != bookId {
		this.JsonResult(6004, "保存的文档不属于指定书籍")
	}
	if doc.Version != version && !strings.EqualFold(isCover, "yes") {
		beego.Info("%d|", version, doc.Version)
		this.JsonResult(6005, "文档已被修改确定要覆盖吗？")
	}

	isSummary := false
	isAuto := false
	//替换文档中的url链接
	if strings.ToLower(doc.Identify) == "summary.md" && (strings.Contains(markdown, "<bookstack-summary></bookstack-summary>") || strings.Contains(doc.Markdown, "<bookstack-summary/>")) {
		//如果标识是summary.md，并且带有bookstack的标签，则表示更新目录
		isSummary = true
		//要清除，避免每次保存的时候都要重新排序
		replaces := []string{"<bookstack-summary></bookstack-summary>", "<bookstack-summary/>"}
		for _, r := range replaces {
			markdown = strings.Replace(markdown, r, "", -1)
			content = strings.Replace(content, r, "", -1)
		}
	}

	//爬虫采集
	access := this.Member.IsAdministrator()
	if op, err := new(models.Option).FindByKey("SPIDER"); err == nil {
		access = access && op.OptionValue == "true"
	}
	if access && strings.ToLower(doc.Identify) == "summary.md" && (strings.Contains(markdown, "<spider></spider>") || strings.Contains(doc.Markdown, "<spider/>")) {
		//如果标识是summary.md，并且带有bookstack的标签，则表示更新目录
		isSummary = true
		//要清除，避免每次保存的时候都要重新排序
		replaces := []string{"<spider></spider>", "<spider/>"}
		for _, r := range replaces {
			markdown = strings.Replace(markdown, r, "", -1)
		}
		content, markdown, _ = new(models.Document).BookStackCrawl(content, markdown, bookId, this.Member.MemberId)
	}

	content = this.replaceLinks(identify, content, isSummary)
	auto := strings.Contains(markdown, "<bookstack-auto></bookstack-auto>") || strings.Contains(doc.Markdown, "<bookstack-auto/>")
	if isSummary || auto {
		//自动生成文档内容
		var imd, icont string
		newDoc := models.NewDocument()
		if strings.ToLower(doc.Identify) == "summary.md" {
			if auto {
				icont, _ = newDoc.CreateDocumentTreeForHtml(doc.BookId, doc.DocumentId)
				imd = html2md.Convert(icont)
			} else {
				imd = html2md.Convert(content)
				markdown = imd
			}
		} else {
			imd, icont = newDoc.BookStackAuto(bookId, docId)
		}
		markdown = strings.Replace(markdown, "<bookstack-auto></bookstack-auto>", imd, -1)
		content = strings.Replace(content, "<bookstack-auto></bookstack-auto>", icont, -1)
		markdown = strings.Replace(markdown, "(/read/"+identify+"/", "($", -1)
		isAuto = true && !isSummary
	}

	var ds = models.DocumentStore{}
	var actionName string

	// 替换掉<git></git>标签内容
	if markdown == "" && content != "" {
		ds.Markdown = content
	} else {
		ds.Markdown = markdown
	}

	ds.Markdown, actionName = parseGitCommit(ds.Markdown)
	ds.Content, _ = parseGitCommit(content)

	if actionName == "" {
		actionName = "--"
	} else {
		isAuto = true
	}

	doc.Version = time.Now().Unix()
	if docId, err := doc.InsertOrUpdate(); err != nil {
		beego.Error("InsertOrUpdate => ", err)
		this.JsonResult(6006, "保存失败")
	} else {
		ds.DocumentId = int(docId)
		if err := ModelStore.InsertOrUpdate(ds, "markdown", "content"); err != nil {
			beego.Error(err)
		}
	}

	//如果启用了文档历史，则添加历史文档
	if this.EnableDocumentHistory > 0 {
		if len(strings.TrimSpace(ds.Markdown)) > 0 { //空内容不存储版本
			now := time.Now().Unix()
			history := models.NewDocumentHistory()
			history.DocumentId = docId
			history.DocumentName = doc.DocumentName
			history.ModifyAt = int(now)
			history.MemberId = this.Member.MemberId
			history.ParentId = doc.ParentId
			history.Version = now
			history.Action = "modify"
			history.ActionName = actionName
			_, err = history.InsertOrUpdate()
			if err != nil {
				beego.Error("DocumentHistory InsertOrUpdate => ", err)
			} else {
				vc := models.NewVersionControl(docId, history.Version)
				vc.SaveVersion(ds.Content, ds.Markdown)
				history.DeleteByLimit(docId, this.EnableDocumentHistory)
			}
		}

	}

	if isAuto {
		errMsg = "auto"
	} else if isSummary {
		errMsg = "true"
	}

	doc.Release = ""
	//注意：如果errMsg的值是true，则表示更新了目录排序，需要刷新，否则不刷新
	this.JsonResult(0, errMsg, doc)

}

// 导出文件
func (this *DocumentController) ExportOld() {
	wecode := strings.TrimSpace(this.GetString("wecode"))
	if wecode == "" && (this.Member == nil || this.Member.MemberId == 0) {
		if tips, ok := this.Option["DOWNLOAD_LIMIT"]; ok {
			tips = strings.TrimSpace(tips)
			if len(tips) > 0 {
				this.JsonResult(1, tips)
			}
		}
	}

	identify := this.Ctx.Input.Param(":key")
	ext := strings.ToLower(this.GetString("output"))
	switch ext {
	case "pdf", "epub", "mobi":
		ext = "." + ext
	default:
		ext = ".pdf"
	}
	if identify == "" {
		this.JsonResult(1, "下载失败，无法识别您要下载的文档")
	}
	book, err := new(models.Book).FindByIdentify(identify)
	if err != nil {
		beego.Error(err.Error())
		this.JsonResult(1, "下载失败，您要下载的书籍当前并未生成可下载的电子书。")
	}
	if book.PrivatelyOwned == 1 && this.Member.MemberId != book.MemberId {
		this.JsonResult(1, "私有文档，只有文档创建人可导出")
	}
	//查询文档是否存在
	obj := fmt.Sprintf("projects/%v/books/%v%v", book.Identify, book.GenerateTime.Unix(), ext)
	link := ""
	switch utils.StoreType {
	case utils.StoreOss:
		if err := store.ModelStoreOss.IsObjectExist(obj); err != nil {
			beego.Error(err, obj)
			this.JsonResult(1, "下载失败，您要下载的书籍当前并未生成可下载的电子书。")
		}
		link = this.OssDomain + "/" + obj
	case utils.StoreLocal:
		obj = "uploads/" + obj
		if err := store.ModelStoreLocal.IsObjectExist(obj); err != nil {
			beego.Error(err, obj)
			this.JsonResult(1, "下载失败，您要下载的书籍当前并未生成可下载的电子书。")
		}
		link = "/" + obj
	}
	if link != "" {
		// 查询是否可以下载
		counter := models.NewDownloadCounter()
		_, err := counter.DoesICanDownload(this.Member.MemberId, wecode)
		if err != nil {
			this.JsonResult(1, err.Error())
		}
		if wecode == "" {
			counter.Increase(this.Member.MemberId)
		}
		this.JsonResult(0, "获取文档下载链接成功", map[string]interface{}{"url": link})
	}
	this.JsonResult(1, "下载失败，您要下载的书籍当前并未生成可下载的电子书。")
}

// 导出文件
func (this *DocumentController) Export() {
	wecode := strings.TrimSpace(this.GetString("wecode"))
	if wecode == "" && (this.Member == nil || this.Member.MemberId == 0) {
		if tips, ok := this.Option["DOWNLOAD_LIMIT"]; ok {
			tips = strings.TrimSpace(tips)
			if len(tips) > 0 {
				this.JsonResult(1, tips)
			}
		}
	}

	identify := this.Ctx.Input.Param(":key")
	ext := strings.ToLower(this.GetString("output"))
	switch ext {
	case "pdf", "epub", "mobi":
		ext = "." + ext
	default:
		ext = ".pdf"
	}
	if identify == "" {
		this.JsonResult(1, "下载失败，无法识别您要下载的电子书")
	}
	book, err := new(models.Book).FindByIdentify(identify)
	if err != nil {
		beego.Error(err.Error())
		this.JsonResult(1, "下载失败，无法识别您要下载的电子书")
	}
	if book.PrivatelyOwned == 1 && this.Member.MemberId != book.MemberId {
		this.JsonResult(1, "私有书籍，只有书籍创建人可导出电子书")
	}
	ebook := models.NewEbook().Get2Download(book.BookId, ext)
	if ebook.Id == 0 || ebook.Status != models.EBookStatusSuccess || ebook.Path == "" {
		this.JsonResult(1, "下载失败，您要下载的书籍当前并未生成电子书。")
	}

	//查询文档是否存在
	obj := strings.TrimLeft(ebook.Path, "./")
	link := ""
	switch utils.StoreType {
	case utils.StoreOss:
		if err := store.ModelStoreOss.IsObjectExist(obj); err != nil {
			beego.Error(err, obj)
			this.JsonResult(1, "下载失败，您要下载的书籍当前并未生成电子书。")
		}
		link = this.OssDomain + "/" + obj
	case utils.StoreLocal:
		if err := store.ModelStoreLocal.IsObjectExist(obj); err != nil {
			beego.Error(err, obj)
			this.JsonResult(1, "下载失败，您要下载的书籍当前并未生成电子书。")
		}
		link = "/" + obj + fmt.Sprintf("?attachment=%s%s", url.QueryEscape(ebook.Title), ebook.Ext)
	}
	if link != "" {
		// 查询是否可以下载
		counter := models.NewDownloadCounter()
		_, err := counter.DoesICanDownload(this.Member.MemberId, wecode)
		if err != nil {
			this.JsonResult(1, err.Error())
		}
		if wecode == "" {
			counter.Increase(this.Member.MemberId)
		}
		this.JsonResult(0, "获取电子书下载链接成功", map[string]interface{}{"url": link})
	}
	this.JsonResult(1, "下载失败，您要下载的书籍当前并未生成电子书。")
}

//生成书籍访问的二维码.

func (this *DocumentController) QrCode() {
	this.Prepare()
	identify := this.GetString(":key")

	book, err := models.NewBook().FindByIdentify(identify)

	if err != nil || book.BookId <= 0 {
		this.Abort("404")
	}

	uri := this.BaseUrl() + beego.URLFor("DocumentController.Index", ":key", identify)
	code, err := qr.Encode(uri, qr.L, qr.Unicode)
	if err != nil {
		beego.Error(err)
		this.Abort("404")
	}
	code, err = barcode.Scale(code, 150, 150)

	if err != nil {
		beego.Error(err)
		this.Abort("404")
	}
	this.Ctx.ResponseWriter.Header().Set("Content-Type", "image/png")

	//imgpath := filepath.Join("cache","qrcode",identify + ".png")

	err = png.Encode(this.Ctx.ResponseWriter, code)
	if err != nil {
		beego.Error(err)
		this.Abort("404")
	}
}

// 书籍内搜索.
func (this *DocumentController) Search() {
	identify := this.Ctx.Input.Param(":key")
	token := this.GetString("token")
	keyword := strings.TrimSpace(this.GetString("keyword"))

	if identify == "" {
		this.JsonResult(6001, "参数错误")
	}
	if !this.EnableAnonymous && this.Member == nil {
		this.Redirect(beego.URLFor("AccountController.Login"), 302)
		return
	}
	bookResult := isReadable(identify, token, this)

	client := models.NewElasticSearchClient()
	if client.On { // 全文搜索
		result, err := client.Search(keyword, 1, 10000, true, bookResult.BookId)
		if err != nil {
			beego.Error(err)
			this.JsonResult(6002, "搜索结果错误")
		}

		var ids []int
		for _, item := range result.Hits.Hits {
			ids = append(ids, item.Source.Id)
		}
		docs, err := models.NewDocumentSearchResult().GetDocsById(ids, true)
		if err != nil {
			beego.Error(err)
		}

		// 如果全文搜索查询不到结果，用 MySQL like 再查询一次
		if len(docs) == 0 {
			if docsMySQL, _, err := models.NewDocumentSearchResult().SearchDocument(keyword, bookResult.BookId, 1, 10000); err != nil {
				beego.Error(err)
				this.JsonResult(6002, "搜索结果错误")
			} else {
				this.JsonResult(0, client.SegWords(keyword), docsMySQL)
			}
		} else {
			this.JsonResult(0, client.SegWords(keyword), docs)
		}

	} else {
		docs, _, err := models.NewDocumentSearchResult().SearchDocument(keyword, bookResult.BookId, 1, 10000)
		if err != nil {
			beego.Error(err)
			this.JsonResult(6002, "搜索结果错误")
		}
		this.JsonResult(0, keyword, docs)
	}
}

// 文档历史列表.
func (this *DocumentController) History() {

	this.TplName = "document/history.html"

	identify := this.GetString("identify")
	docId, _ := this.GetInt("doc_id", 0)
	pageIndex, _ := this.GetInt("page", 1)

	bookId := 0
	//如果是超级管理员则忽略权限判断
	if this.Member.IsAdministrator() {
		book, err := models.NewBook().FindByFieldFirst("identify", identify)
		if err != nil {
			beego.Error("FindByIdentify => ", err)
			this.Data["ErrorMessage"] = "书籍不存在或权限不足"
			return
		}
		bookId = book.BookId
		this.Data["Model"] = book
	} else {
		bookResult, err := models.NewBookResult().FindByIdentify(identify, this.Member.MemberId)

		if err != nil || bookResult.RoleId == conf.BookObserver {
			beego.Error("FindByIdentify => ", err)
			this.Data["ErrorMessage"] = "书籍不存在或权限不足"
			return
		}
		bookId = bookResult.BookId
		this.Data["Model"] = bookResult
	}

	if docId <= 0 {
		this.Data["ErrorMessage"] = "参数错误"
		return
	}

	doc, err := models.NewDocument().Find(docId)

	if err != nil {
		beego.Error("Delete => ", err)
		this.Data["ErrorMessage"] = "获取历史失败"
		return
	}
	//如果文档所属书籍错误
	if doc.BookId != bookId {
		this.Data["ErrorMessage"] = "参数错误"
		return
	}

	histories, totalCount, err := models.NewDocumentHistory().FindToPager(docId, pageIndex, conf.PageSize)
	if err != nil {
		beego.Error("FindToPager => ", err)
		this.Data["ErrorMessage"] = "获取历史失败"
		return
	}

	this.Data["List"] = histories
	this.Data["PageHtml"] = ""
	this.Data["Document"] = doc

	if totalCount > 0 {
		html := utils.GetPagerHtml(this.Ctx.Request.RequestURI, pageIndex, conf.PageSize, totalCount)
		this.Data["PageHtml"] = html
	}
}

func (this *DocumentController) DeleteHistory() {

	this.TplName = "document/history.html"

	identify := this.GetString("identify")
	docId, _ := this.GetInt("doc_id", 0)
	historyId, _ := this.GetInt("history_id", 0)

	if historyId <= 0 {
		this.JsonResult(6001, "参数错误")
	}
	bookId := 0
	//如果是超级管理员则忽略权限判断
	if this.Member.IsAdministrator() {
		book, err := models.NewBook().FindByFieldFirst("identify", identify)
		if err != nil {
			beego.Error("FindByIdentify => ", err)
			this.JsonResult(6002, "书籍不存在或权限不足")
		}
		bookId = book.BookId
	} else {
		bookResult, err := models.NewBookResult().FindByIdentify(identify, this.Member.MemberId)

		if err != nil || bookResult.RoleId == conf.BookObserver {
			beego.Error("FindByIdentify => ", err)
			this.JsonResult(6002, "书籍不存在或权限不足")
		}
		bookId = bookResult.BookId
	}

	if docId <= 0 {
		this.JsonResult(6001, "参数错误")
	}

	doc, err := models.NewDocument().Find(docId)
	if err != nil {
		beego.Error("Delete => ", err)
		this.JsonResult(6001, "获取历史失败")
	}

	//如果文档所属书籍错误
	if doc.BookId != bookId {
		this.JsonResult(6001, "参数错误")
	}

	//err = models.NewDocumentHistory().Delete(history_id, doc_id)
	err = models.NewDocumentHistory().DeleteByHistoryId(historyId)
	if err != nil {
		beego.Error(err)
		this.JsonResult(6002, "删除失败")
	}
	this.JsonResult(0, "ok")
}

func (this *DocumentController) RestoreHistory() {

	this.TplName = "document/history.html"

	identify := this.GetString("identify")
	docId, _ := this.GetInt("doc_id", 0)

	historyId, _ := this.GetInt("history_id", 0)
	if historyId <= 0 {
		this.JsonResult(6001, "参数错误")
	}

	bookId := 0
	//如果是超级管理员则忽略权限判断
	if this.Member.IsAdministrator() {
		book, err := models.NewBook().FindByFieldFirst("identify", identify)
		if err != nil {
			beego.Error("FindByIdentify => ", err)
			this.JsonResult(6002, "书籍不存在或权限不足")
		}
		bookId = book.BookId
	} else {
		bookResult, err := models.NewBookResult().FindByIdentify(identify, this.Member.MemberId)
		if err != nil || bookResult.RoleId == conf.BookObserver {
			beego.Error("FindByIdentify => ", err)
			this.JsonResult(6002, "书籍不存在或权限不足")
		}
		bookId = bookResult.BookId
	}

	if docId <= 0 {
		this.JsonResult(6001, "参数错误")
	}

	doc, err := models.NewDocument().Find(docId)

	if err != nil {
		beego.Error("Delete => ", err)
		this.JsonResult(6001, "获取历史失败")
	}
	//如果文档所属书籍错误
	if doc.BookId != bookId {
		this.JsonResult(6001, "参数错误")
	}

	err = models.NewDocumentHistory().Restore(historyId, docId, this.Member.MemberId)
	if err != nil {
		beego.Error(err)
		this.JsonResult(6002, "删除失败")
	}
	this.JsonResult(0, "ok", doc)
}

func (this *DocumentController) Compare() {
	this.TplName = "document/compare.html"
	historyId, _ := strconv.Atoi(this.Ctx.Input.Param(":id"))
	identify := this.Ctx.Input.Param(":key")

	bookId := 0
	//如果是超级管理员则忽略权限判断
	if this.Member.IsAdministrator() {
		book, err := models.NewBook().FindByFieldFirst("identify", identify)
		if err != nil {
			beego.Error("DocumentController.Compare => ", err)
			this.Abort("404")
			return
		}
		bookId = book.BookId
		this.Data["Model"] = book
	} else {
		bookResult, err := models.NewBookResult().FindByIdentify(identify, this.Member.MemberId)

		if err != nil || bookResult.RoleId == conf.BookObserver {
			beego.Error("FindByIdentify => ", err)
			this.Abort("404")
			return
		}
		bookId = bookResult.BookId
		this.Data["Model"] = bookResult
	}

	if historyId <= 0 {
		this.JsonResult(60002, "参数错误")
	}

	history, err := models.NewDocumentHistory().Find(historyId)
	if err != nil {
		beego.Error("DocumentController.Compare => ", err)
		this.ShowErrorPage(60003, err.Error())
	}
	doc, err := models.NewDocument().Find(history.DocumentId)

	if doc.BookId != bookId {
		this.ShowErrorPage(60002, "参数错误")
	}
	vc := models.NewVersionControl(doc.DocumentId, history.Version)
	this.Data["HistoryId"] = historyId
	this.Data["DocumentId"] = doc.DocumentId
	ModelStore := new(models.DocumentStore)
	this.Data["HistoryContent"] = vc.GetVersionContent(false)
	this.Data["Content"] = ModelStore.GetFiledById(doc.DocumentId, "markdown")
}

// 递归生成文档序列数组.
func RecursiveFun(parentId int, prefix, dpath string, this *DocumentController, book *models.BookResult, docs []*models.Document, paths *list.List) {
	for _, item := range docs {
		if item.ParentId == parentId {
			name := prefix + strconv.Itoa(item.ParentId) + strconv.Itoa(item.OrderSort) + strconv.Itoa(item.DocumentId)
			fpath := dpath + "/" + name + ".html"
			paths.PushBack(fpath)

			f, err := os.OpenFile(fpath, os.O_CREATE|os.O_RDWR, 0777)

			if err != nil {
				beego.Error(err)
				this.Abort("404")
			}

			html, err := this.ExecuteViewPathTemplate("document/export.html", map[string]interface{}{"Model": book, "Lists": item, "BaseUrl": this.BaseUrl()})
			if err != nil {
				f.Close()
				beego.Error(err)
				this.Abort("404")
			}

			buf := bytes.NewReader([]byte(html))
			doc, _ := goquery.NewDocumentFromReader(buf)
			doc.Find("img").Each(func(i int, contentSelection *goquery.Selection) {
				if src, ok := contentSelection.Attr("src"); ok && strings.HasPrefix(src, "/uploads/") {
					contentSelection.SetAttr("src", this.BaseUrl()+src)
				}
			})
			html, err = doc.Html()

			if err != nil {
				f.Close()
				beego.Error(err)
				this.Abort("404")
			}
			//html = strings.Replace(html, "<img src=\"/uploads", "<img src=\""+this.BaseUrl()+"/uploads", -1)

			f.WriteString(html)
			f.Close()

			for _, sub := range docs {
				if sub.ParentId == item.DocumentId {
					RecursiveFun(item.DocumentId, name, dpath, this, book, docs, paths)
					break
				}
			}
		}
	}
}

//
