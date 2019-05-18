package api

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/TruthHun/BookStack/models"
	"github.com/TruthHun/BookStack/utils"
	"github.com/TruthHun/gotil/cryptil"
	"github.com/TruthHun/gotil/util"
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

	var user APIUser

	utils.CopyObject(&member, &user)

	user.Uid = member.MemberId

	user.Token = cryptil.Md5Crypt(fmt.Sprintf("%v-%v", time.Now().Unix(), util.InterfaceToJson(user)))
	err = models.NewAuth().Insert(user.Token, user.Uid)
	if err != nil {
		beego.Error(err.Error())
		this.Response(http.StatusInternalServerError, messageInternalServerError)
	}
	user.Avatar = utils.JoinURL(models.GetAPIStaticDomain(), user.Avatar)
	this.Response(http.StatusOK, messageLoginSuccess, user)
}

func (this *BaseController) Register() {

}

func (this *BaseController) About() {

}

func (this *BaseController) UserInfo() {

}

func (this *BaseController) UserStar() {

}

func (this *BaseController) UserFans() {

}

func (this *BaseController) UserFollow() {

}

func (this *BaseController) UserReleaseBook() {

}
func (this *CommonController) TODO() {

}

func (this *BaseController) FindPassword() {

}

func (this *BaseController) Search() {

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
			category.Icon = utils.JoinURL(models.GetAPIStaticDomain(), category.Icon)
			categories[idx] = category
		}
	}

	this.Response(http.StatusOK, messageSuccess, categories)
}

func (this *BaseController) BookInfo() {
	var (
		book    *models.Book
		err     error
		apiBook APIBookList
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

	if book.BookId == 0 {
		this.Response(http.StatusNotFound, messageNotFound)
	}

	utils.CopyObject(book, &apiBook)

	apiBook.Cover = utils.JoinURL(models.GetAPIStaticDomain(), apiBook.Cover)
	apiBook.User = models.NewMember().GetNicknameByUid(book.MemberId)

	this.Response(http.StatusOK, messageSuccess, apiBook)
}

func (this *BaseController) BookContent() {

}

func (this *BaseController) BookMenu() {

}

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
		var lists []APIBookList
		var list APIBookList

		for _, book := range books {
			book.Cover = utils.JoinURL(models.GetAPIStaticDomain(), book.Cover)
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

func (this *BaseController) ReadProcess() {

}

func (this *BaseController) Bookmarks() {

}

func (this *BaseController) Banner() {

}
