package controllers

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/TruthHun/BookStack/graphics"
	"github.com/russross/blackfriday"

	"github.com/TruthHun/BookStack/commands"
	"github.com/TruthHun/BookStack/conf"
	"github.com/TruthHun/BookStack/models"
	"github.com/TruthHun/BookStack/utils"
	"github.com/TruthHun/gotil/filetil"
	"github.com/TruthHun/gotil/mdtil"
	"github.com/TruthHun/gotil/util"
	"github.com/TruthHun/gotil/ziptil"
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/logs"
	"github.com/astaxie/beego/orm"
)

type BookController struct {
	BaseController
}

func (this *BookController) Index() {
	this.Data["SettingBook"] = true
	this.TplName = "book/index.html"
	private, _ := this.GetInt("private", 1) //是否是私有文档
	this.Data["Private"] = private
	pageIndex, _ := this.GetInt("page", 1)
	books, totalCount, err := models.NewBook().FindToPager(pageIndex, conf.PageSize, this.Member.MemberId, private)
	if err != nil {
		logs.Error("BookController.Index => ", err)
		this.Abort("500")
	}
	if totalCount > 0 {
		//this.Data["PageHtml"] = utils.GetPagerHtml(this.Ctx.Request.RequestURI, pageIndex, conf.PageSize, totalCount)
		this.Data["PageHtml"] = utils.NewPaginations(conf.RollPage, totalCount, conf.PageSize, pageIndex, beego.URLFor("BookController.Index"), fmt.Sprintf("&private=%v", private))
	} else {
		this.Data["PageHtml"] = ""
	}
	//处理封面图片
	for idx, book := range books {
		book.Cover = utils.ShowImg(book.Cover, "cover")
		books[idx] = book
	}
	b, err := json.Marshal(books)
	if err != nil || len(books) <= 0 {
		this.Data["Result"] = template.JS("[]")
	} else {
		this.Data["Result"] = template.JS(string(b))
	}
}

//收藏书籍
func (this *BookController) Star() {
	if uid := this.BaseController.Member.MemberId; uid > 0 {
		if id, _ := this.GetInt(":id"); id > 0 {
			cancel, err := new(models.Star).Star(uid, id)
			data := map[string]bool{"IsCancel": cancel}
			if err != nil {
				beego.Error(err.Error())
				if cancel {
					this.JsonResult(1, "取消收藏失败", data)
				} else {
					this.JsonResult(1, "添加收藏失败", data)
				}
			} else {
				if cancel {
					this.JsonResult(0, "取消收藏成功", data)
				} else {
					this.JsonResult(0, "添加收藏成功", data)
				}
			}
		} else {
			this.JsonResult(1, "收藏失败，项目不存在")
		}
	} else {
		this.JsonResult(1, "收藏失败，请先登录")
	}

}

// Dashboard 项目概要 .
func (this *BookController) Dashboard() {
	this.Prepare()
	this.TplName = "book/dashboard.html"

	key := this.Ctx.Input.Param(":key")

	if key == "" {
		this.Abort("404")
	}

	book, err := models.NewBookResult().FindByIdentify(key, this.Member.MemberId)
	if err != nil {
		if err == models.ErrPermissionDenied {
			this.Abort("403")
		}
		beego.Error(err)
		this.Abort("500")
	}

	this.Data["Model"] = *book
}

// Setting 项目设置 .
func (this *BookController) Setting() {
	this.TplName = "book/setting.html"

	key := this.Ctx.Input.Param(":key")

	if key == "" {
		this.Abort("404")
	}

	book, err := models.NewBookResult().FindByIdentify(key, this.Member.MemberId)
	if err != nil {
		if err == orm.ErrNoRows {
			this.Abort("404")
		}
		if err == models.ErrPermissionDenied {
			this.Abort("403")
		}
		beego.Error(err.Error())
		this.Abort("404")
	}
	//如果不是创始人也不是管理员则不能操作
	if book.RoleId != conf.BookFounder && book.RoleId != conf.BookAdmin {
		this.Abort("403")
	}
	if book.PrivateToken != "" {
		book.PrivateToken = this.BaseUrl() + beego.URLFor("DocumentController.Index", ":key", book.Identify, "token", book.PrivateToken)
	}

	//查询当前书籍的分类id
	if selectedCates, rows, _ := new(models.BookCategory).GetByBookId(book.BookId); rows > 0 {
		var maps = make(map[int]bool)
		for _, cate := range selectedCates {
			maps[cate.Id] = true
		}
		this.Data["Maps"] = maps
	}

	this.Data["Cates"], _ = new(models.Category).GetCates(-1, 1)
	this.Data["Model"] = book

}

