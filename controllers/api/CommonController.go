package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/TruthHun/html2json/html2json"

	"github.com/TruthHun/BookStack/models/store"
	"github.com/TruthHun/BookStack/oauth"

	"github.com/PuerkitoBio/goquery"
	"github.com/TruthHun/BookStack/conf"
	"github.com/TruthHun/gotil/cryptil"
	"github.com/TruthHun/gotil/util"
	"github.com/unknwon/com"

	"github.com/TruthHun/BookStack/models"
	"github.com/TruthHun/BookStack/utils"
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/orm"
)

// 不登录也能调用的接口放这里
type CommonController struct {
	BaseController
}

// 普通登录
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

func (this *CommonController) LoginedBindWechat() {
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

func (this *CommonController) LoginBindWechat() {
	form := &WechatBindForm{}
	err := this.ParseForm(form)
	if err != nil {
		this.Response(http.StatusBadRequest, err.Error())
	}

	if form.Username == "" || form.Password == "" || form.Sess == "" {
		this.Response(http.StatusBadRequest, messageBadRequest)
	}

	we, err := models.NewWechat().GetUserBySess(form.Sess)
	if err != nil && err != orm.ErrNoRows {
		beego.Error(err)
		this.Response(http.StatusInternalServerError, messageInternalServerError)
	}

	if we.Id == 0 {
		this.Response(http.StatusBadRequest, messageBadRequest)
	}

	member := &models.Member{}
	if we.MemberId > 0 { // 已经绑定过
		member, _ = member.Find(we.MemberId)
		this.login(*member)
	}

	if form.Nickname != "" || form.Email != "" || form.RePassword != "" { //只要有其中任意一项不为空，则表示新注册
		if form.Password != form.RePassword {
			this.Response(http.StatusBadRequest, messageNotEqualTwicePassword)
		}
		if form.Nickname == "" || form.Email == "" || form.RePassword == "" {
			this.Response(http.StatusBadRequest, messageBadRequest)
		}
		if we.AvatarURL == "" {
			we.AvatarURL = conf.GetDefaultAvatar()
		}
		member = &models.Member{
			Account:  form.Username,
			Password: form.Password,
			Nickname: form.Nickname,
			Avatar:   we.AvatarURL,
			Email:    form.Email,
			Status:   0,
			Role:     conf.MemberGeneralRole,
		}
		err = member.Add()
		if err != nil {
			this.Response(http.StatusBadRequest, err.Error())
		}
		// 执行绑定
		we.Bind(we.Openid, member.MemberId)
	} else {
		*member, _ = models.NewMember().GetByUsername(form.Username)

		if member.MemberId == 0 {
			this.Response(http.StatusBadRequest, "您要绑定的用户不存在")
		}

		if ok, _ := utils.PasswordVerify(member.Password, form.Password); !ok {
			beego.Error(err)
			this.Response(http.StatusBadRequest, messageUsernameOrPasswordError)
		}

		we.Bind(we.Openid, member.MemberId)
	}
	this.login(*member)
}

// 微信登录
func (this *CommonController) LoginByWechat() {
	form := &WechatForm{}
	err := this.ParseForm(form)
	if err != nil {
		this.Response(http.StatusBadRequest, err.Error())
	}
	appId, secret := beego.AppConfig.String("appId"), beego.AppConfig.String("appSecret")
	if form.Code == "" || form.UserInfo == "" {
		this.Response(http.StatusBadRequest, messageBadRequest)
	}
	sess, err := oauth.GetWechatSessKey(appId, secret, form.Code)
	if err != nil {
		this.Response(http.StatusBadRequest, err.Error())
	}
	if sess.ErrMsg != "" {
		this.Response(http.StatusBadRequest, sess.ErrMsg)
	}
	m := models.NewWechat()
	user, err := m.GetUserByOpenid(sess.OpenId)
	if user.MemberId > 0 { // 之前已经绑定过注册账号，直接登录成功
		member, _ := models.NewMember().Find(user.MemberId)
		this.login(*member)
	}

	wechatUser := &oauth.WechatUser{}
	if err = json.Unmarshal([]byte(form.UserInfo), wechatUser); err != nil {
		this.Response(http.StatusBadRequest, err.Error())
	}
	m = &models.Wechat{Openid: sess.OpenId, AvatarURL: wechatUser.AvatarURL, Nickname: wechatUser.NickName, SessKey: sess.SessionKey, Unionid: sess.UnionId}
	if err = m.Insert(); err != nil {
		this.Response(http.StatusBadRequest, err.Error())
	}
	this.Response(http.StatusUnauthorized, "未绑定账号，请先绑定账号", map[string]string{"sess": sess.SessionKey})
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
	user.Avatar = this.completeLink(utils.ShowImg(user.Avatar, "avatar"))
	data := map[string]interface{}{"user": user}
	this.Response(http.StatusOK, messageSuccess, data)
}

func (this *CommonController) GetUserMoreInfo() {
	uid, _ := this.GetInt("uid")
	if uid <= 0 {
		this.Response(http.StatusBadRequest, messageBadRequest)
	}
	cols := []string{"create_time", "total_reading_time", "total_sign", "total_continuous_sign", "history_total_continuous_sign"}
	m, err := models.NewMember().Find(uid, cols...)
	if err != nil {
		this.Response(http.StatusInternalServerError, messageInternalServerError)
	}
	rt := models.NewReadingTime()
	u := UserMoreInfo{
		MemberId:              uid,
		SignedAt:              models.NewSign().LatestSignTime(uid),
		CreatedAt:             int(m.CreateTime.Unix()),
		TotalSign:             m.TotalSign,
		TotalContinuousSign:   m.TotalContinuousSign,
		HistoryContinuousSign: m.HistoryTotalContinuousSign,
		TodayReading:          rt.GetReadingTime(uid, models.PeriodDay),
		MonthReading:          rt.GetReadingTime(uid, models.PeriodMonth),
		TotalReading:          m.TotalReadingTime,
	}
	this.Response(http.StatusOK, messageSuccess, map[string]interface{}{"info": u})
}

// 【OK】
func (this *CommonController) Register() {
	rl := models.NewRegLimit()
	realIP := utils.GetIP(this.Ctx, rl.RealIPField)
	if this.Ctx.Request.Method == http.MethodGet {
		this.Response(http.StatusOK, "Register", map[string]interface{}{"IP": realIP, "Request": this.Ctx.Request.Header})
	}

	allowHour, allowDaily := rl.CheckIPIsAllowed(realIP)
	if !allowHour {
		this.Response(http.StatusBadRequest, fmt.Sprintf("同一IP，每小时只能注册 %v 个账户", rl.HourRegNum))
	}
	if !allowDaily {
		this.Response(http.StatusBadRequest, fmt.Sprintf("同一IP，每天只能注册 %v 个账户", rl.DailyRegNum))
	}

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
	if err = rl.Insert(realIP); err != nil {
		beego.Error(err.Error())
	}
	this.login(member)
}

func (this *CommonController) UserFollow() {
	this.getFansOrFollow(false)
}

func (this *CommonController) UserFans() {
	this.getFansOrFollow(true)
}

func (this *CommonController) getFansOrFollow(isGetFans bool) {
	page, _ := this.GetInt("page", 1)
	size, _ := this.GetInt("size", 10)
	if page <= 0 {
		page = 1
	}
	size = utils.RangeNumber(size, 10, maxPageSize)
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
		user.Avatar = this.completeLink(utils.ShowImg(user.Avatar, "avatar"))
		users = append(users, *user)
	}

	data := map[string]interface{}{"total": totalCount}
	if len(users) > 0 {
		data["users"] = users
	}
	this.Response(http.StatusOK, messageSuccess, data)
}

