package api

import (
	"bytes"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/TruthHun/BookStack/conf"
	"github.com/TruthHun/gotil/cryptil"
	"github.com/TruthHun/gotil/util"
	"github.com/unknwon/com"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/TruthHun/BookStack/models"
	"github.com/TruthHun/BookStack/utils"
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/orm"
)

// 不登录也能调用的接口放这里
type CommonController struct {
	BaseController
}

// [OK]
func (this *CommonController) Login() {
	username := this.GetString("username") //username or email
	password := this.GetString("password")
	member, err := models.NewMember().GetByUsername(username)

	if err != nil {
		if err == orm.ErrNoRows {
			this.Response(http.StatusBadRequest, messageUsernameOrPasswordError)
		}
		beego.Error(err)
		this.Response(http.StatusInternalServerError, messageInternalServerError)
	}
	if err != nil {
		beego.Error(err)
		this.Response(http.StatusInternalServerError, messageInternalServerError)
	}
	if ok, _ := utils.PasswordVerify(member.Password, password); !ok {
		beego.Error(err)
		this.Response(http.StatusBadRequest, messageUsernameOrPasswordError)
	}

	this.login(member)
}

// 【OK】
func (this *CommonController) login(member models.Member) {
	var user APIUser
	utils.CopyObject(&member, &user)
	user.Uid = member.MemberId
	user.Token = cryptil.Md5Crypt(fmt.Sprintf("%v-%v", time.Now().Unix(), util.InterfaceToJson(user)))
	err := models.NewAuth().Insert(user.Token, user.Uid)
	if err != nil {
		beego.Error(err.Error())
		this.Response(http.StatusInternalServerError, messageInternalServerError)
	}
	user.Avatar = this.completeLink(user.Avatar)
	this.Response(http.StatusOK, messageSuccess, user)
}

// 【OK】
func (this *CommonController) Register() {
	var register APIRegister
	err := this.ParseForm(&register)
	if err != nil {
		beego.Error(err.Error())
		this.Response(http.StatusBadRequest, messageBadRequest)
	}

	if !com.IsEmail(register.Email) {
		this.Response(http.StatusBadRequest, messageEmailError)
	}

	if register.Account == "" || register.Nickname == "" || register.Password == "" || register.RePassword == "" {
		this.Response(http.StatusBadRequest, messageRequiredInput)
	}

	if register.Password != register.RePassword {
		this.Response(http.StatusBadRequest, messageNotEqualTwicePassword)
	}
	var member models.Member

	utils.CopyObject(&register, &member)

	member.Role = conf.MemberGeneralRole
	member.Avatar = conf.GetDefaultAvatar()
	member.CreateAt = int(time.Now().Unix())
	member.Status = 0
	if err = member.Add(); err != nil {
		this.Response(http.StatusBadRequest, err.Error())
	}

	this.login(member)
}

func (this *BaseController) About() {

}

func (this *BaseController) UserInfo() {

}

func (this *BaseController) UserStar() {

}

func (this *BaseController) UserFollow() {
	this.getFansOrFollow(false)
}

func (this *BaseController) UserFans() {
	this.getFansOrFollow(true)
}

func (this *BaseController) getFansOrFollow(isGetFans bool) {
	page, _ := this.GetInt("page", 1)
	size, _ := this.GetInt("size", 10)
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 10
	}
	if size > maxPageSize {
		size = maxPageSize
	}

	uid, _ := this.GetInt("uid")
	if uid <= 0 {
		uid = this.isLogin()
	}
	if uid <= 0 {
		this.Response(http.StatusBadRequest, messageBadRequest)
	}
	var (
		fans       []models.FansResult
		totalCount int64
		err        error
		model      = new(models.Fans)
	)

	if isGetFans {
		fans, totalCount, err = model.GetFansList(uid, page, size)
	} else {
		fans, totalCount, err = model.GetFollowList(uid, page, size)
	}
	if err != nil {
		beego.Error(err.Error())
		this.Response(http.StatusInternalServerError, messageInternalServerError)
	}

	var users []APIUser
	for _, item := range fans {
		user := &APIUser{}
		utils.CopyObject(&item, user)
		user.Avatar = this.completeLink(user.Avatar)
		users = append(users, *user)
	}

	data := map[string]interface{}{"total": totalCount}
	if len(users) > 0 {
		data["fans"] = users
	}
	this.Response(http.StatusOK, messageSuccess, data)
}

