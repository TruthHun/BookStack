package api

import (
	"net/http"

	"github.com/TruthHun/BookStack/models"
)

// 登录之后才能调用的接口放这里
type LoginedController struct {
	BaseController
}

func (this *LoginedController) Prepare() {
	this.BaseController.Prepare()
	if models.NewAuth().GetByToken(this.Token).Uid == 0 {
		this.Response(http.StatusUnauthorized, messageRequiredLogin)
	}
}

func (this *LoginedController) UserInfo() {

}

func (this *LoginedController) Logout() {
	models.NewAuth().DeleteByToken(this.Token)
	this.Response(http.StatusOK, messageLogoutSuccess)
}