// 查询用户的公开信息
func (this *CommonController) UserInfo() {
	uid, _ := this.GetInt("uid")
	if uid <= 0 {
		uid = this.isLogin()
	}
	if uid <= 0 {
		if this.Token != "" {
			this.Response(http.StatusUnauthorized, messageRequiredLogin)
		}
		this.Response(http.StatusNotFound, messageNotFound)
	}
	member, err := models.NewMember().Find(uid)
	if err != nil && err != orm.ErrNoRows {
		beego.Error(err.Error())
		this.Response(http.StatusInternalServerError, messageInternalServerError)
	}
	if member.MemberId == 0 {
		this.Response(http.StatusNotFound, messageNotFound)
	}
	var user APIUser
	utils.CopyObject(member, &user)

	// 由于是公开信息，不显示用户email
	user.Email = ""
	user.Uid = member.MemberId
	user.Avatar = this.completeLink(utils.ShowImg(user.Avatar, "avatar"))
	this.Response(http.StatusOK, messageSuccess, map[string]interface{}{"user": user})
}

// 如果不传用户id，则表示查询当前登录的用户发布的书籍
func (this *CommonController) UserReleaseBook() {
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
	size, _ := this.GetInt("size", 10)
	size = utils.RangeNumber(size, 10, maxPageSize)

	res, totalCount, err := models.NewBook().FindToPager(page, size, uid, "", 0)
	if err != nil {
		this.Response(http.StatusInternalServerError, messageInternalServerError)
	}

	var books []APIBook
	for _, item := range res {
		book := &APIBook{}
		utils.CopyObject(item, book)
		book.Cover = this.completeLink(utils.ShowImg(book.Cover, "cover"))
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

// [OK]
func (this *CommonController) SearchBook() {
	wd := this.GetString("wd")
	if wd == "" {
		this.Response(http.StatusBadRequest, messageBadRequest)
	}

	if models.NewOption().IsResponseEmptyForAPP(this.Version, wd) {
		this.Response(http.StatusOK, messageSuccess, map[string]interface{}{"total": 0})
	}

	var (
		page, _  = this.GetInt("page", 1)
		size, _  = this.GetInt("size", 10)
		ids      []int
		total    int
		apiBooks []APIBook
		book     APIBook
	)

	size = utils.RangeNumber(size, 10, maxPageSize)

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
			book.Cover = this.completeLink(utils.ShowImg(book.Cover, "cover"))
			apiBooks = append(apiBooks, book)
		}
		data["result"] = apiBooks
	}

	this.Response(http.StatusOK, messageSuccess, data)
}

