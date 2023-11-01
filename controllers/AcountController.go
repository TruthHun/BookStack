package controllers

import (
	_ "embed"
	"fmt"
	"github.com/TruthHun/BookStack/conf"
	"github.com/TruthHun/BookStack/models"
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/cache"
	"github.com/astaxie/beego/utils/captcha"
	"github.com/casdoor/casdoor-go-sdk/auth"
	"strings"
	//"github.com/lifei6671/gocaptcha"
)

//go:embed token_jwt_key.pem
var JwtPublicKey string

// AccountController 用户登录与注册.
type AccountController struct {
	BaseController
}

var cpt *captcha.Captcha

func init() {
	InitAuthConfig()
	// use beego cache system store the captcha data
	fc := &cache.FileCache{CachePath: "./cache/captcha"}
	cpt = captcha.NewWithFilter("/captcha/", fc)
}

func InitAuthConfig() {
	casdoorEndpoint := beego.AppConfig.String("oauth::casdoorEndpoint")
	clientId := beego.AppConfig.String("oauth::clientId")
	clientSecret := beego.AppConfig.String("oauth::clientSecret")
	casdoorOrganization := beego.AppConfig.String("oauth::casdoorOrganization")
	casdoorApplication := beego.AppConfig.String("oauth::casdoorApplication")

	auth.InitConfig(casdoorEndpoint, clientId, clientSecret, JwtPublicKey, casdoorOrganization, casdoorApplication)
}

// @Title Login
func (c *BaseController) Login() {
	redirectUrl := auth.GetSigninUrl(beego.AppConfig.String("oauth::redirectUrl"))
	c.Redirect(redirectUrl, 302)
}

// @Title Signup
func (c *BaseController) Signup() {
	redirectUrl := auth.GetSignupUrl(true, beego.AppConfig.String("oauth::redirectUrl"))
	c.Redirect(redirectUrl, 302)
}

// @Title Callback
// @Description sign in as a member
func (c *BaseController) Callback() {

	var (
		nickname string //昵称
		avatar   string //头像的http链接地址
		email    string //邮箱地址
		username string //用户名
	)

	code := c.Input().Get("code")
	state := c.Input().Get("state")

	token, err := auth.GetOAuthToken(code, state)
	if err != nil {
		beego.Error(err)
		c.Abort("404")
	}

	claims, err := auth.ParseJwtToken(token.AccessToken)
	if err != nil {
		panic(err)
	}

	claims.AccessToken = token.AccessToken
	c.SetSessionClaims(claims)

	info := &claims.User
	nickname = info.DisplayName
	username = info.Name
	avatar = info.Avatar
	email = info.Email

	member, err := models.NewMember().FindByAccount(username)
	if member.MemberId == 0 {
		//用户不存在，则重新注册用户
		member.Account = username
		member.Nickname = nickname
		member.Role = conf.MemberGeneralRole
		member.Avatar = avatar
		member.CreateAt = 0
		member.Email = email
		member.Status = 0
		if err := member.Add(); err != nil {
			beego.Error(err)
		}
	}

	if err = c.loginByMemberId(member.MemberId); err != nil {
		beego.Error(err.Error())
	}
	c.Redirect(beego.URLFor("HomeController.Index"), 302)
}

// @Title Signout
// @Description sign out the current member
// @Success 200 {object} controllers.api_controller.Response The Response object
// @router /signout [post]
// @Tag Account API
func (c *BaseController) Logout() {

	c.SetSessionClaims(nil)
	c.SetMember(models.Member{})

	c.SetSecureCookie(conf.GetAppKey(), "login", "", -3600)
	c.Redirect(beego.URLFor("HomeController.Index"), 302)
}

//记录笔记
func (c *AccountController) Note() {
	docid, _ := c.GetInt("doc_id")
	fmt.Println(docid)
	if strings.ToLower(c.Ctx.Request.Method) == "post" {

	} else {
		c.Data["SeoTitle"] = "笔记"
		c.TplName = "account/note.html"
	}
}