//保存项目信息
func (this *BookController) SaveBook() {
	bookResult, err := this.IsPermission()

	if err != nil {
		this.JsonResult(6001, err.Error())
	}
	book, err := models.NewBook().Find(bookResult.BookId)

	if err != nil {
		logs.Error("SaveBook => ", err)
		this.JsonResult(6002, err.Error())
	}

	book_name := strings.TrimSpace(this.GetString("book_name"))
	description := strings.TrimSpace(this.GetString("description", ""))
	comment_status := this.GetString("comment_status")
	tag := strings.TrimSpace(this.GetString("label"))
	editor := strings.TrimSpace(this.GetString("editor"))

	if strings.Count(description, "") > 500 {
		this.JsonResult(6004, "项目描述不能大于500字")
	}
	if comment_status != "open" && comment_status != "closed" && comment_status != "group_only" && comment_status != "registered_only" {
		comment_status = "closed"
	}
	if tag != "" {
		tags := strings.Split(tag, ",")
		if len(tags) > 10 {
			this.JsonResult(6005, "最多允许添加10个标签")
		}
	}
	if editor != "markdown" && editor != "html" {
		editor = "markdown"
	}

	book.BookName = book_name
	book.Description = description
	book.CommentStatus = comment_status
	book.Label = tag
	book.Editor = editor

	if err := book.Update(); err != nil {
		this.JsonResult(6006, "保存失败")
	}
	bookResult.BookName = book_name
	bookResult.Description = description
	bookResult.CommentStatus = comment_status
	bookResult.Label = tag
	//更新书籍分类
	if cids, ok := this.Ctx.Request.Form["cid"]; ok {
		new(models.BookCategory).SetBookCates(book.BookId, cids)
	}
	this.JsonResult(0, "ok", bookResult)
}

//设置项目私有状态.
func (this *BookController) PrivatelyOwned() {

	status := this.GetString("status")
	if status != "open" && status != "close" {
		this.JsonResult(6003, "参数错误")
	}
	state := 0
	if status == "open" {
		state = 0
	} else {
		state = 1
	}

	bookResult, err := this.IsPermission()

	if err != nil {
		this.JsonResult(6001, err.Error())
	}
	//只有创始人才能变更私有状态
	if bookResult.RoleId != conf.BookFounder {
		this.JsonResult(6002, "权限不足")
	}

	if _, err = orm.NewOrm().QueryTable("md_books").Filter("book_id", bookResult.BookId).Update(orm.Params{
		"privately_owned": state,
	}); err != nil {
		logs.Error("PrivatelyOwned => ", err)
		this.JsonResult(6004, "保存失败")
	}
	this.JsonResult(0, "ok")
}

// Transfer 转让项目.
func (this *BookController) Transfer() {
	this.Prepare()
	account := this.GetString("account")

	if account == "" {
		this.JsonResult(6004, "接受者账号不能为空")
	}
	member, err := models.NewMember().FindByAccount(account)

	if err != nil {
		logs.Error("FindByAccount => ", err)
		this.JsonResult(6005, "接受用户不存在")
	}
	if member.Status != 0 {
		this.JsonResult(6006, "接受用户已被禁用")
	}
	if member.MemberId == this.Member.MemberId {
		this.JsonResult(6007, "不能转让给自己")
	}
	bookResult, err := this.IsPermission()

	if err != nil {
		this.JsonResult(6001, err.Error())
	}

	err = models.NewRelationship().Transfer(bookResult.BookId, this.Member.MemberId, member.MemberId)

	if err != nil {
		logs.Error("Transfer => ", err)
		this.JsonResult(6008, err.Error())
	}
	this.JsonResult(0, "ok")
}