// [OK]
func (this *CommonController) SearchDoc() {
	wd := this.GetString("wd")
	if wd == "" {
		this.Response(http.StatusBadRequest, messageBadRequest)
	}

	if models.NewOption().IsResponseEmptyForAPP(this.Version, wd) {
		this.Response(http.StatusOK, messageSuccess, map[string]interface{}{"total": 0})
	}

	var (
		page, _ = this.GetInt("page", 1)
		size, _ = this.GetInt("size", 10)
		ids     []int
		total   int
		docs    []APIDoc
		doc     APIDoc
		bookId  = this.getBookIdByIdentify(this.GetString("identify"))
	)

	size = utils.RangeNumber(size, 10, maxPageSize)

	if bookId > 0 {
		page = 1
		size = 2000
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
			ids = append(ids, book.DocumentId)
		}
	}

	data := map[string]interface{}{"total": total}

	if len(ids) > 0 {
		var (
			result    []models.DocResult
			bookIds   []int
			bookIdMap = make(map[int]bool)
			bookInfo  = make(map[int]string) // bookId:bookName
		)
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
			if _, ok := bookIdMap[doc.BookId]; !ok {
				bookIdMap[doc.BookId] = true
				bookIds = append(bookIds, doc.BookId)
			}
			docs = append(docs, doc)
		}

		books, _ := models.NewBook().GetBooksById(bookIds, "book_id", "book_name")
		for _, book := range books {
			bookInfo[book.BookId] = book.BookName
		}
		for idx, doc := range docs {
			if bookName, ok := bookInfo[doc.BookId]; ok {
				doc.BookName = bookName
			} else {
				doc.BookName = "-"
			}
			docs[idx] = doc
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
	m := models.NewOption()
	categories, _ := model.GetCates(pid, 1)
	for idx, category := range categories {
		if m.IsResponseEmptyForAPP(this.Version, category.Title) {
			// 为0，APP端就不会显示该分类
			category.Cnt = 0
		}
		if category.Icon != "" {
			if category.Icon == "" {
				category.Icon = "/static/images/cate.png"
			}
			category.Icon = this.completeLink(category.Icon)
			categories[idx] = category
		}
	}

	this.Response(http.StatusOK, messageSuccess, map[string]interface{}{"categories": categories})
}

