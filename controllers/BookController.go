package controllers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/TruthHun/BookStack/graphics"
	"github.com/TruthHun/BookStack/models/store"
	"github.com/russross/blackfriday"

	"github.com/TruthHun/BookStack/conf"
	"github.com/TruthHun/BookStack/models"
	"github.com/TruthHun/BookStack/utils"
	"github.com/TruthHun/BookStack/utils/html2md"
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

// 替换字符串
func (this *BookController) Replace() {
	identify := this.GetString(":key")
	src := this.GetString("src")
	dst := this.GetString("dst")

	if this.Member.MemberId == 0 {
		this.JsonResult(1, "请先登录")
	}

	book, err := models.NewBookResult().FindByIdentify(identify, this.Member.MemberId)
	if err != nil {
		if err == orm.ErrNoRows {
			this.JsonResult(1, "内容不存在")
		}
		this.JsonResult(1, err.Error())
	}

	models.NewBook().Replace(book.BookId, src, dst)

	this.JsonResult(0, "替换成功")
}

func (this *BookController) Index() {
	this.Data["SettingBook"] = true
	this.TplName = "book/index.html"
	private, _ := this.GetInt("private", 1) //是否是私有文档
	wd := this.GetString("wd", "")
	this.Data["Private"] = private
	this.Data["Wd"] = wd
	pageIndex, _ := this.GetInt("page", 1)
	books, totalCount, _ := models.NewBook().FindToPager(pageIndex, conf.PageSize, this.Member.MemberId, wd, private)
	ebookStats := make(map[int]map[string]models.Ebook)
	modelEbook := models.NewEbook()
	for _, book := range books {
		ebookStats[book.BookId] = modelEbook.GetStats(book.BookId)
	}
	ebookJSON, _ := json.Marshal(ebookStats)
	this.Data["EbookStats"] = template.JS(string(ebookJSON))
	if totalCount > 0 {
		this.Data["PageHtml"] = utils.NewPaginations(conf.RollPage, totalCount, conf.PageSize, pageIndex, beego.URLFor("BookController.Index"), fmt.Sprintf("&private=%v&wd=%s", private, url.QueryEscape(wd)))
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
	installedDependencies := utils.GetInstalledDependencies()
	for _, item := range installedDependencies {
		this.Data[item.Name+"_is_installed"] = item.IsInstalled
	}
}

// 收藏书籍
func (this *BookController) Star() {
	uid := this.BaseController.Member.MemberId
	if uid <= 0 {
		this.JsonResult(1, "收藏失败，请先登录")
	}

	id, _ := this.GetInt(":id")
	if id <= 0 {
		this.JsonResult(1, "收藏失败，书籍不存在")
	}

	cancel, err := new(models.Star).Star(uid, id)
	data := map[string]bool{"IsCancel": cancel}
	if err != nil {
		beego.Error(err.Error())
		if cancel {
			this.JsonResult(1, "取消收藏失败", data)
		}
		this.JsonResult(1, "添加收藏失败", data)
	}

	if cancel {
		this.JsonResult(0, "取消收藏成功", data)
	}
	this.JsonResult(0, "添加收藏成功", data)
}

// Dashboard 书籍概要 .
func (this *BookController) Dashboard() {

	this.TplName = "book/dashboard.html"

	key := this.Ctx.Input.Param(":key")

	if key == "" {
		this.Abort("404")
	}

	book, err := models.NewBookResult().FindByIdentify(key, this.Member.MemberId)
	if err != nil {
		beego.Error(err)
		if err == models.ErrPermissionDenied {
			this.Abort("404")
		}
		this.Abort("404")
	}

	this.Data["Model"] = *book
}

// Setting 书籍设置 .
func (this *BookController) Setting() {

	key := this.Ctx.Input.Param(":key")

	if key == "" {
		this.Abort("404")
	}

	book, err := models.NewBookResult().FindByIdentify(key, this.Member.MemberId)
	if err != nil && err != orm.ErrNoRows {
		beego.Error(err.Error())

		if err == orm.ErrNoRows {
			this.Abort("404")
		}

		if err == models.ErrPermissionDenied {
			this.Abort("404")
		}
		this.Abort("404")
	}

	//如果不是创始人也不是管理员则不能操作
	if book.RoleId != conf.BookFounder && book.RoleId != conf.BookAdmin {
		this.Abort("404")
	}

	if book.PrivateToken != "" {
		//book.PrivateToken = this.BaseUrl() + beego.URLFor("DocumentController.Index", ":key", book.Identify, "token", book.PrivateToken)
		tipsFmt := "访问链接：%v  访问密码：%v"
		book.PrivateToken = fmt.Sprintf(tipsFmt, this.BaseUrl()+beego.URLFor("DocumentController.Index", ":key", book.Identify), book.PrivateToken)
	}

	//查询当前书籍的分类id
	if selectedCates, rows, _ := new(models.BookCategory).GetByBookId(book.BookId); rows > 0 {
		var maps = make(map[int]bool)
		for _, cate := range selectedCates {
			maps[cate.Id] = true
		}
		this.Data["Maps"] = maps
	}

	ver := models.NewVersion()
	this.Data["Versions"] = ver.All()
	this.Data["VersionItem"] = ver.GetVersionItem(book.Identify)

	this.Data["Cates"], _ = new(models.Category).GetCates(-1, 1)
	this.Data["Model"] = book
	this.TplName = "book/setting.html"
	installedDependencies := utils.GetInstalledDependencies()
	for _, item := range installedDependencies {
		this.Data[item.Name+"_is_installed"] = item.IsInstalled
	}
}

// SaveBook 保存书籍信息
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

	bookName := strings.TrimSpace(this.GetString("book_name"))
	description := strings.TrimSpace(this.GetString("description", ""))
	commentStatus := this.GetString("comment_status")
	tag := strings.TrimSpace(this.GetString("label"))
	editor := strings.TrimSpace(this.GetString("editor"))

	if strings.Count(description, "") > 500 {
		this.JsonResult(6004, "书籍描述不能大于500字")
	}
	if commentStatus != "open" && commentStatus != "closed" && commentStatus != "group_only" && commentStatus != "registered_only" {
		commentStatus = "closed"
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

	book.BookName = bookName
	book.Description = description
	book.CommentStatus = commentStatus
	book.Label = tag
	book.Editor = editor
	book.Author = this.GetString("author")
	book.AuthorURL = this.GetString("author_url")
	book.Lang = this.GetString("lang")
	book.AdTitle = this.GetString("ad_title")
	book.AdLink = this.GetString("ad_link")
	_, book.NavJSON = this.parseBookNav()

	if err := book.Update(); err != nil {
		this.JsonResult(6006, "保存失败")
	}
	bookResult.BookName = bookName
	bookResult.Description = description
	bookResult.CommentStatus = commentStatus
	bookResult.Label = tag

	//更新书籍分类
	if cids, ok := this.Ctx.Request.Form["cid"]; ok {
		new(models.BookCategory).SetBookCates(book.BookId, cids)
	}

	go func() {
		es := models.ElasticSearchData{
			Id:       book.BookId,
			BookId:   0,
			Title:    book.BookName,
			Keywords: book.Label,
			Content:  book.Description,
			Vcnt:     book.Vcnt,
			Private:  book.PrivatelyOwned,
		}
		client := models.NewElasticSearchClient()
		if errSearch := client.BuildIndex(es); errSearch != nil && client.On {
			beego.Error(errSearch.Error())
		}
	}()
	versionId, _ := this.GetInt("version")
	versionNO := this.GetString("version_no")
	models.NewVersion().InsertOrUpdateVersionItem(versionId, book.Identify, book.BookName, versionNO)
	go models.CountCategory()
	this.JsonResult(0, "ok", bookResult)
}

func (this *BookController) parseBookNav() (navs models.BookNavs, navStr string) {
	var data struct {
		Name   []string `json:"name"`
		URL    []string `json:"url"`
		Sort   []string `json:"sort"`
		Icon   []string `json:"icon"`
		Color  []string `json:"color"`
		Target []string `json:"target"`
	}
	b, _ := json.Marshal(this.Ctx.Request.PostForm)
	json.Unmarshal(b, &data)
	lenName := len(data.Name)
	if lenName == 0 || lenName != len(data.URL) || lenName != len(data.Sort) || lenName != len(data.Icon) || lenName != len(data.Color) || lenName != len(data.Target) {
		return
	}

	for idx, name := range data.Name {
		name = strings.TrimSpace(name)
		url := strings.TrimSpace(data.URL[idx])
		if url == "" || name == "" {
			continue
		}
		nav := models.BookNav{
			Name:   name,
			URL:    url,
			Color:  strings.TrimSpace(data.Color[idx]),
			Icon:   strings.TrimSpace(data.Icon[idx]),
			Target: strings.TrimSpace(data.Target[idx]),
		}
		nav.Sort, _ = strconv.Atoi(strings.TrimSpace(data.Sort[idx]))
		navs = append(navs, nav)
	}
	if len(navs) > 0 {
		// 排序
		sort.Sort(navs)
		b, _ := json.Marshal(navs)
		navStr = string(b)
	}
	return
}

// 设置书籍私有状态.
func (this *BookController) PrivatelyOwned() {

	status := this.GetString("status")
	if this.forbidGeneralRole() && status == "open" {
		this.JsonResult(6001, "您的角色非作者和管理员，无法将书籍设置为公开")
	}
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
	go func() {
		models.CountCategory()

		public := true
		if state == 1 {
			public = false
		}
		client := models.NewElasticSearchClient()
		if errSet := client.SetBookPublic(bookResult.BookId, public); errSet != nil && client.On {
			beego.Error(errSet.Error())
		}
	}()
	this.JsonResult(0, "ok")
}

// Transfer 转让书籍.
func (this *BookController) Transfer() {

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

// 上传书籍封面.
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

	filePath = strings.ReplaceAll(filepath.Join("uploads", time.Now().Format("200601"), fileName+ext), "\\", "/")

	//生成缩略图并保存到磁盘
	err = graphics.ImageResizeSaveFile(subImg, 175, 230, filePath)
	if err != nil {
		logs.Error("ImageResizeSaveFile => ", err.Error())
		this.JsonResult(500, "保存图片失败")
	}

	url := "/" + strings.Replace(filePath, "\\", "/", -1)
	if strings.HasPrefix(url, "//") {
		url = string(url[1:])
	}

	oldCover := strings.ReplaceAll(book.Cover, "\\", "/")
	osspath := fmt.Sprintf("projects/%v/%v", book.Identify, strings.TrimLeft(url, "./"))
	book.Cover = "/" + osspath
	if utils.StoreType == utils.StoreLocal {
		book.Cover = url
	}

	if err := book.Update(); err != nil {
		this.JsonResult(6001, "保存图片失败")
	}

	//如果原封面不是默认封面则删除
	if oldCover != conf.GetDefaultCover() {
		os.Remove("." + oldCover)
		switch utils.StoreType {
		case utils.StoreOss:
			store.ModelStoreOss.DelFromOss(oldCover) //从OSS执行一次删除
		case utils.StoreLocal:
			store.ModelStoreLocal.DelFiles(oldCover) //从本地执行一次删除
		}
	}

	switch utils.StoreType {
	case utils.StoreOss: //oss
		if err := store.ModelStoreOss.MoveToOss("."+url, osspath, true, false); err != nil {
			beego.Error(err.Error())
		} else {
			url = strings.TrimRight(beego.AppConfig.String("oss::Domain"), "/ ") + "/" + osspath + "/cover"
		}
	case utils.StoreLocal:
		save := book.Cover
		if err := store.ModelStoreLocal.MoveToStore("."+url, save); err != nil {
			beego.Error(err.Error())
		} else {
			url = book.Cover
		}
	}
	this.JsonResult(0, "ok", url)
}

// Users 用户列表.
func (this *BookController) Users() {

	pageIndex, _ := this.GetInt("page", 1)

	key := this.Ctx.Input.Param(":key")
	if key == "" {
		this.Abort("404")
	}

	book, err := models.NewBookResult().FindByIdentify(key, this.Member.MemberId)
	if err != nil {
		if err == models.ErrPermissionDenied {
			this.Abort("404")
		}
		this.Abort("404")
	}

	this.Data["Model"] = *book
	pageSize := 10
	members, totalCount, _ := models.NewMemberRelationshipResult().FindForUsersByBookId(book.BookId, pageIndex, pageSize)

	for idx, member := range members {
		member.Avatar = utils.ShowImg(member.Avatar, "avatar")
		members[idx] = member
	}

	if totalCount > 0 {
		html := utils.GetPagerHtml(this.Ctx.Request.RequestURI, pageIndex, pageSize, totalCount)
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
	this.TplName = "book/users.html"
}

// Create 创建书籍.
func (this *BookController) Create() {
	if opt, err := models.NewOption().FindByKey("ALL_CAN_WRITE_BOOK"); err == nil {
		if opt.OptionValue == "false" && this.Member.Role == conf.MemberGeneralRole { // 读者无权限创建书籍
			this.JsonResult(1, "普通读者无法创建书籍，如需创建书籍，请向管理员申请成为作者")
		}
	}

	bookName := strings.TrimSpace(this.GetString("book_name", ""))
	identify := strings.TrimSpace(this.GetString("identify", ""))
	description := strings.TrimSpace(this.GetString("description", ""))
	author := strings.TrimSpace(this.GetString("author", ""))
	authorURL := strings.TrimSpace(this.GetString("author_url", ""))
	privatelyOwned, _ := strconv.Atoi(this.GetString("privately_owned"))
	commentStatus := this.GetString("comment_status")

	if bookName == "" {
		this.JsonResult(6001, "书籍名称不能为空")
	}

	if identify == "" {
		this.JsonResult(6002, "书籍标识不能为空")
	}

	ok, err1 := regexp.MatchString(`^[a-zA-Z0-9_\-\.]*$`, identify)
	if !ok || err1 != nil {
		this.JsonResult(6003, "书籍标识只能包含字母、数字，以及“-”、“.”和“_”符号，且不能是纯数字")
	}

	if num, _ := strconv.Atoi(identify); strconv.Itoa(num) == identify {
		this.JsonResult(6003, "书籍标识不能是纯数字")
	}

	if strings.Count(identify, "") > 50 {
		this.JsonResult(6004, "书籍标识不能超过50字")
	}

	if strings.Count(description, "") > 500 {
		this.JsonResult(6004, "书籍描述不能大于500字")
	}

	if privatelyOwned != 0 && privatelyOwned != 1 {
		privatelyOwned = 1
	}
	if commentStatus != "open" && commentStatus != "closed" && commentStatus != "group_only" && commentStatus != "registered_only" {
		commentStatus = "closed"
	}

	book := models.NewBook()

	if books, _ := book.FindByField("identify", identify); len(books) > 0 {
		this.JsonResult(6006, "书籍标识已存在")
	}

	book.Label = strings.Join(utils.SegWords(bookName+"。"+description, 5), ",")
	book.BookName = bookName
	book.Author = author
	book.AuthorURL = authorURL
	book.Description = description
	book.CommentCount = 0
	book.PrivatelyOwned = privatelyOwned
	book.CommentStatus = commentStatus
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
	book.GenerateTime, _ = time.Parse("2006-01-02 15:04:05", "2000-01-02 15:04:05") // 电子书生成的默认时间
	book.ReleaseTime = defaultTime

	if err := book.Insert(); err != nil {
		logs.Error("Insert => ", err)
		this.JsonResult(6005, "保存书籍失败")
	}

	bookResult, err := models.NewBookResult().FindByIdentify(book.Identify, this.Member.MemberId)
	if err != nil {
		beego.Error(err)
	}

	this.JsonResult(0, "ok", bookResult)
}

// Create 创建书籍.
func (this *BookController) Copy() {
	if opt, err := models.NewOption().FindByKey("ALL_CAN_WRITE_BOOK"); err == nil {
		if opt.OptionValue == "false" && this.Member.Role == conf.MemberGeneralRole { // 读者无权限创建书籍
			this.JsonResult(1, "普通读者无法创建书籍，如需创建书籍，请向管理员申请成为作者")
		}
	}
	identify := strings.TrimSpace(this.GetString("identify", ""))
	sourceIdentify := strings.TrimSpace(this.GetString("source_identify", ""))
	sourceBook, err := models.NewBook().FindByIdentify(sourceIdentify)
	if err != nil {
		this.JsonResult(1, err.Error())
	}
	existBook, _ := models.NewBook().FindByIdentify(identify, "book_id")
	if existBook != nil && existBook.BookId > 0 {
		this.JsonResult(1, "请更换新的书籍标识")
	}

	// 如果是私有书籍，且不是团队的人，不允许拷贝该项目
	if sourceBook.PrivatelyOwned == 1 {
		rel, err := models.NewRelationship().FindByBookIdAndMemberId(sourceBook.BookId, this.Member.MemberId)
		if err != nil || rel == nil || rel.RelationshipId == 0 {
			this.JsonResult(1, "无拷贝书籍权限")
		}
	}
	sourceBook.BookId = 0
	sourceBook.BookName = strings.TrimSpace(this.GetString("book_name", ""))
	sourceBook.Identify = identify
	sourceBook.Description = strings.TrimSpace(this.GetString("description", ""))
	sourceBook.Author = strings.TrimSpace(this.GetString("author", ""))
	sourceBook.AuthorURL = strings.TrimSpace(this.GetString("author_url", ""))
	sourceBook.PrivatelyOwned, _ = strconv.Atoi(this.GetString("privately_owned"))
	sourceBook.MemberId = this.Member.MemberId
	err = sourceBook.Copy(sourceIdentify)
	if err != nil {
		this.JsonResult(1, "拷贝书籍失败："+err.Error())
	}

	this.JsonResult(0, "拷贝书籍成功")
}

// CreateToken 创建访问来令牌.
func (this *BookController) CreateToken() {
	if this.forbidGeneralRole() {
		this.JsonResult(6001, "您的角色非作者和管理员，无法创建访问令牌")
	}
	action := this.GetString("action")

	bookResult, err := this.IsPermission()

	if err != nil {
		if err == models.ErrPermissionDenied {
			this.JsonResult(403, "权限不足")
		}

		if err == orm.ErrNoRows {
			this.JsonResult(404, "书籍不存在")
		}

		logs.Error("生成阅读令牌失败 =>", err)
		this.JsonResult(6002, err.Error())
	}

	book := models.NewBook()
	if _, err := book.Find(bookResult.BookId); err != nil {
		this.JsonResult(6001, "书籍不存在")
	}

	if action == "create" {
		if bookResult.PrivatelyOwned == 0 {
			this.JsonResult(6001, "公开书籍不能创建阅读令牌")
		}

		book.PrivateToken = string(utils.Krand(conf.GetTokenSize(), utils.KC_RAND_KIND_ALL))
		if err := book.Update(); err != nil {
			logs.Error("生成阅读令牌失败 => ", err)
			this.JsonResult(6003, "生成阅读令牌失败")
		}
		//book.PrivateToken = this.BaseUrl() + beego.URLFor("DocumentController.Index", ":key", book.Identify, "token", book.PrivateToken)
		tipsFmt := "访问链接：%v  访问密码：%v"
		privateToken := fmt.Sprintf(tipsFmt, this.BaseUrl()+beego.URLFor("DocumentController.Index", ":key", book.Identify), book.PrivateToken)
		this.JsonResult(0, "ok", privateToken)
	}

	book.PrivateToken = ""
	if err := book.Update(); err != nil {
		logs.Error("CreateToken => ", err)
		this.JsonResult(6004, "删除令牌失败")
	}
	this.JsonResult(0, "ok", "")
}

// Delete 删除书籍.
func (this *BookController) Delete() {

	bookResult, err := this.IsPermission()
	if err != nil {
		this.JsonResult(6001, err.Error())
	}

	if bookResult.RoleId != conf.BookFounder {
		this.JsonResult(6002, "只有创始人才能删除书籍")
	}

	//用户密码
	pwd := this.GetString("password")
	if m, err := models.NewMember().Login(this.Member.Account, pwd); err != nil || m.MemberId == 0 {
		this.JsonResult(1, "书籍删除失败，您的登录密码不正确")
	}

	err = models.NewBook().ThoroughDeleteBook(bookResult.BookId)
	if err == orm.ErrNoRows {
		this.JsonResult(6002, "书籍不存在")
	}

	if err != nil {
		logs.Error("删除书籍 => ", err)
		this.JsonResult(6003, "删除失败")
	}

	go func() {
		client := models.NewElasticSearchClient()
		if errDel := client.DeleteIndex(bookResult.BookId, true); errDel != nil && client.On {
			beego.Error(errDel.Error())
		}
	}()
	go models.CountCategory()
	this.JsonResult(0, "ok")
}

// 发布书籍.
func (this *BookController) Release() {
	identify := this.GetString("identify")
	force, _ := this.GetBool("force")
	bookId := 0
	if this.Member.IsAdministrator() {
		book, err := models.NewBook().FindByFieldFirst("identify", identify)
		if err != nil {
			beego.Error(err)
		}
		bookId = book.BookId
	} else {
		book, err := models.NewBookResult().FindByIdentify(identify, this.Member.MemberId)
		if err != nil {
			if err == models.ErrPermissionDenied {
				this.JsonResult(6001, "权限不足")
			}
			if err == orm.ErrNoRows {
				this.JsonResult(6002, "书籍不存在")
			}
			beego.Error(err)
			this.JsonResult(6003, "未知错误")
		}
		if book.RoleId != conf.BookAdmin && book.RoleId != conf.BookFounder && book.RoleId != conf.BookEditor {
			this.JsonResult(6003, "权限不足")
		}
		bookId = book.BookId
	}

	go models.NewDocument().ReleaseContent(bookId, this.BaseUrl(), force)

	this.JsonResult(0, "发布任务已推送到任务队列，稍后将在后台执行。")
}

// 生成电子书
func (this *BookController) Generate() {
	identify := this.GetString(":key")

	if !models.NewBook().HasProjectAccess(identify, this.Member.MemberId, conf.BookAdmin) {
		this.JsonResult(1, "您没有操作权限，只有书籍创始人和书籍管理员才有权限")
	}

	book, err := models.NewBook().FindByIdentify(identify)
	if err != nil {
		beego.Error(err)
		this.JsonResult(1, "书籍不存在")
	}

	ebookModel := models.NewEbook()

	// 电子书不是处于完成状态，不允许再添加到电子书生成队列中
	if ok := ebookModel.IsFinish(book.BookId); !ok {
		this.JsonResult(1, "电子书生成任务已在处理中，如需再次生成，请您稍后再试。")
	}

	// 添加到电子书生成队列
	if err = ebookModel.AddToGenerate(book.BookId); err != nil {
		this.JsonResult(1, err.Error())
	}

	this.JsonResult(0, "电子书生成任务已交由后台执行，请您耐心等待。")
}

// 文档排序.
func (this *BookController) SaveSort() {

	identify := this.Ctx.Input.Param(":key")
	if identify == "" {
		this.Abort("404")
	}

	bookId := 0
	if this.Member.IsAdministrator() {
		book, err := models.NewBook().FindByFieldFirst("identify", identify)
		if err != nil {
			beego.Error(err)
		}
		bookId = book.BookId
	} else {
		bookResult, err := models.NewBookResult().FindByIdentify(identify, this.Member.MemberId)
		if err != nil {
			beego.Error("DocumentController.Edit => ", err)
			this.Abort("404")
		}

		if bookResult.RoleId == conf.BookObserver {
			this.JsonResult(6002, "书籍不存在或权限不足")
		}
		bookId = bookResult.BookId
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

	qs := orm.NewOrm().QueryTable("md_documents").Filter("book_id", bookId)
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

// 判断是否具有管理员或管理员以上权限
func (this *BookController) IsPermission() (*models.BookResult, error) {

	identify := this.GetString("identify")

	book, err := models.NewBookResult().FindByIdentify(identify, this.Member.MemberId)
	if err != nil {
		if err == models.ErrPermissionDenied {
			return book, errors.New("权限不足")
		}
		if err == orm.ErrNoRows {
			return book, errors.New("书籍不存在")
		}
		return book, err
	}

	if book.RoleId != conf.BookAdmin && book.RoleId != conf.BookFounder {
		return book, errors.New("权限不足")
	}
	return book, nil
}

// 从github等拉取下载markdown书籍
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
		this.JsonResult(1, "导入失败，只有书籍创建人才有权限导入书籍")
	}
	//GitHub书籍链接
	link := this.GetString("link")
	if strings.ToLower(filepath.Ext(link)) != ".zip" {
		this.JsonResult(1, "只支持拉取zip压缩的markdown书籍")
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

// 从Git仓库拉取书籍
func (this *BookController) GitPull() {
	//处理步骤
	//1、接受上传上来的zip文件，并存放到store/temp目录下
	//2、解压zip到当前目录，然后移除非图片文件
	//3、将文件夹移动到uploads目录下

	identify := this.GetString("identify")

	if !models.NewBook().HasProjectAccess(identify, this.Member.MemberId, conf.BookEditor) {
		this.JsonResult(1, "无操作权限")
	}

	book, _ := models.NewBookResult().FindByIdentify(identify, this.Member.MemberId)
	if book.BookId == 0 {
		this.JsonResult(1, "导入失败，只有书籍创建人才有权限导入书籍")
	}
	//GitHub书籍链接
	link := this.GetString("link")
	go func() {
		folder := "store/" + identify
		err := utils.GitClone(link, folder)
		if err != nil {
			beego.Error(err.Error())
		}
		this.loadByFolder(book.BookId, identify, folder)
	}()

	this.JsonResult(0, "提交成功，请耐心等待。")
}

// 上传书籍
func (this *BookController) UploadProject() {
	//处理步骤
	//1、接受上传上来的zip文件，并存放到store/temp目录下
	//2、解压zip到当前目录，然后移除非图片文件
	//3、将文件夹移动到uploads目录下

	identify := this.GetString("identify")

	if !models.NewBook().HasProjectAccess(identify, this.Member.MemberId, conf.BookEditor) {
		this.JsonResult(1, "无操作权限")
	}

	book, _ := models.NewBookResult().FindByIdentify(identify, this.Member.MemberId)
	if book.BookId == 0 {
		this.JsonResult(1, "书籍不存在")
	}

	f, h, err := this.GetFile("zipfile")
	if err != nil {
		this.JsonResult(1, err.Error())
	}
	defer f.Close()
	if strings.ToLower(filepath.Ext(h.Filename)) != ".zip" && strings.ToLower(filepath.Ext(h.Filename)) != ".epub" {
		this.JsonResult(1, "请上传指定格式文件")
	}
	tmpFile := "store/" + identify + ".zip" //保存的文件名
	if err := this.SaveToFile("zipfile", tmpFile); err == nil {
		go this.unzipToData(book.BookId, identify, tmpFile, h.Filename)
	} else {
		beego.Error(err.Error())
	}
	this.JsonResult(0, "上传成功")
}

// 将zip压缩文件解压并录入数据库
// @param            book_id             书籍id(其实有想不标识了可以不要这个的，但是这里的书籍标识只做目录)
// @param            identify            书籍标识
// @param            zipfile             压缩文件
// @param            originFilename      上传文件的原始文件名
func (this *BookController) unzipToData(bookId int, identify, zipFile, originFilename string) {

	//说明：
	//OSS中的图片存储规则为"projects/$identify/书籍中图片原路径"
	//本地存储规则为"uploads/projects/$identify/书籍中图片原路径"

	projectRoot := "" //书籍根目录

	//解压目录
	unzipPath := "store/" + identify

	//如果存在相同目录，则率先移除
	if err := os.RemoveAll(unzipPath); err != nil {
		beego.Error(err.Error())
	}
	os.MkdirAll(unzipPath, os.ModePerm)

	imgMap := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".bmp": true, ".svg": true, ".webp": true}

	defer func() {
		os.Remove(zipFile)      //最后删除上传的临时文件
		os.RemoveAll(unzipPath) //删除解压后的文件夹
	}()

	//注意：这里的prefix必须是判断是否是GitHub之前的prefix
	if err := ziptil.Unzip(zipFile, unzipPath); err != nil {
		beego.Error("解压失败", zipFile, err.Error())
		return
	}

	//读取文件，把图片文档录入oss
	if files, err := filetil.ScanFiles(unzipPath); err == nil {
		projectRoot = this.getProjectRoot(files)

		this.fixFileLinks(projectRoot, identify)

		ModelStore := new(models.DocumentStore)
		//文档对应的标识
		for _, file := range files {
			if !file.IsDir {
				ext := strings.ToLower(filepath.Ext(file.Path))
				if ok, _ := imgMap[ext]; ok { //图片，录入oss
					switch utils.StoreType {
					case utils.StoreOss:
						if err := store.ModelStoreOss.MoveToOss(file.Path, filepath.Join("projects/"+identify, strings.TrimPrefix(file.Path, projectRoot)), false, false); err != nil {
							beego.Error(err)
						}
					case utils.StoreLocal:
						if err := store.ModelStoreLocal.MoveToStore(file.Path, filepath.Join("uploads/projects/"+identify, strings.TrimPrefix(file.Path, projectRoot))); err != nil {
							beego.Error(err)
						}
					}
				} else if ext == ".md" || ext == ".markdown" || ext == ".html" { //markdown文档，提取文档内容，录入数据库
					doc := new(models.Document)
					var mdcont string
					var htmlStr string
					if b, err := ioutil.ReadFile(file.Path); err == nil {
						mdcont = strings.TrimSpace(string(b))
						htmlStr = mdtil.Md2html(mdcont)

						if !strings.HasPrefix(mdcont, "[TOC]") {
							mdcont = "[TOC]\r\n\r\n" + mdcont
						}
						doc.DocumentName = utils.ParseTitleFromMdHtml(htmlStr)
						doc.BookId = bookId
						//文档标识
						doc.Identify = strings.Replace(strings.Trim(strings.TrimPrefix(file.Path, projectRoot), "/"), "/", "-", -1)
						doc.Identify = strings.Replace(doc.Identify, ")", "", -1)
						doc.MemberId = this.Member.MemberId
						doc.OrderSort = 1
						if strings.HasSuffix(strings.ToLower(file.Name), "summary.md") {
							doc.OrderSort = 0
						}
						if strings.HasSuffix(strings.ToLower(file.Name), "summary.html") {
							mdcont += "<bookstack-summary></bookstack-summary>"
							// 生成带$的文档标识，阅读BaseController.go代码可知，
							// 要使用summary.md的排序功能，必须在链接中带上符号$
							mdcont = strings.Replace(mdcont, "](", "]($", -1)
							// 去掉可能存在的url编码的右括号，否则在url译码后会与markdown语法混淆
							mdcont = strings.Replace(mdcont, "%29", "", -1)
							mdcont, _ = url.QueryUnescape(mdcont)
							doc.OrderSort = 0
							doc.Identify = "summary.md"
						}
						if docId, err := doc.InsertOrUpdate(); err == nil {
							if err := ModelStore.InsertOrUpdate(models.DocumentStore{
								DocumentId: int(docId),
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

func (this *BookController) loadByFolder(bookId int, identify, folder string) {
	//说明：

	imgMap := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".bmp": true, ".svg": true, ".webp": true}

	defer func() {
		os.RemoveAll(folder) //删除解压后的文件夹
	}()

	//注意：这里的prefix必须是判断是否是GitHub之前的prefix

	//读取文件，把图片文档录入oss
	files, err := filetil.ScanFiles(folder)
	if err != nil {
		beego.Error(err)
		return
	}

	this.fixFileLinks(folder, identify)

	ModelStore := new(models.DocumentStore)

	//文档对应的标识
	for _, file := range files {
		if !file.IsDir {
			ext := strings.ToLower(filepath.Ext(file.Path))

			if ok, _ := imgMap[ext]; ok { //图片，录入oss
				switch utils.StoreType {
				case utils.StoreOss:
					if err := store.ModelStoreOss.MoveToOss(file.Path, "projects/"+identify+strings.TrimPrefix(file.Path, folder), false, false); err != nil {
						beego.Error(err)
					}
				case utils.StoreLocal:
					if err := store.ModelStoreLocal.MoveToStore(file.Path, "uploads/projects/"+identify+strings.TrimPrefix(file.Path, folder)); err != nil {
						beego.Error(err)
					}
				}
			} else if ext == ".md" || ext == ".markdown" { //markdown文档，提取文档内容，录入数据库
				doc := new(models.Document)
				if b, err := ioutil.ReadFile(file.Path); err == nil {
					mdCont := strings.TrimSpace(string(b))
					if !strings.HasPrefix(mdCont, "[TOC]") {
						mdCont = "[TOC]\r\n\r\n" + mdCont
					}
					htmlStr := mdtil.Md2html(mdCont)
					doc.DocumentName = utils.ParseTitleFromMdHtml(htmlStr)
					doc.BookId = bookId
					//文档标识
					doc.Identify = strings.Replace(strings.Trim(strings.TrimPrefix(file.Path, folder), "/"), "/", "-", -1)
					doc.MemberId = this.Member.MemberId
					doc.OrderSort = 1
					if strings.HasSuffix(strings.ToLower(file.Name), "summary.md") {
						doc.OrderSort = 0
					}
					if docId, err := doc.InsertOrUpdate(); err == nil {
						if err := ModelStore.InsertOrUpdate(models.DocumentStore{
							DocumentId: int(docId),
							Markdown:   mdCont,
							Content:    "",
						}, "markdown", "content"); err != nil {
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

// 获取书籍的根目录
func (this *BookController) getProjectRoot(fl []filetil.FileList) (root string) {
	var strs []string
	for _, f := range fl {
		if !f.IsDir {
			strs = append(strs, f.Path)
		}
	}
	return utils.LongestCommonPrefix(strs)
}

// 查找并替换markdown文件中的路径，把图片链接替换成url的相对路径，把文档间的链接替换成【$+文档标识链接】
func (this *BookController) fixFileLinks(projectRoot string, identify string) {
	imgBaseUrl := "/uploads/projects/" + identify
	switch utils.StoreType {
	case utils.StoreLocal:
		imgBaseUrl = "/uploads/projects/" + identify
	case utils.StoreOss:
		//imgBaseUrl = this.BaseController.OssDomain + "/projects/" + identify
		imgBaseUrl = "/projects/" + identify
	}
	files, _ := filetil.ScanFiles(projectRoot)
	for _, file := range files {
		ext := strings.ToLower(filepath.Ext(file.Path))
		if !(ext == ".md" || ext == ".markdown" || ext == ".html" || ext == ".xhtml") {
			continue
		}

		//mdb ==> markdown byte
		mdb, _ := ioutil.ReadFile(file.Path)
		mdCont := string(mdb)
		if ext == ".html" || ext == ".xhtml" {
			mdCont = html2md.Convert(mdCont)
		}

		basePath := filepath.Dir(file.Path)
		basePath = strings.Trim(strings.Replace(basePath, "\\", "/", -1), "/")
		basePathSlice := strings.Split(basePath, "/")
		l := len(basePathSlice)
		b, _ := ioutil.ReadFile(file.Path)
		output := blackfriday.Run(b)
		doc, _ := goquery.NewDocumentFromReader(bytes.NewReader(output))

		//图片链接处理
		doc.Find("img").Each(func(i int, selection *goquery.Selection) {
			//非http://、// 和 https:// 开头的图片地址，即是相对地址
			src, ok := selection.Attr("src")
			lowerSrc := strings.ToLower(src)
			if ok &&
				!strings.HasPrefix(lowerSrc, "http://") &&
				!strings.HasPrefix(lowerSrc, "https://") {
				newSrc := src //默认为旧地址
				if strings.HasPrefix(lowerSrc, "//") {
					newSrc = "https:" + newSrc
				} else {
					if cnt := strings.Count(src, "../"); cnt < l { //以或者"../"开头的路径
						newSrc = strings.Join(basePathSlice[0:l-cnt], "/") + "/" + strings.TrimLeft(src, "./")
					}
					newSrc = imgBaseUrl + "/" + strings.TrimLeft(strings.TrimPrefix(strings.TrimLeft(newSrc, "./"), projectRoot), "/")
				}
				mdCont = strings.Replace(mdCont, src, newSrc, -1)
			}
		})

		//a标签链接处理。要注意判断有锚点的情况
		doc.Find("a").Each(func(i int, selection *goquery.Selection) {
			href, ok := selection.Attr("href")
			lowerHref := strings.TrimSpace(strings.ToLower(href))
			// 链接存在，且不以 // 、 http、https、mailto 开头
			if ok &&
				!strings.HasPrefix(lowerHref, "//") &&
				!strings.HasPrefix(lowerHref, "http://") &&
				!strings.HasPrefix(lowerHref, "https://") &&
				!strings.HasPrefix(lowerHref, "mailto:") &&
				!strings.HasPrefix(lowerHref, "#") {
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

// 给书籍打分
func (this *BookController) Score() {
	bookId, _ := this.GetInt(":id")
	if bookId == 0 {
		this.JsonResult(1, "文档不存在")
	}

	score, _ := this.GetInt("score")
	if uid := this.Member.MemberId; uid > 0 {
		if err := new(models.Score).AddScore(uid, bookId, score); err != nil {
			this.JsonResult(1, err.Error())
		}
		this.JsonResult(0, "感谢您给当前文档打分")
	}
	this.JsonResult(1, "给文档打分失败，请先登录再操作")
}

// 添加评论
func (this *BookController) Comment() {
	if this.Member.MemberId == 0 {
		this.JsonResult(1, "请先登录在评论")
	}
	content := this.GetString("content")
	if l := len(content); l < 5 || l > 255 {
		this.JsonResult(1, "评论内容限 5 - 255 个字符")
	}
	bookId, _ := this.GetInt(":id")
	pid, _ := this.GetInt("pid")
	docId, _ := this.GetInt("doc_id")
	if bookId > 0 {
		if err := new(models.Comments).AddComments(this.Member.MemberId, bookId, pid, docId, content); err != nil {
			this.JsonResult(1, err.Error())
		}
		this.JsonResult(0, "评论成功")
	}
	this.JsonResult(1, "书籍不存在")
}

// ExportMarkdown 将书籍导出为markdown
// 注意：系统管理员和书籍参与者有权限导出
func (this *BookController) Export2Markdown() {
	identify := this.GetString("identify")
	if this.Member.MemberId == 0 {
		this.JsonResult(1, "请先登录")
	}
	if !this.Member.IsAdministrator() {
		if _, err := models.NewBookResult().FindByIdentify(identify, this.Member.MemberId); err != nil {
			this.JsonResult(1, "无操作权限")
		}
	}
	path, err := models.NewBook().Export2Markdown(identify)
	if err != nil {
		this.JsonResult(1, err.Error())
	}
	defer func() {
		os.Remove(path)
	}()
	attchmentName := filepath.Base(path)
	if book, _ := models.NewBook().FindByIdentify(identify, "book_name", "book_id"); book != nil && book.BookId > 0 {
		attchmentName = book.BookName + ".zip"
	}
	this.Ctx.Output.Download(strings.TrimLeft(path, "./"), attchmentName)
}