//上传项目封面.
func (this *BookController) UploadCover() {

	bookResult, err := this.IsPermission()

	if err != nil {
		this.JsonResult(6001, err.Error())
	}
	book, err := models.NewBook().Find(bookResult.BookId)

	if err != nil {
		logs.Error("SaveBook => ", err)
		this.JsonResult(6002, err.Error())
	}

	file, moreFile, err := this.GetFile("image-file")
	if err != nil {
		logs.Error("", err.Error())
		this.JsonResult(500, "读取文件异常")
	}
	defer file.Close()

	ext := filepath.Ext(moreFile.Filename)

	if !strings.EqualFold(ext, ".png") && !strings.EqualFold(ext, ".jpg") && !strings.EqualFold(ext, ".gif") && !strings.EqualFold(ext, ".jpeg") {
		this.JsonResult(500, "不支持的图片格式")
	}
	//
	x1, _ := strconv.ParseFloat(this.GetString("x"), 10)
	y1, _ := strconv.ParseFloat(this.GetString("y"), 10)
	w1, _ := strconv.ParseFloat(this.GetString("width"), 10)
	h1, _ := strconv.ParseFloat(this.GetString("height"), 10)

	x := int(x1)
	y := int(y1)
	width := int(w1)
	height := int(h1)

	fileName := strconv.FormatInt(time.Now().UnixNano(), 16)

	filePath := filepath.Join("uploads", time.Now().Format("200601"), fileName+ext)

	path := filepath.Dir(filePath)

	os.MkdirAll(path, os.ModePerm)

	err = this.SaveToFile("image-file", filePath)

	if err != nil {
		logs.Error("", err)
		this.JsonResult(500, "图片保存失败")
	}
	if utils.StoreType != utils.StoreLocal {
		defer func(filePath string) {
			os.Remove(filePath)
		}(filePath)
	}

	//剪切图片
	subImg, err := graphics.ImageCopyFromFile(filePath, x, y, width, height)

	if err != nil {
		logs.Error("graphics.ImageCopyFromFile => ", err)
		this.JsonResult(500, "图片剪切")
	}

	filePath = filepath.Join(commands.WorkingDirectory, "uploads", time.Now().Format("200601"), fileName+ext)

	//生成缩略图并保存到磁盘
	err = graphics.ImageResizeSaveFile(subImg, 175, 230, filePath)

	if err != nil {
		logs.Error("ImageResizeSaveFile => ", err.Error())
		this.JsonResult(500, "保存图片失败")
	}

	url := "/" + strings.Replace(strings.TrimPrefix(filePath, commands.WorkingDirectory), "\\", "/", -1)

	if strings.HasPrefix(url, "//") {
		url = string(url[1:])
	}

	old_cover := book.Cover
	osspath := fmt.Sprintf("projects/%v/%v", book.Identify, strings.TrimLeft(url, "./"))
	book.Cover = "/" + osspath
	if utils.StoreType == utils.StoreLocal {
		book.Cover = url
	}

	if err := book.Update(); err != nil {
		this.JsonResult(6001, "保存图片失败")
	}
	//如果原封面不是默认封面则删除
	if old_cover != conf.GetDefaultCover() {
		os.Remove("." + old_cover)
		switch utils.StoreType {
		case utils.StoreOss:
			models.ModelStoreOss.DelFromOss(old_cover) //从OSS执行一次删除
		case utils.StoreLocal:
			models.ModelStoreLocal.DelFiles(old_cover) //从本地执行一次删除
		}

	}
	switch utils.StoreType {
	case utils.StoreOss: //oss
		if err := models.ModelStoreOss.MoveToOss("."+url, osspath, true, false); err != nil {
			beego.Error(err.Error())
		} else {
			url = strings.TrimRight(beego.AppConfig.String("oss::Domain"), "/ ") + "/" + osspath + "/cover"
		}
	case utils.StoreLocal:
		save := book.Cover
		if err := models.ModelStoreLocal.MoveToStore("."+url, save); err != nil {
			beego.Error(err.Error())
		} else {
			url = book.Cover
		}
	}

	this.JsonResult(0, "ok", url)
}

// Users 用户列表.
func (this *BookController) Users() {
	this.TplName = "book/users.html"

	key := this.Ctx.Input.Param(":key")
	pageIndex, _ := this.GetInt("page", 1)

	if key == "" {
		this.Abort("404")
	}

	book, err := models.NewBookResult().FindByIdentify(key, this.Member.MemberId)
	if err != nil {
		if err == models.ErrPermissionDenied {
			this.Abort("403")
		}
		this.Abort("500")
	}

	this.Data["Model"] = *book

	members, totalCount, err := models.NewMemberRelationshipResult().FindForUsersByBookId(book.BookId, pageIndex, 15)
	for idx, member := range members {
		member.Avatar = utils.ShowImg(member.Avatar, "avatar")
		members[idx] = member
	}
	if totalCount > 0 {
		html := utils.GetPagerHtml(this.Ctx.Request.RequestURI, pageIndex, 10, totalCount)
		this.Data["PageHtml"] = html
	} else {
		this.Data["PageHtml"] = ""
	}
	b, err := json.Marshal(members)

	if err != nil {
		this.Data["Result"] = template.JS("[]")
	} else {
		this.Data["Result"] = template.JS(string(b))
	}
}

