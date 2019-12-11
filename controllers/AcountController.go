package controllers

import (
	"regexp"
	"strings"
	"time"

	"errors"

	"fmt"

	"github.com/TruthHun/BookStack/conf"
	"github.com/TruthHun/BookStack/models"
	"github.com/TruthHun/BookStack/oauth"
	"github.com/TruthHun/BookStack/utils"
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/cache"
	"github.com/astaxie/beego/orm"
	"github.com/astaxie/beego/utils/captcha"
	//"github.com/lifei6671/gocaptcha"
)

// AccountController 用户登录与注册.
type AccountController struct {
	BaseController
}

var cpt *captcha.Captcha

func init() {
	// use beego cache system store the captcha data
	fc := &cache.FileCache{CachePath: "./cache/captcha"}
	cpt = captcha.NewWithFilter("/captcha/", fc)
}

//第三方登录回调
//封装一个内部调用的函数，loginByMemberId
func (this *AccountController) Oauth() {

	var (
		nickname  string //昵称
		avatar    string //头像的http链接地址
		email     string //邮箱地址
		username  string //用户名
		tips      string
		id        interface{} //第三方的用户id，唯一识别码
		IsEmail   bool        //是否是使用邮箱注册
		captchaOn bool        //是否开启了验证码
	)

	//如果开启了验证码
	if v, ok := this.Option["ENABLED_CAPTCHA"]; ok && strings.EqualFold(v, "true") {
		captchaOn = true
		this.Data["CaptchaOn"] = captchaOn
	}

	oauthLogin := false
	if v, ok := this.Option["LOGIN_QQ"]; ok && strings.EqualFold(v, "true") {
		this.Data["LoginQQ"] = true
		oauthLogin = true
	}
	if v, ok := this.Option["LOGIN_GITHUB"]; ok && strings.EqualFold(v, "true") {
		this.Data["LoginGitHub"] = true
		oauthLogin = true
	}
	if v, ok := this.Option["LOGIN_GITEE"]; ok && strings.EqualFold(v, "true") {
		this.Data["LoginGitee"] = true
		oauthLogin = true
	}
	this.Data["OauthLogin"] = oauthLogin

	oa := this.GetString(":oauth")
	code := this.GetString("code")
	switch oa {
	case "gitee":
		tips = `您正在使用【码云】登录`
		token, err := oauth.GetGiteeAccessToken(code)
		if err != nil {
			beego.Error(err)
			this.Abort("404")
		}

		info, err := oauth.GetGiteeUserInfo(token.AccessToken)
		if err != nil {
			beego.Error(err)
			this.Abort("404")
		}

		if info.Id > 0 {
			existInfo, _ := models.ModelGitee.GetUserByGiteeId(info.Id, "id", "member_id")
			if existInfo.MemberId > 0 { //直接登录
				err = this.loginByMemberId(existInfo.MemberId)
				if err != nil {
					beego.Error(err)
					this.Abort("404")
				}
				this.Redirect(beego.URLFor("HomeController.Index"), 302)
				return
			}
			if existInfo.Id == 0 { //原本不存在于数据库中的数据需要入库
				orm.NewOrm().Insert(&models.Gitee{GiteeUser: info})
			}
			nickname = info.Name
			username = info.Login
			avatar = info.AvatarURL
			email = info.Email
			id = info.Id
		} else {
			err = errors.New("获取gitee用户数据失败")
			beego.Error(err)
			this.Abort("404")
		}
	case "github":
		tips = `您正在使用【GitHub】登录`
		token, err := oauth.GetGithubAccessToken(code)
		if err != nil {
			beego.Error(err.Error())
			this.Abort("404")
		}

		info, err := oauth.GetGithubUserInfo(token.AccessToken)
		if err != nil {
			beego.Error(err.Error())
			this.Abort("404")
		}

		if info.Id > 0 {
			existInfo, _ := models.ModelGithub.GetUserByGithubId(info.Id, "id", "member_id")
			if existInfo.MemberId > 0 { //直接登录
				err = this.loginByMemberId(existInfo.MemberId)
				if err != nil {
					beego.Error(err.Error())
					this.Abort("404")
				}
				this.Redirect(beego.URLFor("HomeController.Index"), 302)
				return
			}
			if existInfo.Id == 0 { //原本不存在于数据库中的数据需要入库
				orm.NewOrm().Insert(&models.Github{GithubUser: info})
			}
			nickname = info.Name
			username = info.Login
			avatar = info.AvatarURL
			email = info.Email
			id = info.Id
		} else {
			err = errors.New("获取github用户数据失败")
			beego.Error(err.Error())
			this.Abort("404")
		}

	case "qq":
		tips = `您正在使用【QQ】登录`
		token, err := oauth.GetQQAccessToken(code)
		if err != nil {
			beego.Error(err)
			this.Abort("404")
		}

		openid, err := oauth.GetQQOpenId(token)
		if err != nil {
			beego.Error(err.Error())
			this.Abort("404")
		}

		info, err := oauth.GetQQUserInfo(token.AccessToken, openid)
		if err != nil {
			beego.Error(err.Error())
			this.Abort("404")
		}

		if info.Ret == 0 {
			existInfo, _ := models.ModelQQ.GetUserByOpenid(openid, "id", "member_id")
			if existInfo.MemberId > 0 { //直接登录
				err = this.loginByMemberId(existInfo.MemberId)
				if err != nil {
					beego.Error(err.Error())
					this.Abort("404")
				}
				this.Redirect(beego.URLFor("HomeController.Index"), 302)
				return
			}

			if existInfo.Id == 0 { //原本不存在于数据库中的数据需要入库
				orm.NewOrm().Insert(&models.QQ{
					OpenId:    openid,
					Name:      info.Name,
					Gender:    info.Gender,
					AvatarURL: info.AvatarURL,
				})
			}
			nickname = info.Name
			username = ""
			avatar = info.AvatarURL
			email = ""
			id = openid
		} else {
			err = errors.New(info.Msg)
			beego.Error(err)
			this.Abort("404")
		}
	default: //email
		IsEmail = true
	}

	this.Data["IsEmail"] = IsEmail
	this.Data["Nickname"] = nickname
	this.Data["Avatar"] = avatar
	this.Data["Email"] = email
	this.Data["Username"] = username
	this.Data["AuthType"] = oa
	this.Data["SeoTitle"] = "完善信息"
	this.Data["Tips"] = tips
	this.Data["Id"] = id
	this.Data["GiteeClientId"] = beego.AppConfig.String("oauth::giteeClientId")
	this.Data["GiteeCallback"] = beego.AppConfig.String("oauth::giteeCallback")
	this.Data["GithubClientId"] = beego.AppConfig.String("oauth::githubClientId")
	this.Data["GithubCallback"] = beego.AppConfig.String("oauth::githubCallback")
	this.Data["QQClientId"] = beego.AppConfig.String("oauth::qqClientId")
	this.Data["QQCallback"] = beego.AppConfig.String("oauth::qqCallback")
	this.Data["RandomStr"] = time.Now().Unix()
	this.SetSession("auth", fmt.Sprintf("%v-%v", oa, id)) //存储标识，以标记是哪个用户，在完善用户信息的时候跟传递过来的auth和id进行校验
	this.TplName = "account/bind.html"

}