// 如果不传用户id，则表示查询当前登录的用户发布的书籍
func (this *BaseController) UserReleaseBook() {
	uid, _ := this.GetInt("uid")
	if uid <= 0 {
		if login := this.isLogin(); login > 0 {
			uid = login
		} else {
			this.Response(http.StatusBadRequest, messageBadRequest)
		}
	}

	page, _ := this.GetInt("page")
	if page <= 0 {
		page = 1
	}
	size := 10

	res, totalCount, err := models.NewBook().FindToPager(page, size, uid, 0)
	if err != nil {
		this.Response(http.StatusInternalServerError, messageInternalServerError)
	}

	var books []APIBook
	for _, item := range res {
		book := &APIBook{}
		utils.CopyObject(item, book)
		book.Cover = this.completeLink(book.Cover)
		books = append(books, *book)
	}
	data := map[string]interface{}{"total": totalCount}

	if len(books) > 0 {
		data["books"] = books
	}

	this.Response(http.StatusOK, messageSuccess, data)
}

func (this *CommonController) TODO() {
	this.Response(http.StatusOK, "TODO")
}

func (this *BaseController) FindPassword() {

}

// [OK]
func (this *BaseController) SearchBook() {
	wd := this.GetString("wd")
	if wd == "" {
		this.Response(http.StatusBadRequest, messageBadRequest)
	}

	var (
		page, _  = this.GetInt("page", 1)
		size, _  = this.GetInt("size", 10)
		ids      []int
		total    int
		apiBooks []APIBook
		book     APIBook
	)

	if size <= 0 {
		size = 10
	}

	if size > maxPageSize {
		size = maxPageSize
	}

	client := models.NewElasticSearchClient()

	if client.On { // elasticsearch 进行全文搜索
		result, err := models.NewElasticSearchClient().Search(wd, page, size, false)
		if err != nil {
			beego.Error(err.Error())
			this.Response(http.StatusInternalServerError, messageInternalServerError)
		}

		total = result.Hits.Total
		for _, item := range result.Hits.Hits {
			ids = append(ids, item.Source.Id)
		}

	} else { //MySQL like 查询
		books, count, err := models.NewBook().SearchBook(wd, page, size)
		if err != nil {
			beego.Error(err.Error())
			this.Response(http.StatusInternalServerError, messageInternalServerError)
		}
		total = count
		for _, book := range books {
			ids = append(ids, book.BookId)
		}
	}

	data := map[string]interface{}{"total": total}

	if len(ids) > 0 {
		books, _ := models.NewBook().GetBooksById(ids)
		for _, item := range books {
			utils.CopyObject(&item, &book)
			book.Cover = this.completeLink(book.Cover)
			apiBooks = append(apiBooks, book)
		}
		data["result"] = apiBooks
	}

	this.Response(http.StatusOK, messageSuccess, data)
}