// Create 创建项目.
func (this *BookController) Create() {

	if this.Ctx.Input.IsPost() {
		book_name := strings.TrimSpace(this.GetString("book_name", ""))
		identify := strings.TrimSpace(this.GetString("identify", ""))
		description := strings.TrimSpace(this.GetString("description", ""))
		privately_owned, _ := strconv.Atoi(this.GetString("privately_owned"))
		comment_status := this.GetString("comment_status")

		if book_name == "" {
			this.JsonResult(6001, "项目名称不能为空")
		}
		if identify == "" {
			this.JsonResult(6002, "项目标识不能为空")
		}
		if ok, err := regexp.MatchString(`[a-zA-Z0-9_\-]*$`, identify); !ok || err != nil {
			this.JsonResult(6003, "项目标识只能包含字母、数字，以及“-”和“_”符号头，且不能是纯数字")
		}
		if num, _ := strconv.Atoi(identify); strconv.Itoa(num) == identify {
			this.JsonResult(6003, "项目标识只能包含字母、数字，以及“-”和“_”符号头，且不能是纯数字")
		}
		if strings.Count(identify, "") > 50 {
			this.JsonResult(6004, "文档标识不能超过50字")
		}
		if strings.Count(description, "") > 500 {
			this.JsonResult(6004, "项目描述不能大于500字")
		}
		if privately_owned != 0 && privately_owned != 1 {
			privately_owned = 1
		}
		if comment_status != "open" && comment_status != "closed" && comment_status != "group_only" && comment_status != "registered_only" {
			comment_status = "closed"
		}

		book := models.NewBook()

		if books, _ := book.FindByField("identify", identify); len(books) > 0 {
			this.JsonResult(6006, "项目标识已存在")
		}

		book.Label = utils.SegWord(book_name)
		book.BookName = book_name
		book.Description = description
		book.CommentCount = 0
		book.PrivatelyOwned = privately_owned
		book.CommentStatus = comment_status
		book.Identify = identify
		book.DocCount = 0
		book.MemberId = this.Member.MemberId
		book.CommentCount = 0
		book.Version = time.Now().Unix()
		book.Cover = conf.GetDefaultCover()
		book.Editor = "markdown"
		book.Theme = "default"
		book.Score = 40 //默认评分，40即表示4星
		//设置默认时间，因为beego的orm好像无法设置datetime的默认值
		defaultTime, _ := time.Parse("2006-01-02 15:04:05", "2006-01-02 15:04:05")
		book.LastClickGenerate = defaultTime
		book.GenerateTime, _ = time.Parse("2006-01-02 15:04:05", "2000-01-02 15:04:05") //默认生成文档的时间
		book.ReleaseTime = defaultTime

		err := book.Insert()

		if err != nil {
			logs.Error("Insert => ", err)
			this.JsonResult(6005, "保存项目失败")
		}
		bookResult, err := models.NewBookResult().FindByIdentify(book.Identify, this.Member.MemberId)

		if err != nil {
			beego.Error(err)
		}

		this.JsonResult(0, "ok", bookResult)
	}
	this.JsonResult(6001, "error")
}

// CreateToken 创建访问来令牌.
func (this *BookController) CreateToken() {

	action := this.GetString("action")

	bookResult, err := this.IsPermission()

	if err != nil {
		if err == models.ErrPermissionDenied {
			this.JsonResult(403, "权限不足")
		}
		if err == orm.ErrNoRows {
			this.JsonResult(404, "项目不存在")
		}
		logs.Error("生成阅读令牌失败 =>", err)
		this.JsonResult(6002, err.Error())
	}
	book := models.NewBook()

	if _, err := book.Find(bookResult.BookId); err != nil {
		this.JsonResult(6001, "项目不存在")
	}
	if action == "create" {
		if bookResult.PrivatelyOwned == 0 {
			this.JsonResult(6001, "公开项目不能创建阅读令牌")
		}

		book.PrivateToken = string(utils.Krand(conf.GetTokenSize(), utils.KC_RAND_KIND_ALL))
		if err := book.Update(); err != nil {
			logs.Error("生成阅读令牌失败 => ", err)
			this.JsonResult(6003, "生成阅读令牌失败")
		}
		this.JsonResult(0, "ok", this.BaseUrl()+beego.URLFor("DocumentController.Index", ":key", book.Identify, "token", book.PrivateToken))
	} else {
		book.PrivateToken = ""
		if err := book.Update(); err != nil {
			logs.Error("CreateToken => ", err)
			this.JsonResult(6004, "删除令牌失败")
		}
		this.JsonResult(0, "ok", "")
	}
}

// Delete 删除项目.
func (this *BookController) Delete() {

	bookResult, err := this.IsPermission()

	if err != nil {
		this.JsonResult(6001, err.Error())
	}

	if bookResult.RoleId != conf.BookFounder {
		this.JsonResult(6002, "只有创始人才能删除项目")
	}

	//用户密码
	pwd := this.GetString("password")

	if m, err := models.NewMember().Login(this.Member.Account, pwd); err != nil || m.MemberId == 0 {
		this.JsonResult(1, "项目删除失败，您的登录密码不正确")
	}

	err = models.NewBook().ThoroughDeleteBook(bookResult.BookId)

	if err == orm.ErrNoRows {
		this.JsonResult(6002, "项目不存在")
	}
	if err != nil {
		logs.Error("删除项目 => ", err)
		this.JsonResult(6003, "删除失败")
	}

	this.JsonResult(0, "ok")
}