// Login 用户登录.
func (this *AccountController) Login() {
	var (
		remember  CookieRemember
		captchaOn bool //是否开启了验证码
	)

	this.TplName = "account/login.html"

	//如果开启了验证码
	if v, ok := this.Option["ENABLED_CAPTCHA"]; ok && strings.EqualFold(v, "true") {
		captchaOn = true
		this.Data["CaptchaOn"] = captchaOn
	}

	oauthLogin := false
	if v, ok := this.Option["LOGIN_QQ"]; ok && strings.EqualFold(v, "true") {
		this.Data["LoginQQ"] = true
		oauthLogin = true
	}
	if v, ok := this.Option["LOGIN_GITHUB"]; ok && strings.EqualFold(v, "true") {
		this.Data["LoginGitHub"] = true
		oauthLogin = true
	}
	if v, ok := this.Option["LOGIN_GITEE"]; ok && strings.EqualFold(v, "true") {
		this.Data["LoginGitee"] = true
		oauthLogin = true
	}
	this.Data["OauthLogin"] = oauthLogin

	//如果Cookie中存在登录信息
	if cookie, ok := this.GetSecureCookie(conf.GetAppKey(), "login"); ok {
		if err := utils.Decode(cookie, &remember); err == nil {
			if err = this.loginByMemberId(remember.MemberId); err == nil {
				this.Redirect(beego.URLFor("HomeController.Index"), 302)
				return
			}
		}
	}

	if this.Ctx.Input.IsPost() {
		account := this.GetString("account")
		password := this.GetString("password")

		if captchaOn && !cpt.VerifyReq(this.Ctx.Request) {
			this.JsonResult(1, "验证码不正确")
		}

		member, err := models.NewMember().Login(account, password)

		//如果没有数据
		if err != nil {
			beego.Error("用户登录 =>", err)
			this.JsonResult(500, "账号或密码错误", nil)
		}
		member.LastLoginTime = time.Now()
		member.Update()
		this.SetMember(*member)
		remember.MemberId = member.MemberId
		remember.Account = member.Account
		remember.Time = time.Now()
		v, err := utils.Encode(remember)
		if err == nil {
			this.SetSecureCookie(conf.GetAppKey(), "login", v, 24*3600*365)
		}
		this.JsonResult(0, "ok")
	}

	this.Data["GiteeClientId"] = beego.AppConfig.String("oauth::giteeClientId")
	this.Data["GiteeCallback"] = beego.AppConfig.String("oauth::giteeCallback")
	this.Data["GithubClientId"] = beego.AppConfig.String("oauth::githubClientId")
	this.Data["GithubCallback"] = beego.AppConfig.String("oauth::githubCallback")
	this.Data["QQClientId"] = beego.AppConfig.String("oauth::qqClientId")
	this.Data["QQCallback"] = beego.AppConfig.String("oauth::qqCallback")
	this.Data["RandomStr"] = time.Now().Unix()
	this.GetSeoByPage("login", map[string]string{
		"title":       "登录 - " + this.Sitename,
		"keywords":    "登录," + this.Sitename,
		"description": this.Sitename + "专注于文档在线写作、协作、分享、阅读与托管，让每个人更方便地发布、分享和获得知识。",
	})
}