// 【OK】
func (this *CommonController) BookInfo() {
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

	apiBook.Cover = this.completeLink(utils.ShowImg(apiBook.Cover, "cover"))
	apiBook.User = models.NewMember().GetNicknameByUid(book.MemberId)
	apiBook.DocReaded = new(models.ReadRecord).BooksProgress(this.isLogin(), apiBook.BookId)[apiBook.BookId] // 这里的map是一定会有值，所以这样取值
	if this.isLogin() > 0 {
		apiBook.IsStar = new(models.Star).DoesStar(this.isLogin(), apiBook.BookId)
	}
	this.Response(http.StatusOK, messageSuccess, map[string]interface{}{"book": apiBook})
}

// 返回用户对该章节是否已读
func (this *CommonController) BookMenu() {
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

	var docs []APIDoc
	uid := this.isLogin()
	readed := make(map[int]bool)
	latestReadId := 0
	if uid > 0 {
		lists, _, _ := new(models.ReadRecord).List(uid, book.BookId)
		if len(lists) > 0 {
			latestReadId = lists[0].DocId
		}
		for _, item := range lists {
			readed[item.DocId] = true
		}
	}

	for _, item := range docsOri {
		var doc APIDoc
		utils.CopyObject(item, &doc)
		if val, ok := readed[doc.DocumentId]; ok {
			doc.Readed = val
		}
		docs = append(docs, doc)
	}
	this.Response(http.StatusOK, messageSuccess, map[string]interface{}{"menu": docs, "latest_read_id": latestReadId})
}

// 【OK】
func (this *CommonController) BookLists() {
	sort := this.GetString("sort", "new") // new、recommend、hot、pin
	page, _ := this.GetInt("page", 1)
	cid, _ := this.GetInt("cid")
	lang := this.GetString("lang")
	size, _ := this.GetInt("size", 10)

	if page <= 0 {
		page = 1
	}

	size = utils.RangeNumber(size, 10, maxPageSize)

	model := models.NewBook()

	fields := []string{"book_id", "book_name", "identify", "order_index", "description", "label", "doc_count",
		"vcnt", "star", "lang", "cover", "score", "cnt_score", "cnt_comment", "modify_time", "create_time", "release_time",
	}

	books, total, _ := model.HomeData(page, size, models.BookOrder(sort), lang, cid, fields...)
	data := map[string]interface{}{"total": total}
	if len(books) > 0 {
		var lists []APIBook
		var list APIBook

		for _, book := range books {
			book.Cover = this.completeLink(utils.ShowImg(book.Cover, "cover"))
			utils.CopyObject(&book, &list)
			lists = append(lists, list)
		}
		data["books"] = lists
	}
	this.Response(http.StatusOK, messageSuccess, data)
}

