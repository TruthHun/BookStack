package api

import (
	"fmt"
	"net/http"
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

}

func (this *BaseController) BookContent() {

}

func (this *BaseController) BookMenu() {

}

func (this *BaseController) BookLists() {

}

func (this *BaseController) ReadProcess() {

}

func (this *BaseController) Bookmarks() {

}

func (this *BaseController) Banner() {

}