//发布项目.
func (this *BookController) Release() {

	identify := this.GetString("identify")
	book_id := 0
	if this.Member.IsAdministrator() {
		book, err := models.NewBook().FindByFieldFirst("identify", identify)
		if err != nil {
			beego.Error(err)
		}
		book_id = book.BookId
	} else {
		book, err := models.NewBookResult().FindByIdentify(identify, this.Member.MemberId)

		if err != nil {
			if err == models.ErrPermissionDenied {
				this.JsonResult(6001, "权限不足")
			}
			if err == orm.ErrNoRows {
				this.JsonResult(6002, "项目不存在")
			}
			beego.Error(err)
			this.JsonResult(6003, "未知错误")
		}
		if book.RoleId != conf.BookAdmin && book.RoleId != conf.BookFounder && book.RoleId != conf.BookEditor {
			this.JsonResult(6003, "权限不足")
		}
		book_id = book.BookId
	}

	if exist := utils.BooksRelease.Exist(book_id); exist {
		this.JsonResult(1, "上次内容发布正在执行中，请稍后再操作")
	}

	go func(identify string) {
		models.NewDocument().ReleaseContent(book_id, this.BaseUrl())
	}(identify)

	this.JsonResult(0, "发布任务已推送到任务队列，稍后将在后台执行。")
}

//生成下载文档
//加锁，防止用户不停地点击生成下载文档造成服务器资源开销.
func (this *BookController) Generate() {
	identify := this.GetString(":key")
	book, err := models.NewBook().FindByIdentify(identify)

	//书籍正在生成离线文档
	if isGenerating := utils.BooksGenerate.Exist(book.BookId); isGenerating {
		this.JsonResult(1, "上一次下载文档生成任务正在后台执行，请您稍后再执行新的下载文档生成操作")
	}

	if err != nil || book.MemberId != this.Member.MemberId {
		beego.Error(err)
		this.JsonResult(1, "项目不存在；或您不是文档创始人，没有文档生成权限")
	}

	baseUrl := "http://localhost:" + beego.AppConfig.String("httpport")
	go new(models.Document).GenerateBook(book, baseUrl)

	this.JsonResult(0, "下载文档生成任务已交由后台执行，请您耐心等待。")
}

//文档排序.
func (this *BookController) SaveSort() {

	identify := this.Ctx.Input.Param(":key")
	if identify == "" {
		this.Abort("404")
	}

	book_id := 0
	if this.Member.IsAdministrator() {
		book, err := models.NewBook().FindByFieldFirst("identify", identify)
		if err != nil {

		}
		book_id = book.BookId
	} else {
		bookResult, err := models.NewBookResult().FindByIdentify(identify, this.Member.MemberId)
		if err != nil {
			beego.Error("DocumentController.Edit => ", err)

			this.Abort("403")
		}
		if bookResult.RoleId == conf.BookObserver {
			this.JsonResult(6002, "项目不存在或权限不足")
		}
		book_id = bookResult.BookId
	}

	content := this.Ctx.Input.RequestBody

	var docs []struct {
		Id     int `json:"id"`
		Sort   int `json:"sort"`
		Parent int `json:"parent"`
	}

	err := json.Unmarshal(content, &docs)

	if err != nil {
		beego.Error(err)
		this.JsonResult(6003, "数据错误")
	}
	qs := orm.NewOrm().QueryTable("md_documents").Filter("book_id", book_id)
	now := time.Now()
	for _, item := range docs {
		qs.Filter("document_id", item.Id).Update(orm.Params{
			"parent_id":   item.Parent,
			"order_sort":  item.Sort,
			"modify_time": now,
		})
	}
	this.JsonResult(0, "ok")
}

func (this *BookController) IsPermission() (*models.BookResult, error) {
	identify := this.GetString("identify")

	book, err := models.NewBookResult().FindByIdentify(identify, this.Member.MemberId)

	if err != nil {
		if err == models.ErrPermissionDenied {
			return book, errors.New("权限不足")
		}
		if err == orm.ErrNoRows {
			return book, errors.New("项目不存在")
		}
		return book, err
	}
	if book.RoleId != conf.BookAdmin && book.RoleId != conf.BookFounder {
		return book, errors.New("权限不足")
	}
	return book, nil
}