func (this *CommonController) BookListsByCids() {
	sort := this.GetString("sort", "new") // new、recommend、hot、pin
	page, _ := this.GetInt("page", 1)
	lang := this.GetString("lang")
	size, _ := this.GetInt("size", 10)
	cids := this.GetString("cids")

	var cidArr []int
	slice := strings.Split(cids, ",")
	for _, item := range slice {
		if cid, _ := strconv.Atoi(strings.TrimSpace(item)); cid > 0 {
			cidArr = append(cidArr, cid)
		}
	}

	if len(cidArr) == 0 {
		this.Response(http.StatusBadRequest, messageBadRequest)
	}

	if page <= 0 {
		page = 1
	}

	size = utils.RangeNumber(size, 5, maxPageSize)

	model := models.NewBook()

	fields := []string{"book_id", "book_name", "identify", "order_index", "description", "label", "doc_count",
		"vcnt", "star", "lang", "cover", "score", "cnt_score", "cnt_comment", "modify_time", "create_time", "release_time",
	}
	data := make(map[int]interface{})
	for _, cid := range cidArr {
		books, _, _ := model.HomeData(page, size, models.BookOrder(sort), lang, cid, fields...)
		if len(books) > 0 {
			var lists []APIBook
			var list APIBook
			for _, book := range books {
				book.Cover = this.completeLink(utils.ShowImg(book.Cover, "cover"))
				utils.CopyObject(&book, &list)
				lists = append(lists, list)
			}
			data[cid] = lists
		}
	}

	this.Response(http.StatusOK, messageSuccess, map[string]interface{}{"books": data})
}

func (this *CommonController) Read() {
	identify := this.GetString("identify")
	slice := strings.Split(identify, "/")
	fromAPP, _ := this.GetBool("from-app") // 是否来自app
	enhanceRichtext, _ := this.GetBool("enhance-richtext")
	filterLinks, _ := this.GetBool("links")
	filterImages, _ := this.GetBool("images")
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
	cols := []string{"book_id", "privately_owned", "member_id", "identify"}
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

	//书籍阅读人次+1
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

	var (
		nodes    interface{}
		apiDoc   APIDoc
		apiDocV2 APIDocV2
		isMark   = models.NewBookmark().Exist(this.isLogin(), doc.DocumentId)
		images   []string
		links    []map[string]string
	)
	// 图片链接地址补全
	if doc.Release != "" {
		if enhanceRichtext {
			nodes, images, links = this.handleReleaseV3(doc.Release, bookIdentify)
		} else if fromAPP { // 兼容 app
			nodes = this.handleReleaseV2(doc.Release, bookIdentify)
		} else {
			// 兼容微信小程序
			doc.Release = this.handleReleaseV1(doc.Release, bookIdentify)
		}
	}

	if fromAPP || enhanceRichtext {
		utils.CopyObject(doc, &apiDocV2)
		apiDocV2.Release = nodes
		apiDocV2.Bookmark = isMark
		apiDocV2.Links = []map[string]string{}
		apiDocV2.Images = []string{}

		if filterLinks {
			apiDocV2.Links = links
		}
		if filterImages {
			apiDocV2.Images = images
			apiDocV2.Domain = strings.TrimRight(models.GetAPIStaticDomain(), "/ ")
		}
		this.Response(http.StatusOK, messageSuccess, map[string]interface{}{"article": apiDocV2})
	} else {
		utils.CopyObject(doc, &apiDoc)
		apiDoc.Bookmark = isMark
		this.Response(http.StatusOK, messageSuccess, map[string]interface{}{"article": apiDoc})
	}

}

func (this *CommonController) handleReleaseV1(release string, bookIdentify string) (htmlStr string) {
	query, err := goquery.NewDocumentFromReader(bytes.NewBufferString(release))
	if err != nil {
		beego.Error(err)
	} else {
		// 处理svg
		query = utils.HandleSVG(query, bookIdentify)
		query.Find(".reference-link").Remove()
		query.Find(".header-link").Remove()

		allTags := make(map[string]bool)
		query.Find("body").Find("*").Each(func(i int, selection *goquery.Selection) {
			if len(selection.Nodes) > 0 {
				allTags[strings.ToLower(selection.Nodes[0].Data)] = true
			}
		})

		for tag, _ := range allTags {
			if _, ok := weixinTagsMap.Load(tag); !ok {
				for len(query.Find(tag).Nodes) > 0 {
					query.Find(tag).Each(func(i int, selection *goquery.Selection) {
						if ret, err := selection.Html(); err == nil {
							selection.BeforeHtml(ret)
							selection.Remove()
						}
					})
				}
			}
		}

		weixinTagsMap.Range(func(tag, value interface{}) bool {
			t := tag.(string)
			query.Find(t).AddClass("-" + t).RemoveAttr("id")
			return true
		})

		query.Find("img").Each(func(i int, contentSelection *goquery.Selection) {
			if src, ok := contentSelection.Attr("src"); ok {
				contentSelection.SetAttr("src", this.completeLink(src))
			}
		})

		htmlStr, err = query.Find("body").Html()
		if err != nil {
			beego.Error(err)
		}
	}
	return
}