// [OK]
func (this *BaseController) SearchDoc() {
	wd := this.GetString("wd")
	if wd == "" {
		this.Response(http.StatusBadRequest, messageBadRequest)
	}

	var (
		page, _   = this.GetInt("page", 1)
		size      = 10
		ids       []int
		total     int
		docs      []APIDoc
		doc       APIDoc
		bookId, _ = this.GetInt("book_id")
	)

	if bookId > 0 {
		page = 1
		size = 1000
	}

	client := models.NewElasticSearchClient()

	if client.On { // elasticsearch 进行全文搜索
		result, err := models.NewElasticSearchClient().Search(wd, page, size, true, bookId)
		if err != nil {
			beego.Error(err.Error())
			this.Response(http.StatusInternalServerError, messageInternalServerError)
		}

		total = result.Hits.Total
		for _, item := range result.Hits.Hits {
			ids = append(ids, item.Source.Id)
		}

	} else { //MySQL like 查询
		result, count, err := models.NewDocumentSearchResult().SearchDocument(wd, bookId, page, size)
		if err != nil {
			beego.Error(err.Error())
			this.Response(http.StatusInternalServerError, messageInternalServerError)
		}
		total = count
		for _, book := range result {
			ids = append(ids, book.BookId)
		}
	}

	data := map[string]interface{}{"total": total}

	if len(ids) > 0 {
		var result []models.DocResult
		if bookId > 0 {
			result, _ = models.NewDocumentSearchResult().GetDocsById(ids, true)
		} else {
			result, _ = models.NewDocumentSearchResult().GetDocsById(ids)
		}
		for _, item := range result {
			utils.CopyObject(&item, &doc)
			if len(doc.Release) > 0 {
				doc.Release = beego.Substr(utils.GetTextFromHtml(doc.Release), 0, 150) + "..."
			}
			docs = append(docs, doc)
		}
		data["result"] = docs
	}
	this.Response(http.StatusOK, messageSuccess, data)
}

func (this *CommonController) Categories() {

	model := models.NewCategory()

	pid, err := this.GetInt("pid")
	if err != nil {
		pid = -1
	}

	categories, _ := model.GetCates(pid, 1)
	for idx, category := range categories {
		if category.Icon != "" {
			category.Icon = this.completeLink(category.Icon)
			categories[idx] = category
		}
	}

	this.Response(http.StatusOK, messageSuccess, categories)
}

// 【OK】
func (this *BaseController) BookInfo() {
	var (
		book    *models.Book
		err     error
		apiBook APIBook
	)

	identify := this.GetString("identify")
	model := models.NewBook()
	id, _ := strconv.Atoi(identify)

	if id > 0 {
		book, err = model.Find(id)
	} else {
		book, err = model.FindByIdentify(identify)
	}
	if err != nil {
		beego.Error(err.Error())
	}

	if book.BookId == 0 || (book.PrivatelyOwned == 1 && this.isLogin() != book.MemberId) {
		this.Response(http.StatusNotFound, messageNotFound)
	}

	utils.CopyObject(book, &apiBook)

	apiBook.Cover = this.completeLink(apiBook.Cover)
	apiBook.User = models.NewMember().GetNicknameByUid(book.MemberId)

	this.Response(http.StatusOK, messageSuccess, apiBook)
}

func (this *BaseController) BookContent() {

}

// TODO: 根据用户登录情况，判断书籍是私有还是公有，并再决定是否显示
// 返回用户对该章节是否已读
func (this *BaseController) BookMenu() {
	var (
		book models.Book
		o    = orm.NewOrm()
	)
	identify := this.GetString("identify")
	q := o.QueryTable(book)
	cols := []string{"book_id", "privately_owned", "member_id"}
	if id, _ := strconv.Atoi(identify); id > 0 {
		q.Filter("book_id", id).One(&book, cols...)
	} else {
		q.Filter("identify", identify).One(&book, cols...)
	}

	if book.BookId == 0 || (book.PrivatelyOwned == 1 && this.isLogin() != book.MemberId) {
		this.Response(http.StatusNotFound, messageNotFound)
	}

	docsOri, err := models.NewDocument().FindListByBookId(book.BookId, true)
	if err != nil {
		beego.Error(err.Error())
		this.Response(http.StatusInternalServerError, messageInternalServerError)
	}

	var (
		docs []APIDoc
		doc  APIDoc
	)

	uid := this.isLogin()
	readed := make(map[int]bool)
	if uid > 0 {
		lists, _, _ := new(models.ReadRecord).List(uid, book.BookId)
		for _, item := range lists {
			readed[item.DocId] = true
		}
	}

	for _, item := range docsOri {
		utils.CopyObject(item, &doc)
		if _, ok := readed[doc.DocumentId]; ok {
			doc.Readed = true
		}
		docs = append(docs, doc)
	}

	this.Response(http.StatusOK, messageSuccess, docs)
}