//用户注册.[移除用户注册，直接叫用户绑定]
//注意：如果用户输入的账号密码跟现有的账号密码相一致，则表示绑定账号，否则表示注册新账号。
func (this *AccountController) Bind() {
	var err error
	account := this.GetString("account")
	nickname := strings.TrimSpace(this.GetString("nickname"))
	password1 := this.GetString("password1")
	password2 := this.GetString("password2")
	email := this.GetString("email")
	oauthType := this.GetString("oauth")
	oauthId := this.GetString("id")
	avatar := this.GetString("avatar") //用户头像
	isbind, _ := this.GetInt("isbind", 0)

	ibind := func(oauthType string, oauthId, memberId interface{}) (err error) {
		//注册成功，绑定用户
		switch oauthType {
		case "gitee":
			err = models.ModelGitee.Bind(oauthId, memberId)
		case "github":
			err = models.ModelGithub.Bind(oauthId, memberId)
		case "qq":
			err = models.ModelQQ.Bind(oauthId, memberId)
		}
		return
	}

	if oauthType != "email" {
		if auth, ok := this.GetSession("auth").(string); !ok || fmt.Sprintf("%v-%v", oauthType, oauthId) != auth {
			this.JsonResult(6005, "绑定信息有误，授权类型不符")
		}
	} else { //邮箱登录，如果开启了验证码，则对验证码进行校验
		if v, ok := this.Option["ENABLED_CAPTCHA"]; ok && strings.EqualFold(v, "true") {
			if !cpt.VerifyReq(this.Ctx.Request) {
				this.JsonResult(1, "验证码不正确")
			}
		}
	}

	member := models.NewMember()

	if isbind == 1 {
		if member, err = models.NewMember().Login(account, password1); err != nil || member.MemberId == 0 {
			beego.Error("绑定用户失败", err, member)
			this.JsonResult(1, "绑定用户失败，用户名或密码不正确")
		}
	} else {
		if password1 != password2 {
			this.JsonResult(6003, "登录密码与确认密码不一致")
		}

		if ok, err := regexp.MatchString(conf.RegexpAccount, account); account == "" || !ok || err != nil {
			this.JsonResult(6001, "用户名只能由英文字母数字组成，且在3-50个字符")
		}
		if l := strings.Count(password1, ""); password1 == "" || l > 50 || l < 6 {
			this.JsonResult(6002, "密码必须在6-50个字符之间")
		}

		if ok, err := regexp.MatchString(conf.RegexpEmail, email); !ok || err != nil || email == "" {
			this.JsonResult(6004, "邮箱格式不正确")
		}
		if l := strings.Count(nickname, "") - 1; l < 2 || l > 20 {
			this.JsonResult(6004, "用户昵称限制在2-20个字符")
		}

		//出错或者用户不存在，则重新注册用户，否则直接登录
		member.Account = account
		member.Nickname = nickname
		member.Password = password1
		member.Role = conf.MemberGeneralRole
		member.Avatar = conf.GetDefaultAvatar()
		member.CreateAt = 0
		member.Email = email
		member.Status = 0
		if len(avatar) > 0 {
			member.Avatar = avatar
		}
		if err := member.Add(); err != nil {
			beego.Error(err)
			this.JsonResult(6006, err.Error())
		}
	}
	if err = this.loginByMemberId(member.MemberId); err != nil {
		beego.Error(err.Error())
		this.JsonResult(1, err.Error())
	}

	if err = ibind(oauthType, oauthId, member.MemberId); err != nil {
		beego.Error(err)
		this.JsonResult(0, "登录失败")
	}

	if oauthType == "email" {
		this.JsonResult(0, "注册成功")
	}
	this.JsonResult(0, "登录成功")
}

