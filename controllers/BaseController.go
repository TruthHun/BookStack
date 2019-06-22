package controllers

import (
	"bytes"
	"fmt"
	"net/http"
	"strconv"
	"unicode/utf8"

	"github.com/TruthHun/BookStack/models/store"

	"github.com/TruthHun/BookStack/utils"

	"encoding/json"
	"io"
	"strings"

	"compress/gzip"

	"io/ioutil"
	"path/filepath"

	"time"

	"errors"

	"github.com/PuerkitoBio/goquery"
	"github.com/TruthHun/BookStack/conf"
	"github.com/TruthHun/BookStack/models"
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/orm"
)

type BaseController struct {
	beego.Controller
	Member                *models.Member
	Option                map[string]string
	EnableAnonymous       bool
	EnableDocumentHistory int
	Sitename              string
	OssDomain             string
}
type CookieRemember struct {
	MemberId int
	Account  string
	Time     time.Time
}

// Prepare 预处理.
func (this *BaseController) Prepare() {
	this.Data["Version"] = utils.Version
	this.Member = models.NewMember() //初始化
	this.EnableAnonymous = false
	this.EnableDocumentHistory = 0
	this.OssDomain = strings.TrimRight(beego.AppConfig.String("oss::Domain"), "/ ")
	this.Data["OssDomain"] = this.OssDomain
	this.Data["StaticDomain"] = strings.Trim(beego.AppConfig.DefaultString("static_domain", ""), "/")
	//从session中获取用户信息
	if member, ok := this.GetSession(conf.LoginSessionName).(models.Member); ok && member.MemberId > 0 {
		m, _ := models.NewMember().Find(member.MemberId)
		this.Member = m
	} else {
		//如果Cookie中存在登录信息，从cookie中获取用户信息
		if cookie, ok := this.GetSecureCookie(conf.GetAppKey(), "login"); ok {
			var remember CookieRemember
			err := utils.Decode(cookie, &remember)
			if err == nil {
				member, err := models.NewMember().Find(remember.MemberId)
				if err == nil {
					this.SetMember(*member)
					this.Member = member
				}
			}
		}

	}
	if this.Member.RoleName == "" {
		this.Member.ResolveRoleName()
	}
	this.Data["Member"] = this.Member
	this.Data["BaseUrl"] = this.BaseUrl()

	if options, err := models.NewOption().All(); err == nil {
		this.Option = make(map[string]string, len(options))
		for _, item := range options {
			if item.OptionName == "SITE_NAME" {
				this.Sitename = item.OptionValue
			}
			this.Data[item.OptionName] = item.OptionValue
			this.Option[item.OptionName] = item.OptionValue
			if strings.EqualFold(item.OptionName, "ENABLE_ANONYMOUS") && item.OptionValue == "true" {
				this.EnableAnonymous = true
			}
			if verNum, _ := strconv.Atoi(item.OptionValue); strings.EqualFold(item.OptionName, "ENABLE_DOCUMENT_HISTORY") && verNum > 0 {
				this.EnableDocumentHistory = verNum
			}
		}
	}
	this.Data["SiteName"] = this.Sitename
	this.Data["Friendlinks"] = new(models.FriendLink).GetList(false)
}

// SetMember 获取或设置当前登录用户信息,如果 MemberId 小于 0 则标识删除 Session
func (this *BaseController) SetMember(member models.Member) {

	if member.MemberId <= 0 {
		this.DelSession(conf.LoginSessionName)
		this.DelSession("uid")
		this.DestroySession()
	} else {
		this.SetSession(conf.LoginSessionName, member)
		this.SetSession("uid", member.MemberId)
	}
}

// JsonResult 响应 json 结果
func (this *BaseController) JsonResult(errCode int, errMsg string, data ...interface{}) {
	jsonData := make(map[string]interface{}, 3)
	jsonData["errcode"] = errCode
	jsonData["message"] = errMsg

	if len(data) > 0 && data[0] != nil {
		jsonData["data"] = data[0]
	}
	returnJSON, err := json.Marshal(jsonData)
	if err != nil {
		beego.Error(err)
	}
	this.Ctx.ResponseWriter.Header().Set("Content-Type", "application/json; charset=utf-8")
	//this.Ctx.ResponseWriter.Header().Set("Cache-Control", "no-cache, no-store")//解决回退出现json的问题
	//使用gzip原始，json数据会只有原本数据的10分之一左右
	if strings.Contains(strings.ToLower(this.Ctx.Request.Header.Get("Accept-Encoding")), "gzip") {
		this.Ctx.ResponseWriter.Header().Set("Content-Encoding", "gzip")
		//gzip压缩
		w := gzip.NewWriter(this.Ctx.ResponseWriter)
		defer w.Close()
		w.Write(returnJSON)
		w.Flush()
	} else {
		io.WriteString(this.Ctx.ResponseWriter, string(returnJSON))
	}
	this.StopRun()
}