// 【OK】
func (this *CommonController) BookLists() {
	sort := this.GetString("sort", "new") // new、recommend、hot、pin
	page, _ := this.GetInt("page", 1)
	cid, _ := this.GetInt("cid")
	lang := this.GetString("lang")
	pageSize, _ := this.GetInt("size", 10)

	if page <= 0 {
		page = 1
	}

	if page <= 0 {
		page = 10
	}

	if pageSize > 20 {
		pageSize = 20
	}

	model := models.NewBook()

	fields := []string{"book_id", "book_name", "identify", "order_index", "description", "label", "doc_count",
		"vcnt", "star", "lang", "cover", "score", "cnt_score", "cnt_comment", "modify_time", "create_time",
	}

	books, total, _ := model.HomeData(page, pageSize, models.BookOrder(sort), lang, cid, fields...)
	data := map[string]interface{}{"total": total}
	if len(books) > 0 {
		var lists []APIBook
		var list APIBook

		for _, book := range books {
			book.Cover = this.completeLink(book.Cover)
			if book.Lang == "" {
				book.Lang = ""
			}
			utils.CopyObject(&book, &list)
			lists = append(lists, list)
		}
		data["books"] = lists
	}
	this.Response(http.StatusOK, messageSuccess, data)
}

func (this *CommonController) Read() {
	identify := this.GetString("identify")
	slice := strings.Split(identify, "/")
	if len(slice) != 2 {
		this.Response(http.StatusBadRequest, messageBadRequest)
	}
	bookIdentify, docIdentify := slice[0], slice[1]
	if bookIdentify == "" || docIdentify == "" {
		this.Response(http.StatusBadRequest, messageBadRequest)
	}

	// 1. 如果书籍是私有的，则必须是作者本人才能阅读，否则无法阅读
	book := models.NewBook()
	bookId, _ := strconv.Atoi(bookIdentify)
	cols := []string{"book_id", "privately_owned", "member_id"}
	if bookId > 0 {
		book, _ = book.Find(bookId, cols...)
	} else {
		book, _ = book.FindByIdentify(bookIdentify, cols...)
	}

	if book.PrivatelyOwned == 1 && this.isLogin() != book.MemberId {
		this.Response(http.StatusNotFound, messageNotFound)
	}

	doc := models.NewDocument()
	docId, _ := strconv.Atoi(docIdentify)
	if docId > 0 {
		doc, _ = doc.Find(docId)
	} else {
		doc, _ = doc.FindByBookIdAndDocIdentify(book.BookId, docIdentify)
	}

	if doc.DocumentId == 0 {
		this.Response(http.StatusNotFound, messageNotFound)
	}

	var err error

	// 文档阅读人次+1
	if err = models.SetIncreAndDecre("md_documents", "vcnt",
		fmt.Sprintf("document_id=%v", doc.DocumentId),
		true, 1,
	); err != nil {
		beego.Error(err.Error())
	}

	//项目阅读人次+1
	if err = models.SetIncreAndDecre("md_books", "vcnt",
		fmt.Sprintf("book_id=%v", doc.BookId),
		true, 1,
	); err != nil {
		beego.Error(err.Error())
	}

	// 增加用户阅读记录
	if this.isLogin() > 0 {
		if err = new(models.ReadRecord).Add(doc.DocumentId, this.isLogin()); err != nil {
			beego.Error(err.Error())
		}
	}

	// 图片链接地址补全
	if doc.Release != "" {
		query, err := goquery.NewDocumentFromReader(bytes.NewBufferString(doc.Release))
		if err != nil {
			beego.Error(err)
		} else {
			query.Find("img").Each(func(i int, contentSelection *goquery.Selection) {
				if src, ok := contentSelection.Attr("src"); ok {
					contentSelection.SetAttr("src", this.completeLink(src))
				}
				if alt, _ := contentSelection.Attr("alt"); alt == "" {
					contentSelection.SetAttr("alt", doc.DocumentName+" - 图"+fmt.Sprint(i+1))
				}
			})
			html, err := query.Find("body").Html()
			if err != nil {
				beego.Error(err)
			} else {
				doc.Release = html
			}
		}
	}

	// TODO: 可能还需要对内容中一些无用的HTML标签或属性进行移除处理，如提出 alt、title 等标签属性
	var apiDoc APIDoc

	utils.CopyObject(doc, &apiDoc)

	this.Response(http.StatusOK, messageSuccess, apiDoc)
}