func (this *CommonController) handleReleaseV2(release, bookIdentify string) interface{} {
	query, err := goquery.NewDocumentFromReader(bytes.NewBufferString(release))
	if err != nil {
		beego.Error(err)
		return release
	}
	// 处理svg
	utils.HandleSVG(query, bookIdentify)
	query.Find(".reference-link").Remove()
	query.Find(".header-link").Remove()
	release, _ = query.Html()

	nodes, err := html2json.NewDefault().Parse(release, models.GetAPIStaticDomain())
	if err != nil {
		beego.Error(err)
		return release
	}
	return nodes
}

func (this *CommonController) handleReleaseV3(release, bookIdentify string) (htmlNodes interface{}, images []string, links []map[string]string) {
	query, err := goquery.NewDocumentFromReader(bytes.NewBufferString(release))
	if err != nil {
		beego.Error(err)
		htmlNodes = release
		return
	}
	// 处理svg
	utils.HandleSVG(query, bookIdentify)
	query.Find(".reference-link").Remove()
	query.Find(".header-link").Remove()
	medias := []string{"audio", "video"}
	for _, tag := range medias {
		query.Find(tag).Each(func(idx int, sel *goquery.Selection) {
			src, ok := sel.Attr("src")
			if ok && !(strings.HasPrefix(src, "https://") || strings.HasPrefix(src, "http://")) {
				if utils.StoreType == utils.StoreOss { // OSS 云存储，则使用OSS签名，否则使用本地存储的链接签名
					if bucket, err := store.ModelStoreOss.GetBucket(); err == nil {
						src = strings.TrimLeft(src, "/")
						src, _ = bucket.SignURL(src, http.MethodGet, utils.MediaDuration)
						if slice := strings.Split(src, "/"); len(slice) > 2 {
							src = strings.Join(slice[3:], "/")
						}
					}
				} else {
					if sign, err := utils.GenerateMediaSign(src, time.Now().UnixNano(), time.Duration(utils.MediaDuration)); err == nil {
						if strings.Contains(src, "?") {
							src = src + "&sign=" + sign
						} else {
							src = src + "?sign=" + sign
						}
					}
				}
			}
			sel.SetAttr("src", src)
		})
	}

	query.Find("img").Each(func(idx int, sel *goquery.Selection) {
		if src, ok := sel.Attr("src"); ok {
			images = append(images, src)
		}
	})

	prefix := "/read/"
	query.Find("a").Each(func(idx int, sel *goquery.Selection) {
		if href, ok := sel.Attr("href"); ok && strings.HasPrefix(href, prefix) {
			if text := strings.TrimSpace(sel.Text()); text != "" {
				href = strings.Split(strings.Split(strings.TrimPrefix(href, prefix), "?")[0], "#")[0]
				links = append(links, map[string]string{"title": text, "href": href})
			}
		}
	})

	release, _ = query.Html()
	nodes, err := html2json.NewDefault().ParseByByteV2([]byte(release), models.GetAPIStaticDomain())
	if err != nil {
		beego.Error(err)
		htmlNodes = release
		return
	}
	htmlNodes = nodes
	return
}