// ExecuteViewPathTemplate 执行指定的模板并返回执行结果.
func (this *BaseController) ExecuteViewPathTemplate(tplName string, data interface{}) (string, error) {
	var buf bytes.Buffer

	viewPath := this.ViewPath

	if this.ViewPath == "" {
		viewPath = beego.BConfig.WebConfig.ViewsPath

	}

	if err := beego.ExecuteViewPathTemplate(&buf, tplName, viewPath, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (this *BaseController) BaseUrl() string {
	host := beego.AppConfig.String("sitemap_host")
	if len(host) > 0 {
		if strings.HasPrefix(host, "http://") || strings.HasPrefix(host, "https://") {
			return host
		}
		return this.Ctx.Input.Scheme() + "://" + host
	}
	return this.Ctx.Input.Scheme() + "://" + this.Ctx.Request.Host
}

//显示错误信息页面.
func (this *BaseController) ShowErrorPage(errCode int, errMsg string) {
	this.TplName = "errors/error.html"
	this.Data["ErrorMessage"] = errMsg
	this.Data["ErrorCode"] = errCode
	this.StopRun()
}

//根据页面获取seo
//@param			page			页面标识
//@param			defSeo			默认的seo的map，必须有title、keywords和description字段
func (this *BaseController) GetSeoByPage(page string, defSeo map[string]string) {
	var seo models.Seo

	orm.NewOrm().QueryTable(models.TableSeo).Filter("Page", page).One(&seo)
	defSeo["sitename"] = this.Sitename
	if seo.Id > 0 {
		for k, v := range defSeo {
			seo.Title = strings.Replace(seo.Title, fmt.Sprintf("{%v}", k), v, -1)
			seo.Keywords = strings.Replace(seo.Keywords, fmt.Sprintf("{%v}", k), v, -1)
			seo.Description = strings.Replace(seo.Description, fmt.Sprintf("{%v}", k), v, -1)
		}
	}
	this.Data["SeoTitle"] = seo.Title
	this.Data["SeoKeywords"] = seo.Keywords
	this.Data["SeoDescription"] = seo.Description
}

//站点地图
func (this *BaseController) Sitemap() {
	this.Data["SeoTitle"] = "站点地图 - " + this.Sitename
	page, _ := this.GetInt("page")
	listRows := 100
	totalCount, docs := models.SitemapData(page, listRows)
	if totalCount > 0 {
		html := utils.GetPagerHtml(this.Ctx.Request.RequestURI, page, listRows, int(totalCount))
		this.Data["PageHtml"] = html
	} else {
		this.Data["PageHtml"] = ""
	}
	//this.JsonResult(0, "aaa", docs)
	this.Data["Docs"] = docs
	this.TplName = "widgets/sitemap.html"
}

//静态文件，这个加在路由的最后
func (this *BaseController) StaticFile() {
	splat := this.GetString(":splat")
	ext := filepath.Ext(splat)
	if strings.Contains(beego.AppConfig.String("StaticExt"), strings.ToLower(ext)) {
		if b, err := ioutil.ReadFile(splat); err == nil {
			this.Ctx.ResponseWriter.Write(b)
			return
		}
	}
	this.Abort("404")
}

func (this *BaseController) loginByMemberId(memberId int) (err error) {
	member, err := models.NewMember().Find(memberId)
	if member.MemberId == 0 {
		return errors.New("用户不存在")
	}
	//如果没有数据
	if err != nil {
		return err
	}
	member.LastLoginTime = time.Now()
	member.Update()
	this.SetMember(*member)
	var remember CookieRemember
	remember.MemberId = member.MemberId
	remember.Account = member.Account
	remember.Time = time.Now()
	v, err := utils.Encode(remember)
	if err == nil {
		this.SetSecureCookie(conf.GetAppKey(), "login", v, 24*3600*365)
	}
	return err
}

//在markdown头部加上<bookstack></bookstack>或者<bookstack/>，即解析markdown中的ul>li>a链接作为目录
func (this *BaseController) sortBySummary(bookIdentify, htmlStr string, bookId int) {
	debug := beego.AppConfig.String("runmod") != "prod"
	o := orm.NewOrm()
	qs := o.QueryTable("md_documents").Filter("book_id", bookId)
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlStr))
	if err != nil {
		beego.Error(err)
	}
	idx := 1
	if debug {
		beego.Info("根据summary文件进行排序")
	}

	//查找ul>li下的所有a标签，并提取text和href，查询数据库，如果标识不存在，则把这些新的数据录入数据库
	var hrefs = make(map[string]string)
	var hrefSlice []interface{}
	var docs []models.Document
	doc.Find("li>a").Each(func(i int, selection *goquery.Selection) {
		if href, ok := selection.Attr("href"); ok && strings.HasPrefix(href, "$") {
			href = strings.TrimLeft(strings.Replace(href, "/", "-", -1), "$")
			if utf8.RuneCountInString(href) <= 100 {
				hrefs[href] = selection.Text()
				hrefSlice = append(hrefSlice, href)
			}
		}
	})
	if debug {
		beego.Info(hrefs)
	}
	if len(hrefSlice) > 0 {
		if _, err := qs.Filter("identify__in", hrefSlice...).Limit(len(hrefSlice)).All(&docs, "identify"); err != nil {
			beego.Error(err.Error())
		} else {
			for _, doc := range docs {
				//删除存在的标识
				delete(hrefs, doc.Identify)
			}
		}
	}
	if len(hrefs) > 0 { //存在未创建的文档，先创建
		ModelStore := new(models.DocumentStore)
		for identify, docName := range hrefs {
			doc := models.Document{
				BookId:       bookId,
				Identify:     identify,
				DocumentName: docName,
				CreateTime:   time.Now(),
				ModifyTime:   time.Now(),
			}
			// 如果文档标识超过了规定长度（100），则进行忽略
			if utf8.RuneCountInString(identify) <= 100 {
				if docId, err := doc.InsertOrUpdate(); err == nil {
					if err = ModelStore.InsertOrUpdate(models.DocumentStore{DocumentId: int(docId), Markdown: "[TOC]\n\r\n\r"}); err != nil {
						beego.Error(err.Error())
					}
				}
			}
		}

	}

	// 重置所有之前的文档排序
	_, _ = qs.Update(orm.Params{"order_sort": 100000})

	doc.Find("a").Each(func(i int, selection *goquery.Selection) {
		docName := selection.Text()
		pid := 0
		if docId, exist := selection.Attr("data-pid"); exist {
			did, _ := strconv.Atoi(docId)
			eleParent := selection.Parent().Parent().Parent()
			if eleParent.Is("li") {
				fst := eleParent.Find("a").First()
				pidstr, _ := fst.Attr("data-pid")
				//如果这里的pid为0，表示数据库还没存在这个标识，需要创建
				pid, _ = strconv.Atoi(pidstr)
			}
			if did > 0 {
				qs.Filter("document_id", did).Update(orm.Params{
					"parent_id": pid, "document_name": docName,
					"order_sort": idx, "modify_time": time.Now(),
				})
			}
		} else if href, ok := selection.Attr("href"); ok && strings.HasPrefix(href, "$") {
			identify := strings.TrimPrefix(href, "$") //文档标识
			eleParent := selection.Parent().Parent().Parent()
			if eleParent.Is("li") {
				if parentHref, ok := eleParent.Find("a").First().Attr("href"); ok {
					var one models.Document
					qs.Filter("identify", strings.Split(strings.TrimPrefix(parentHref, "$"), "#")[0]).One(&one, "document_id")
					pid = one.DocumentId
				}
			}

			if _, err := qs.Filter("identify", identify).Update(orm.Params{
				"parent_id": pid, "document_name": docName,
				"order_sort": idx, "modify_time": time.Now(),
			}); err != nil {
				beego.Error(err)
			}
		}
		idx++
	})

	if len(hrefs) > 0 { //如果有新创建的文档，则再调用一遍，用于处理排序
		this.replaceLinks(bookIdentify, htmlStr, true)
	}
}