//找回密码.
func (this *AccountController) FindPassword() {

	this.TplName = "account/find_password_setp1.html"
	mailConf := conf.GetMailConfig()

	if this.Ctx.Input.IsPost() {

		email := this.GetString("email")

		if email == "" {
			this.JsonResult(6005, "邮箱地址不能为空")
		}
		if !mailConf.EnableMail {
			this.JsonResult(6004, "未启用邮件服务")
		}

		//captcha := this.GetString("code")
		//如果开启了验证码
		//if v, ok := this.Option["ENABLED_CAPTCHA"]; ok && strings.EqualFold(v, "true") {
		//	v, ok := this.GetSession(conf.CaptchaSessionName).(string)
		//	if !ok || !strings.EqualFold(v, captcha) {
		//		this.JsonResult(6001, "验证码不正确")
		//	}
		//}

		if !cpt.VerifyReq(this.Ctx.Request) {
			this.JsonResult(6001, "验证码不正确")
		}

		member, err := models.NewMember().FindByFieldFirst("email", email)
		if err != nil {
			beego.Error(err)
			this.JsonResult(6006, "邮箱不存在")
		}
		if member.Status != 0 {
			this.JsonResult(6007, "账号已被禁用")
		}
		if member.AuthMethod == conf.AuthMethodLDAP {
			this.JsonResult(6011, "当前用户不支持找回密码")
		}

		count, err := models.NewMemberToken().FindSendCount(email, time.Now().Add(-1*time.Hour), time.Now())

		if err != nil {
			beego.Error(err)
			this.JsonResult(6008, "发送邮件失败")
		}
		if count > mailConf.MailNumber {
			this.JsonResult(6008, "发送次数太多，请稍候再试")
		}

		memberToken := models.NewMemberToken()

		memberToken.Token = string(utils.Krand(32, utils.KC_RAND_KIND_ALL))
		memberToken.Email = email
		memberToken.MemberId = member.MemberId
		memberToken.IsValid = false
		if _, err := memberToken.InsertOrUpdate(); err != nil {
			this.JsonResult(6009, "邮件发送失败")
		}

		data := map[string]interface{}{
			"SITE_NAME": this.Option["SITE_NAME"],
			"url":       this.BaseUrl() + beego.URLFor("AccountController.FindPassword", "token", memberToken.Token, "mail", email),
		}

		body, err := this.ExecuteViewPathTemplate("account/mail_template.html", data)
		if err != nil {
			beego.Error(err)
			this.JsonResult(6003, "邮件发送失败")
		}

		if err = utils.SendMail(mailConf, "找回密码", email, body); err != nil {
			beego.Error(err)
			this.JsonResult(6003, "邮件发送失败")
		}

		this.JsonResult(0, "ok", this.BaseUrl()+beego.URLFor("AccountController.Login"))
	}

	this.GetSeoByPage("findpwd", map[string]string{
		"title":       "找回密码 - " + this.Sitename,
		"keywords":    "找回密码",
		"description": this.Sitename + "专注于文档在线写作、协作、分享、阅读与托管，让每个人更方便地发布、分享和获得知识。",
	})

	token := this.GetString("token")
	mail := this.GetString("mail")

	if token != "" && mail != "" {
		memberToken, err := models.NewMemberToken().FindByFieldFirst("token", token)

		if err != nil {
			beego.Error(err)
			this.Data["ErrorMessage"] = "邮件已失效"
			this.TplName = "errors/error.html"
			return
		}
		subTime := memberToken.SendTime.Sub(time.Now())

		if !strings.EqualFold(memberToken.Email, mail) || subTime.Minutes() > float64(mailConf.MailExpired) || !memberToken.ValidTime.IsZero() {
			this.Data["ErrorMessage"] = "验证码已过期，请重新操作。"
			this.TplName = "errors/error.html"
			return
		}
		this.Data["Email"] = memberToken.Email
		this.Data["Token"] = memberToken.Token
		this.TplName = "account/find_password_setp2.html"

	}

}