//从github等拉取下载markdown项目
func (this *BookController) DownloadProject() {
	//处理步骤
	//1、接受上传上来的zip文件，并存放到store/temp目录下
	//2、解压zip到当前目录，然后移除非图片文件
	//3、将文件夹移动到uploads目录下
	if _, err := this.IsPermission(); err != nil {
		this.JsonResult(1, err.Error())
	}

	//普通用户没有权限
	if this.Member.Role > 1 {
		this.JsonResult(1, "您没有操作权限")
	}

	identify := this.GetString("identify")
	book, _ := models.NewBookResult().FindByIdentify(identify, this.Member.MemberId)
	if book.BookId == 0 {
		this.JsonResult(1, "导入失败，只有项目创建人才有权限导入项目")
	}
	//GitHub项目链接
	link := this.GetString("link")
	if strings.ToLower(filepath.Ext(link)) != ".zip" {
		this.JsonResult(1, "只支持拉取zip压缩的markdown项目")
	}
	go func() {
		if file, err := util.CrawlFile(link, "store", 60); err != nil {
			beego.Error(err)
		} else {
			this.unzipToData(book.BookId, identify, file, filepath.Base(file))
		}
	}()
	this.JsonResult(0, "提交成功。下载任务已交由后台执行")
}

//上传项目
func (this *BookController) UploadProject() {
	//处理步骤
	//1、接受上传上来的zip文件，并存放到store/temp目录下
	//2、解压zip到当前目录，然后移除非图片文件
	//3、将文件夹移动到uploads目录下
	if _, err := this.IsPermission(); err != nil {
		this.JsonResult(1, err.Error())
	}

	//普通用户没法上传项目
	if this.Member.Role > 1 {
		this.JsonResult(1, "您没有操作权限")
	}

	identify := this.GetString("identify")

	book, _ := models.NewBookResult().FindByIdentify(identify, this.Member.MemberId)
	if book.BookId == 0 {
		this.JsonResult(1, "导入失败，只有项目创建人才有权限导入项目")
	}
	f, h, err := this.GetFile("zipfile")
	if err != nil {
		this.JsonResult(1, err.Error())
	}
	defer f.Close()
	if strings.ToLower(filepath.Ext(h.Filename)) != ".zip" {
		this.JsonResult(1, "请上传zip格式文件")
	}
	tmpfile := "store/" + identify + ".zip" //保存的文件名
	if err := this.SaveToFile("zipfile", tmpfile); err == nil {
		go this.unzipToData(book.BookId, identify, tmpfile, h.Filename)
	} else {
		beego.Error(err.Error())
	}
	this.JsonResult(0, "上传成功")
}

//将zip压缩文件解压并录入数据库
//@param            book_id             项目id(其实有想不标识了可以不要这个的，但是这里的项目标识只做目录)
//@param            identify            项目标识
//@param            zipfile             压缩文件
//@param            originFilename      上传文件的原始文件名
func (this *BookController) unzipToData(book_id int, identify, zipfile, originFilename string) {

	//说明：
	//OSS中的图片存储规则为projects/$identify/项目中图片原路径
	//本地存储规则为uploads/projects/$identify/项目中图片原路径

	projectRoot := "" //项目根目录

	//解压目录
	unzipPath := "store/" + identify

	//如果存在相同目录，则率先移除
	if err := os.RemoveAll(unzipPath); err != nil {
		beego.Error(err.Error())
	}
	os.MkdirAll(unzipPath, os.ModePerm)

	imgMap := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".bmp": true, ".svg": true, ".webp": true}

	defer func() {
		os.Remove(zipfile)      //最后删除上传的临时文件
		os.RemoveAll(unzipPath) //删除解压后的文件夹
	}()

	//注意：这里的prefix必须是判断是否是GitHub之前的prefix
	if err := ziptil.Unzip(zipfile, unzipPath); err != nil {
		beego.Error("解压失败", zipfile, err.Error())
	} else {

		//读取文件，把图片文档录入oss
		if files, err := filetil.ScanFiles(unzipPath); err == nil {
			projectRoot = this.getProjectRoot(files)
			this.replaceToAbs(projectRoot, identify)

			ModelStore := new(models.DocumentStore)
			//文档对应的标识
			for _, file := range files {
				if !file.IsDir {
					ext := strings.ToLower(filepath.Ext(file.Path))
					if ok, _ := imgMap[ext]; ok { //图片，录入oss
						switch utils.StoreType {
						case utils.StoreOss:
							if err := models.ModelStoreOss.MoveToOss(file.Path, "projects/"+identify+strings.TrimPrefix(file.Path, projectRoot), false, false); err != nil {
								beego.Error(err)
							}
						case utils.StoreLocal:
							if err := models.ModelStoreLocal.MoveToStore(file.Path, "uploads/projects/"+identify+strings.TrimPrefix(file.Path, projectRoot)); err != nil {
								beego.Error(err)
							}
						}

					} else if ext == ".md" || ext == ".markdown" { //markdown文档，提取文档内容，录入数据库
						doc := new(models.Document)
						if b, err := ioutil.ReadFile(file.Path); err == nil {
							mdcont := strings.TrimSpace(string(b))
							if !strings.HasPrefix(mdcont, "[TOC]") {
								mdcont = "[TOC]\r\n\r\n" + mdcont
							}
							htmlstr := mdtil.Md2html(mdcont)
							doc.DocumentName = utils.ParseTitleFromMdHtml(htmlstr)
							doc.BookId = book_id
							//文档标识
							doc.Identify = strings.Replace(strings.Trim(strings.TrimPrefix(file.Path, projectRoot), "/"), "/", "-", -1)
							doc.MemberId = this.Member.MemberId
							doc.OrderSort = 1
							if strings.HasSuffix(strings.ToLower(file.Name), "summary.md") {
								doc.OrderSort = 0
							}
							if doc_id, err := doc.InsertOrUpdate(); err == nil {
								if err := ModelStore.InsertOrUpdate(models.DocumentStore{
									DocumentId: int(doc_id),
									Markdown:   mdcont,
								}, "markdown"); err != nil {
									beego.Error(err)
								}
							} else {
								beego.Error(err.Error())
							}
						} else {
							beego.Error("读取文档失败：", file.Path, "错误信息：", err)
						}

					}
				}
			}
		}

	}
}

