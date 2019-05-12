package controllers

import (
	"net/http"
	"strings"

	"github.com/TruthHun/BookStack/utils"

	"github.com/TruthHun/BookStack/models"

	"github.com/astaxie/beego"
)

type APIController struct {
	beego.Controller
}

type APIResponse struct {
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type APIUser struct {
	Token       string `json:"token"`
	Account     string `json:"username"` //对应 member.account
	Nickname    string `json:"nickname"`
	Email       string `json:"email"`
	Phone       string `json:"phone"`
	Avatar      string `json:"avatar"`
	Description string `json:"description"`
}

//###################################//

const (
	messageInternalServerError     = "服务内部错误，请联系管理员"
	messageUsernameOrPasswordError = "用户名或密码不正确"
)

//###################################//

func (a *APIController) Response(httpStatus int, message string, data ...interface{}) {
	a.Ctx.ResponseWriter.Header().Set("Powered By", "BookChat")
	a.Ctx.Output.SetStatus(httpStatus)
	resp := APIResponse{Message: message}
	if len(data) > 0 {
		resp.Data = data[0]
	}

	// support gzip
	if strings.ToLower(a.Ctx.Request.Header.Get("content-encoding")) == "gzip" {

	}
	a.StopRun()
}

// 验证access token
func (a *APIController) Prepare() {

}

func (a *APIController) isLogin() {

}

func (a *APIController) completeImage() {

}

func (a *APIController) Login() {
	username := a.GetString("username") //username or email
	password := a.GetString("password")
	member, err := models.NewMember().GetByUsername(username)
	if err != nil {
		beego.Error(err)
		a.Response(http.StatusInternalServerError, messageInternalServerError)
	}
	if err != nil {
		beego.Error(err)
		a.Response(http.StatusInternalServerError, messageInternalServerError)
	}
	if ok, _ := utils.PasswordVerify(member.Password, password); !ok {
		beego.Error(err)
		a.Response(http.StatusBadRequest, messageUsernameOrPasswordError)
	}

	var user APIUser
	utils.CopyObject(member, user)

}

func (a *APIController) Logout() {

}

func (a *APIController) Register() {

}

func (a *APIController) About() {

}

func (a *APIController) UserInfo() {

}

func (a *APIController) UserStar() {

}

func (a *APIController) UserFans() {

}

func (a *APIController) UserFollow() {

}

func (a *APIController) UserReleaseBook() {

}

func (a *APIController) FindPassword() {

}

func (a *APIController) Search() {

}

func (a *APIController) Categories() {

}

func (a *APIController) BookInfo() {

}

func (a *APIController) BookContent() {

}

func (a *APIController) BookMenu() {

}

func (a *APIController) BookLists() {

}

func (a *APIController) ReadProcess() {

}

func (a *APIController) Bookmarks() {

}

func (a *APIController) Banner() {

}
