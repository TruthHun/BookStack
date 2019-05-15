package api

import (
	"fmt"
	"strings"
	"time"

	"github.com/TruthHun/BookStack/models"

	"github.com/astaxie/beego"
)

type BaseController struct {
	beego.Controller
	Token string
}

type APIResponse struct {
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type APIUser struct {
	Uid         int    `json:"uid"`
	Token       string `json:"token"`
	Account     string `json:"username"` //对应 member.account
	Nickname    string `json:"nickname"`
	Email       string `json:"email"`
	Phone       string `json:"phone"`
	Avatar      string `json:"avatar"`
	Description string `json:"intro"`
}

//###################################//

const (
	messageInternalServerError     = "服务内部错误，请联系管理员"
	messageUsernameOrPasswordError = "用户名或密码不正确"
	messageLoginSuccess            = "登录成功"
	messageRequiredLogin           = "您未登录或者您的登录已过期，请重新登录"
	messageLogoutSuccess           = "退出登录成功"
)

//###################################//

func (this *BaseController) Response(httpStatus int, message string, data ...interface{}) {
	this.Ctx.ResponseWriter.Header().Set("Powered By", "BookChat")
	this.Ctx.Output.SetStatus(httpStatus)
	resp := APIResponse{Message: message}
	if len(data) > 0 {
		resp.Data = data[0]
	}

	// support gzip
	if strings.ToLower(this.Ctx.Request.Header.Get("content-encoding")) == "gzip" {
		// TODO
	}
	this.Data["json"] = resp
	this.ServeJSON()
	this.StopRun()
}

// 验证access token
func (this *BaseController) Prepare() {
	this.Token = this.Ctx.Request.Header.Get("Authorization")
	if beego.AppConfig.String("runmode") == "dev" {
		beego.Debug("auth data: ", fmt.Sprintf("%+v", models.NewAuth().AllFromCache()))
		time.Sleep(1 * time.Second)
	}
}

func (this *BaseController) isLogin() {

}

func (this *BaseController) completeImage() {

}