//func (this *BookController) unzipToData(book_id int, identify, zipfile, originFilename string, github bool) {
//
//	//说明：
//	//OSS中的图片存储规则为projects/$identify/项目中图片原路径
//	//本地存储规则为uploads/projects/$identify/项目中图片原路径
//
//	prefix := "cache/store/" + identify
//	os.RemoveAll(prefix)
//	os.MkdirAll(prefix, os.ModePerm)
//	defer func() {
//		os.Remove(zipfile)   //最后删除上传的临时文件
//		os.RemoveAll(prefix) //删除解压后的文件夹
//	}()
//
//	//注意：这里的prefix必须是判断是否是GitHub之前的prefix
//	if err := ziptil.Unzip(zipfile, prefix); err != nil {
//		beego.Error("解压失败", zipfile, err.Error())
//	} else {
//		if github {
//			prefix = prefix + "/" + strings.TrimSuffix(originFilename, ".zip")
//		}
//		this.replaceToAbs(prefix, identify)
//
//		//读取文件，把图片文档录入oss
//		if files, err := filetil.ScanFiles(prefix); err == nil {
//			ModelStore := new(models.DocumentStore)
//			//文档对应的标识
//			for _, file := range files {
//				if !file.IsDir {
//					ext := strings.ToLower(filepath.Ext(file.Path))
//					if strings.Contains(".jpg.jpeg.png.gif.bmp.svg.webp", ext) { //图片，录入oss
//						switch utils.StoreType {
//						case utils.StoreOss:
//							if err := models.ModelStoreOss.MoveToOss(file.Path, "projects/"+identify+strings.TrimPrefix(file.Path, prefix), false, false); err != nil {
//								beego.Error(err)
//							}
//						case utils.StoreLocal:
//							if err := models.ModelStoreLocal.MoveToStore(file.Path, "uploads/projects/"+identify+strings.TrimPrefix(file.Path, prefix), false); err != nil {
//								beego.Error(err)
//							}
//						}
//
//					} else if ext == ".md" || ext == ".markdown" { //markdown文档，提取文档内容，录入数据库
//						doc := new(models.Document)
//						if b, err := ioutil.ReadFile(file.Path); err == nil {
//							mdcont := strings.TrimSpace(string(b))
//							if !strings.HasPrefix(mdcont, "[TOC]") {
//								mdcont = "[TOC]\r\n\r\n" + mdcont
//							}
//							htmlstr := mdtil.Md2html(mdcont)
//							doc.DocumentName = utils.ParseTitleFromMdHtml(htmlstr)
//							doc.BookId = book_id
//							//文档标识
//							doc.Identify = strings.Replace(strings.Trim(strings.TrimPrefix(file.Path, prefix), "/"), "/", "-", -1)
//							doc.MemberId = this.Member.MemberId
//							doc.OrderSort = 1
//							if strings.HasSuffix(strings.ToLower(file.Name), "summary.md") {
//								doc.OrderSort = 0
//							}
//							if doc_id, err := doc.InsertOrUpdate(); err == nil {
//								if err := ModelStore.InsertOrUpdate(models.DocumentStore{
//									DocumentId: int(doc_id),
//									Markdown:   mdcont,
//								}, "markdown"); err != nil {
//									beego.Error(err)
//								}
//							} else {
//								beego.Error(err.Error())
//							}
//						} else {
//							beego.Error("读取文档失败：", file.Path, "错误信息：", err)
//						}
//
//					}
//				}
//			}
//		}
//
//	}
//}