//排序
type Sort struct {
	Id        int
	Pid       int
	SortOrder int
	Identify  string
}

//替换链接
//如果是summary，则根据这个进行排序调整
func (this *BaseController) replaceLinks(bookIdentify string, docHtml string, isSummary ...bool) string {
	var (
		book models.Book
		docs []models.Document
		o    = orm.NewOrm()
	)
	o.QueryTable("md_books").Filter("identify", bookIdentify).One(&book, "book_id")
	if book.BookId > 0 {
		o.QueryTable("md_documents").Filter("book_id", book.BookId).Limit(5000).All(&docs, "identify", "document_id")
		if len(docs) > 0 {
			Links := make(map[string]string)
			for _, doc := range docs {
				idStr := strconv.Itoa(doc.DocumentId)
				if len(doc.Identify) > 0 {
					Links["$"+strings.ToLower(doc.Identify)] = beego.URLFor("DocumentController.Read", ":key", bookIdentify, ":id", doc.Identify) + "||" + idStr
				}
				if doc.DocumentId > 0 {
					Links["$"+strconv.Itoa(doc.DocumentId)] = beego.URLFor("DocumentController.Read", ":key", bookIdentify, ":id", doc.DocumentId) + "||" + idStr
				}
			}

			//替换文档内容中的链接
			if gq, err := goquery.NewDocumentFromReader(strings.NewReader(docHtml)); err == nil {

				gq.Find("a").Each(func(i int, selection *goquery.Selection) {
					if href, ok := selection.Attr("href"); ok && strings.HasPrefix(href, "$") {
						if slice := strings.Split(href, "#"); len(slice) > 1 {
							if newHref, ok := Links[strings.ToLower(slice[0])]; ok {
								arr := strings.Split(newHref, "||") //整理的arr数组长度，肯定为2，所以不做数组长度判断
								selection.SetAttr("href", arr[0]+"#"+strings.Join(slice[1:], "#"))
								selection.SetAttr("data-pid", arr[1])
							}
						} else {
							if newHref, ok := Links[strings.ToLower(href)]; ok {
								arr := strings.Split(newHref, "||") //整理的arr数组长度，肯定为2，所以不做数组长度判断
								selection.SetAttr("href", arr[0])
								selection.SetAttr("data-pid", arr[1])
							}
						}
					}
				})

				if newHtml, err := gq.Find("body").Html(); err == nil {
					docHtml = newHtml
					if len(isSummary) > 0 && isSummary[0] == true { //更新排序
						this.sortBySummary(bookIdentify, docHtml, book.BookId) //更新排序
					}
				}
			} else {
				beego.Error(err.Error())
			}
		}
	}

	return docHtml
}

