package oauth

import (
	"crypto/tls"
	"encoding/json"
	"strings"

	"time"

	"github.com/astaxie/beego"
	"github.com/astaxie/beego/httplib"
)

//GitHub accesstoken 数据
type GithubAccessToken struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
}

//GitHub 用户数据
//用户使用GitHub登录的时候，直接根据GitHub的id获取数据
type GithubUser struct {
	Id        int       `json:"id"`                                  //用户id
	MemberId  int       `json:"member_id"`                           //绑定的用户id
	UpdatedAt time.Time `json:"updated_at"`                          //用户资料更新时间
	AvatarURL string    `json:"avatar_url" orm:"column(avatar_url)"` //用户头像链接
	Email     string    `json:"email" orm:"size(50)"`                //电子邮箱
	Login     string    `json:"login" orm:"size(50)"`                //用户名
	Name      string    `json:"name" orm:"size(50)"`                 //昵称
	HtmlURL   string    `json:"html_url" orm:"column(html_url)"`     //github主页
}

//获取accessToken
func GetGithubAccessToken(code string) (token GithubAccessToken, err error) {
	var resp string
	Api := beego.AppConfig.String("oauth::githubAccesstoken")
	ClientId := beego.AppConfig.String("oauth::githubClientId")
	ClientSecret := beego.AppConfig.String("oauth::githubClientSecret")
	Callback := beego.AppConfig.String("oauth::githubCallback")
	req := httplib.Post(Api)
	req.Header("Accept", "application/json")
	if strings.HasPrefix(Api, "https") {
		req.SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true})
	}
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
func GetGithubUserInfo(accessToken string) (info GithubUser, err error) {
	var resp string
	//Api := beego.AppConfig.String("oauth::githubUserInfo") + "?access_token=" + accessToken
	Api := beego.AppConfig.String("oauth::githubUserInfo")
	req := httplib.Get(Api)
	if strings.HasPrefix(Api, "https") {
		req.SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true})
	}
	req.Header("Authorization", "token "+accessToken)
	if resp, err = req.String(); err == nil {
		beego.Debug(resp)
		err = json.Unmarshal([]byte(resp), &info)
	}
	return
}