// 【OK】
func (this *CommonController) Banners() {
	t := this.GetString("type", "wechat")
	banners, _ := models.NewBanner().Lists(t)
	bannerSize, _ := strconv.ParseFloat(models.GetOptionValue("MOBILE_BANNER_SIZE", "2.619"), 64)
	if bannerSize <= 0 {
		bannerSize = 2.619
	}

	for idx, banner := range banners {
		banner.Image = this.completeLink(banner.Image)
		banners[idx] = banner
	}

	this.Response(http.StatusOK, messageSuccess, map[string]interface{}{"banners": banners, "size": bannerSize})
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

	this.Response(http.StatusOK, messageSuccess, map[string]interface{}{"files": data})
}

func (this *CommonController) Bookshelf() {
	uid, _ := this.GetInt("uid")
	if uid <= 0 {
		uid = this.isLogin()
	}

	if uid <= 0 {
		if this.Token != "" {
			this.Response(http.StatusUnauthorized, messageRequiredLogin)
		}
		this.Response(http.StatusBadRequest, messageBadRequest)
	}

	cid, _ := this.GetInt("cid")
	withCate, _ := this.GetInt("with-cate", 0)
	size, _ := this.GetInt("size", 10)
	size = utils.RangeNumber(size, 10, maxPageSize)
	page, _ := this.GetInt("page", 1)
	if page <= 0 {
		page = 1
	}
	total, res, err := new(models.Star).List(uid, page, size, cid)
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
		book.Cover = this.completeLink(utils.ShowImg(book.Cover, "cover"))
		books = append(books, *book)
	}

	data := map[string]interface{}{"total": total}

	if len(booksId) > 0 {
		//data["readed"] = new(models.ReadRecord).BooksProgress(uid, booksId...)
		data["books"] = books
	}

	if withCate > 0 {
		data["categories"] = models.NewCategory().CategoryOfUserCollection(uid, true)
	}

	this.Response(http.StatusOK, messageSuccess, data)
}

// 查询书籍评论
func (this *CommonController) GetComments() {
	page, _ := this.GetInt("page", 1)
	if page <= 0 {
		page = 1
	}
	size, _ := this.GetInt("size", 10)
	size = utils.RangeNumber(size, 10, maxPageSize)

	bookId := this.getBookIdByIdentify(this.GetString("identify"))
	if bookId <= 0 {
		this.Response(http.StatusBadRequest, messageBadRequest)
	}

	uid := this.isLogin()
	myScore := 0
	if uid > 0 {
		myScore = new(models.Score).BookScoreByUid(uid, bookId) / 10
	}
	comments, err := new(models.Comments).Comments(page, size, models.CommentOpt{BookId: bookId, Status: []int{1}, WithoutDocComment: true})
	if err != nil {
		beego.Error(err.Error())
	}
	data := map[string]interface{}{
		"my_score": myScore,
		"comments": []string{},
	}

	if len(comments) > 0 {
		for idx, _ := range comments {
			comments[idx].Avatar = this.completeLink(utils.ShowImg(comments[idx].Avatar, "avatar"))
		}
		data["comments"] = comments
	}

	this.Response(http.StatusOK, messageSuccess, data)
}

func (this *CommonController) RelatedBook() {
	bookId := this.getBookIdByIdentify(this.GetString("identify"))
	if bookId <= 0 {
		this.Response(http.StatusBadRequest, messageBadRequest)
	}
	res := models.NewRelateBook().Lists(bookId)
	var books []APIBook
	for _, item := range res {
		book := APIBook{}
		utils.CopyObject(&item, &book)
		book.Cover = this.completeLink(utils.ShowImg(book.Cover, "cover"))
		books = append(books, book)
	}
	data := map[string]interface{}{"books": []string{}}
	if len(books) > 0 {
		data["books"] = books
	}
	this.Response(http.StatusOK, messageSuccess, data)
}