func (this *BaseController) Progress() {

}

func (this *BaseController) Bookmarks() {

}

// 【OK】
func (this *CommonController) Banners() {
	t := this.GetString("type", "wechat")
	banners, _ := models.NewBanner().Lists(t)
	this.Response(http.StatusOK, messageSuccess, banners)
}

func (this *CommonController) Download() {
	identify := this.GetString("identify")
	if identify == "" {
		this.Response(http.StatusBadRequest, messageBadRequest)
	}

	id, _ := strconv.Atoi(identify)

	book := models.NewBook()
	q := orm.NewOrm().QueryTable(book)
	if id > 0 {
		q.Filter("book_id", id).One(book)
	} else {
		q.Filter("identify", identify).One(book)
	}

	if book.BookId == 0 || book.GenerateTime.Unix() < book.ReleaseTime.Unix() {
		this.Response(http.StatusNotFound, messageNotFound)
	}

	if book.PrivatelyOwned == 1 && this.isLogin() != book.MemberId {
		this.Response(http.StatusNotFound, messageNotFound)
	}

	format := fmt.Sprintf("projects/%v/books/%v", book.Identify, book.GenerateTime.Unix())

	data := map[string]string{
		"pdf":  this.completeLink(format + ".pdf"),
		"mobi": this.completeLink(format + ".mobi"),
		"epub": this.completeLink(format + ".epub"),
	}

	this.Response(http.StatusOK, messageSuccess, data)
}

func (this *CommonController) Bookshelf() {
	uid, _ := this.GetInt("uid")
	if uid <= 0 {
		uid = this.isLogin()
	}

	if uid <= 0 {
		this.Response(http.StatusBadRequest, messageBadRequest)
	}

	size := 1000

	total, res, err := new(models.Star).List(uid, 1, size)
	if err != nil {
		beego.Error(err.Error())
		this.Response(http.StatusInternalServerError, messageInternalServerError)
	}

	var (
		books   []APIBook
		booksId []int
	)
	for _, item := range res {
		book := &APIBook{}
		utils.CopyObject(&item, book)
		booksId = append(booksId, book.BookId)
		book.Cover = this.completeLink(book.Cover)
		books = append(books, *book)
	}

	read := new(models.ReadRecord).BooksProgress(uid, booksId...)

	this.Response(http.StatusOK, messageSuccess, map[string]interface{}{"books": books, "total": total, "readed": read})
}

// 查询书籍评论
func (this *CommonController) GetComments() {
	page, _ := this.GetInt("page", 1)
	if page <= 0 {
		page = 1
	}
	size, _ := this.GetInt("size", 10)
	if size <= 0 {
		size = 10
	}

	if size > maxPageSize {
		size = maxPageSize
	}

	bookId, _ := this.GetInt("book_id")
	if bookId <= 0 {
		this.Response(http.StatusBadRequest, messageBadRequest)
	}

	uid := this.isLogin()
	myScore := 0
	if uid > 0 {
		myScore = new(models.Score).BookScoreByUid(uid, bookId) / 10
	}
	comments, err := new(models.Comments).BookComments(page, size, bookId)
	if err != nil {
		beego.Error(err.Error())
	}
	data := map[string]interface{}{
		"my_score": myScore,
		"comments": []string{},
	}

	if len(comments) > 0 {
		data["comments"] = comments
	}

	this.Response(http.StatusOK, messageSuccess, data)
}