//获取文档项目的根目录
func (this *BookController) getProjectRoot(fl []filetil.FileList) (root string) {
	//获取项目的根目录(感觉这个函数封装的不是很好，有更好的方法，请通过issue告知我，谢谢。)
	i := 1000
	for _, f := range fl {
		if !f.IsDir {
			if cnt := strings.Count(f.Path, "/"); cnt < i {
				root = filepath.Dir(f.Path)
				i = cnt
			}
		}
	}
	return
}

//查找并替换markdown文件中的路径，把图片链接替换成url的相对路径，把文档间的链接替换成【$+文档标识链接】
func (this *BookController) replaceToAbs(projectRoot string, identify string) {
	imgBaseUrl := "/uploads/projects/" + identify
	switch utils.StoreType {
	case utils.StoreLocal:
		imgBaseUrl = "/uploads/projects/" + identify
	case utils.StoreOss:
		imgBaseUrl = this.BaseController.OssDomain + "/projects/" + identify
	}
	files, _ := filetil.ScanFiles(projectRoot)
	for _, file := range files {
		if ext := strings.ToLower(filepath.Ext(file.Path)); ext == ".md" || ext == ".markdown" {
			//mdb ==> markdown byte
			mdb, _ := ioutil.ReadFile(file.Path)
			mdCont := string(mdb)
			basePath := filepath.Dir(file.Path)
			basePath = strings.Trim(strings.Replace(basePath, "\\", "/", -1), "/")
			basePathSlice := strings.Split(basePath, "/")
			l := len(basePathSlice)
			b, _ := ioutil.ReadFile(file.Path)
			output := blackfriday.MarkdownCommon(b)
			doc, _ := goquery.NewDocumentFromReader(strings.NewReader(string(output)))

			//图片链接处理
			doc.Find("img").Each(func(i int, selection *goquery.Selection) {
				//非http开头的图片地址，即是相对地址
				if src, ok := selection.Attr("src"); ok && !strings.HasPrefix(strings.ToLower(src), "http") {
					newSrc := src                                  //默认为旧地址
					if cnt := strings.Count(src, "../"); cnt < l { //以或者"../"开头的路径
						newSrc = strings.Join(basePathSlice[0:l-cnt], "/") + "/" + strings.TrimLeft(src, "./")
					}
					newSrc = imgBaseUrl + "/" + strings.TrimLeft(strings.TrimPrefix(strings.TrimLeft(newSrc, "./"), projectRoot), "/")
					mdCont = strings.Replace(mdCont, src, newSrc, -1)
				}
			})

			//a标签链接处理。要注意判断有锚点的情况
			doc.Find("a").Each(func(i int, selection *goquery.Selection) {
				if href, ok := selection.Attr("href"); ok && !strings.HasPrefix(strings.ToLower(href), "http") && !strings.HasPrefix(href, "#") {
					newHref := href //默认
					if cnt := strings.Count(href, "../"); cnt < l {
						newHref = strings.Join(basePathSlice[0:l-cnt], "/") + "/" + strings.TrimLeft(href, "./")
					}
					newHref = strings.TrimPrefix(strings.Trim(newHref, "/"), projectRoot)
					if !strings.HasPrefix(href, "$") { //原链接不包含$符开头，否则表示已经替换过了。
						newHref = "$" + strings.Replace(strings.Trim(newHref, "/"), "/", "-", -1)
						slice := strings.Split(newHref, "$")
						if ll := len(slice); ll > 0 {
							newHref = "$" + slice[ll-1]
						}
						mdCont = strings.Replace(mdCont, "]("+href, "]("+newHref, -1)
					}
				}
			})
			ioutil.WriteFile(file.Path, []byte(mdCont), os.ModePerm)
		}
	}
}

//给文档项目打分
func (this *BookController) Score() {
	book_id, _ := this.GetInt(":id")
	if book_id == 0 {
		this.JsonResult(1, "文档不存在")
	}
	score, _ := this.GetInt("score")
	if uid := this.Member.MemberId; uid > 0 {
		if err := new(models.Score).AddScore(uid, book_id, score); err == nil {
			this.JsonResult(0, "感谢您给当前文档打分")
		} else {
			this.JsonResult(1, err.Error())
		}
	} else {
		this.JsonResult(1, "给文档打分失败，请先登录再操作")
	}
}

//添加评论
func (this *BookController) Comment() {
	if this.Member.MemberId == 0 {
		this.JsonResult(1, "请先登录在评论")
	}
	content := this.GetString("content")
	if l := len(content); l < 5 || l > 512 {
		this.JsonResult(1, "评论内容先5-512个字符")
	}
	book_id, _ := this.GetInt(":id")
	if book_id > 0 {
		if err := new(models.Comments).AddComments(this.Member.MemberId, book_id, content); err == nil {
			this.JsonResult(0, "评论成功")
		} else {
			this.JsonResult(1, err.Error())
		}
	} else {
		this.JsonResult(1, "文档项目不存在")
	}
}