//内容采集
func (this *BaseController) Crawl() {
	if this.Member.MemberId > 0 {
		if val, ok := this.GetSession("crawl").(string); ok && val == "1" {
			this.JsonResult(1, "您提交的上一次采集未完成，请稍后再提交新的内容采集")
		}
		this.SetSession("crawl", "1")
		defer this.DelSession("crawl")
		urlStr := this.GetString("url")
		force, _ := this.GetBool("force")              //是否是强力采集，强力采集，使用Chrome
		intelligence, _ := this.GetInt("intelligence") //是否是强力采集，强力采集，使用Chrome
		contType, _ := this.GetInt("type")
		diySel := this.GetString("diy")
		content, err := utils.CrawlHtml2Markdown(urlStr, contType, force, intelligence, diySel, []string{}, nil)
		if err != nil {
			this.JsonResult(1, "采集失败："+err.Error())
		}
		this.JsonResult(0, "采集成功", content)
	}
	this.JsonResult(1, "请先登录再操作")
}

//关注或取消关注
func (this *BaseController) SetFollow() {
	var cancel bool
	if this.Member.MemberId == 0 {
		this.JsonResult(1, "请先登录")
	}
	uid, _ := this.GetInt(":uid")
	if uid == this.Member.MemberId {
		this.JsonResult(1, "自己不能关注自己")
	}
	cancel, _ = new(models.Fans).FollowOrCancel(uid, this.Member.MemberId)
	if cancel {
		this.JsonResult(0, "您已经成功取消了关注")
	}
	this.JsonResult(0, "您已经成功关注了Ta")
}

// 项目静态文件
func (this *BaseController) ProjectsFile() {
	prefix := "projects/"
	object := prefix + this.GetString(":splat")

	//这里的时间只是起到缓存的作用
	t, _ := time.Parse("2006-01-02 15:04:05", "2006-01-02 15:04:05")
	date := t.Format(http.TimeFormat)
	since := this.Ctx.Request.Header.Get("If-Modified-Since")
	if since == date {
		this.Ctx.ResponseWriter.WriteHeader(http.StatusNotModified)
		return
	}

	if utils.StoreType == utils.StoreOss { //oss
		reader, err := store.NewOss().GetFileReader(object)
		if err != nil {
			beego.Error(err.Error())
			this.Abort("404")
		}
		b, err := ioutil.ReadAll(reader)
		if err != nil {
			beego.Error(err.Error())
			this.Abort("404")
		}
		this.Ctx.ResponseWriter.Header().Set("Last-Modified", date)
		if strings.HasSuffix(object, ".svg") {
			this.Ctx.ResponseWriter.Header().Set("Content-Type", "image/svg+xml")
		}
		this.Ctx.ResponseWriter.Write(b)
	} else { //local
		this.Abort("404")
	}
}