//校验邮件并修改密码.
func (this *AccountController) ValidEmail() {
	password1 := this.GetString("password1")
	password2 := this.GetString("password2")
	token := this.GetString("token")
	mail := this.GetString("mail")

	if password1 == "" {
		this.JsonResult(6001, "密码不能为空")
	}
	if l := strings.Count(password1, ""); l < 6 || l > 50 {
		this.JsonResult(6001, "密码不能为空且必须在6-50个字符之间")
	}
	if password2 == "" {
		this.JsonResult(6002, "确认密码不能为空")
	}
	if password1 != password2 {
		this.JsonResult(6003, "确认密码输入不正确")
	}

	if !cpt.VerifyReq(this.Ctx.Request) {
		this.JsonResult(6001, "验证码不正确")
	}

	mailConf := conf.GetMailConfig()
	memberToken, err := models.NewMemberToken().FindByFieldFirst("token", token)

	if err != nil {
		beego.Error(err)
		this.JsonResult(6007, "邮件已失效")
	}
	subTime := memberToken.SendTime.Sub(time.Now())

	if !strings.EqualFold(memberToken.Email, mail) || subTime.Minutes() > float64(mailConf.MailExpired) || !memberToken.ValidTime.IsZero() {

		this.JsonResult(6008, "验证码已过期，请重新操作。")
	}
	member, err := models.NewMember().Find(memberToken.MemberId)
	if err != nil {
		beego.Error(err)
		this.JsonResult(6005, "用户不存在")
	}
	hash, err := utils.PasswordHash(password1)

	if err != nil {
		beego.Error(err)
		this.JsonResult(6006, "保存密码失败")
	}

	member.Password = hash

	err = member.Update("password")
	memberToken.ValidTime = time.Now()
	memberToken.IsValid = true
	memberToken.InsertOrUpdate()

	if err != nil {
		beego.Error(err)
		this.JsonResult(6006, "保存密码失败")
	}
	this.JsonResult(0, "ok", this.BaseUrl()+beego.URLFor("AccountController.Login"))
}

// Logout 退出登录.
func (this *AccountController) Logout() {
	this.SetMember(models.Member{})

	this.SetSecureCookie(conf.GetAppKey(), "login", "", -3600)

	this.Redirect(beego.URLFor("AccountController.Login"), 302)
}

//记录笔记
func (this *AccountController) Note() {
	docid, _ := this.GetInt("doc_id")
	fmt.Println(docid)
	if strings.ToLower(this.Ctx.Request.Method) == "post" {

	} else {
		this.Data["SeoTitle"] = "笔记"
		this.TplName = "account/note.html"
	}
}
