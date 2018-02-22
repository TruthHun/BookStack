package oauth

import (
	"crypto/tls"
	"strings"

	"encoding/json"

	"time"

	"github.com/astaxie/beego"
	"github.com/astaxie/beego/httplib"
)

//gitee accesstoken 数据
type GiteeAccessToken struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
	CreatedAt    int    `json:"created_at"`
}

//gitee 用户数据
//用户使用gitee登录的时候，直接根据gitee的id获取数据
type GiteeUser struct {
	Id        int       `json:"id"`                                  //用户id
	MemberId  int       `json:"member_id"`                           //绑定的用户id
	UpdatedAt time.Time `json:"updated_at"`                          //用户资料更新时间
	AvatarURL string    `json:"avatar_url" orm:"column(avatar_url)"` //用户头像链接
	Email     string    `json:"email" orm:"size(50)"`                //电子邮箱
	Login     string    `json:"login" orm:"size(50)"`                //用户名
	Name      string    `json:"name" orm:"size(50)"`                 //昵称
	HtmlURL   string    `json:"html_url" orm:"column(html_url)"`     //gitee主页
}

//获取accessToken
func GetGiteeAccessToken(code string) (token GiteeAccessToken, err error) {
	var resp string
	Api := beego.AppConfig.String("oauth::giteeAccesstoken")
	ClientId := beego.AppConfig.String("oauth::giteeClientId")
	ClientSecret := beego.AppConfig.String("oauth::giteeClientSecret")
	Callback := beego.AppConfig.String("oauth::giteeCallback")
	req := httplib.Post(Api)
	if strings.HasPrefix(Api, "https") {
		req.SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true})
	}
	req.Param("grant_type", "authorization_code")
	req.Param("code", code)
	req.Param("client_id", ClientId)
	req.Param("redirect_uri", Callback)
	req.Param("client_secret", ClientSecret)
	if resp, err = req.String(); err == nil {
		err = json.Unmarshal([]byte(resp), &token)
	}
	return
}

//获取用户信息
func GetGiteeUserInfo(accessToken string) (info GiteeUser, err error) {
	var resp string
	Api := beego.AppConfig.String("oauth::giteeUserInfo") + "?access_token=" + accessToken
	req := httplib.Get(Api)
	if strings.HasPrefix(Api, "https") {
		req.SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true})
	}
	if resp, err = req.String(); err == nil {
		beego.Debug(resp)
		err = json.Unmarshal([]byte(resp), &info)
	}
	return
}