// 查询最近阅读过的书籍，返回最近50本
func (this *CommonController) HistoryReadBook() {
	page, _ := this.GetInt("page", 1)
	size, _ := this.GetInt("size", 10)
	if size <= 0 {
		size = 10
	}
	data := map[string]interface{}{"books": []string{}}
	uid := this.isLogin()
	if uid > 0 {
		books := models.NewReadRecord().HistoryReadBook(uid, page, size)
		for idx, book := range books {
			book.Cover = this.completeLink(book.Cover)
			books[idx] = book
		}
		data["books"] = books
	}
	this.Response(http.StatusOK, messageSuccess, data)
}

func (this *CommonController) LatestVersion() {
	version, _ := strconv.Atoi(models.GetOptionValue("APP_VERSION", "0"))
	page := models.GetOptionValue("APP_PAGE", "")
	this.Response(http.StatusOK, messageSuccess, map[string]interface{}{"version": version, "url": page})
}

func (this *CommonController) Rank() {
	limit, _ := this.GetInt("limit", 50)
	if limit > 200 {
		limit = 200
	}

	data := make(map[string]interface{})

	tab := this.GetString("tab", "all")
	switch tab {
	case "reading":
		rt := models.NewReadingTime()
		data["today"] = rt.Sort(models.PeriodDay, limit, true)
		data["week"] = rt.Sort(models.PeriodWeek, limit, true)
		data["month"] = rt.Sort(models.PeriodMonth, limit, true)
		data["last_week"] = rt.Sort(models.PeriodLastWeek, limit, true)
		data["last_month"] = rt.Sort(models.PeriodLastMoth, limit, true)
		data["all"] = rt.Sort(models.PeriodAll, limit, true)
	case "sign":
		sign := models.NewSign()
		data["continuous_sign"] = sign.Sorted(limit, "total_continuous_sign", true)
		data["total_sign"] = sign.Sorted(limit, "total_sign", true)
		data["this_month_sign"] = sign.SortedByPeriod(limit, models.PeriodMonth, true)
		data["last_month_sign"] = sign.SortedByPeriod(limit, models.PeriodLastMoth, true)
		data["history_continuous_sign"] = sign.Sorted(limit, "history_total_continuous_sign", true)
	case "popular":
		bookCounter := models.NewBookCounter()
		data["today"] = bookCounter.PageViewSort(models.PeriodDay, limit, true)
		data["week"] = bookCounter.PageViewSort(models.PeriodWeek, limit, true)
		data["month"] = bookCounter.PageViewSort(models.PeriodMonth, limit, true)
		data["last_week"] = bookCounter.PageViewSort(models.PeriodLastWeek, limit, true)
		data["last_month"] = bookCounter.PageViewSort(models.PeriodLastMoth, limit, true)
		data["all"] = bookCounter.PageViewSort(models.PeriodAll, limit, true)
	case "star":
		bookCounter := models.NewBookCounter()
		data["today"] = bookCounter.StarSort(models.PeriodDay, limit, true)
		data["week"] = bookCounter.StarSort(models.PeriodWeek, limit, true)
		data["month"] = bookCounter.StarSort(models.PeriodMonth, limit, true)
		data["last_week"] = bookCounter.StarSort(models.PeriodLastWeek, limit, true)
		data["last_month"] = bookCounter.StarSort(models.PeriodLastMoth, limit, true)
		data["all"] = bookCounter.StarSort(models.PeriodAll, limit, true)
	default:
		tab = "all"
		sign := models.NewSign()
		book := models.NewBook()
		data["continuous_sign"] = sign.Sorted(limit, "total_continuous_sign", true)
		data["total_sign"] = sign.Sorted(limit, "total_sign", true)
		data["star_books"] = book.Sorted(limit, "star")
		data["vcnt_books"] = book.Sorted(limit, "vcnt")
		data["comment_books"] = book.Sorted(limit, "cnt_comment")
		// 这里要适配一下APP端，将错就错的写法，转换一下字段
		readers := models.NewReadingTime().Sort(models.PeriodAll, limit, true)
		for idx, reader := range readers {
			reader.TotalReadingTime = reader.SumTime
			readers[idx] = reader
		}
		data["total_reading"] = readers
	}
	this.Response(http.StatusOK, messageSuccess, data)
}
