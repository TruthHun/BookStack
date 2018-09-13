package oauth

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"strings"

	"fmt"

	"github.com/astaxie/beego"
	"github.com/astaxie/beego/httplib"
)

var debug = beego.AppConfig.String("runmode") == "dev"

//qq accesstoken 数据
type QQAccessToken struct {
	AccessToken string `json:"access_token"`
}

//qq 用户数据
//用户使用qq登录的时候，直接根据qq的id获取数据
type QQUser struct {
	Ret       int    `json:"ret"`
	Msg       string `json:"msg"`
	AvatarURL string `json:"figureurl_qq_2"` //用户头像链接
	Name      string `json:"nickname"`       //昵称
	Gender    string `json:"gender"`         //性别
}

//获取accessToken
func GetQQAccessToken(code string) (token QQAccessToken, err error) {
	var resp string
	Api := beego.AppConfig.String("oauth::qqAccesstoken")
	ClientId := beego.AppConfig.String("oauth::qqClientId")
	ClientSecret := beego.AppConfig.String("oauth::qqClientSecret")
	Callback := beego.AppConfig.String("oauth::qqCallback")
	api := fmt.Sprintf(Api+"?grant_type=%v&code=%v&client_id=%v&redirect_uri=%v&client_secret=%v",
		"authorization_code", code, ClientId, Callback, ClientSecret,
	)
	req := httplib.Get(api)
	if strings.HasPrefix(api, "https") {
		req.SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true})
	}
	//返回来的内容是这种形式的：access_token=E1B30B5CC5C7FED0B23AF715B773FA88&expires_in=7776000&refresh_token=0AD5E0F10D314FCD4E1C6410F76CFEF2，感觉好奇葩，居然不是json
	if resp, err = req.String(); err == nil {
		if debug {
			beego.Debug("获取QQ登录的access_token", resp, api)
		}
		if slice := strings.Split(resp, "&"); len(slice) > 0 {
			for _, item := range slice {
				if strings.HasPrefix(item, "access_token=") {
					token.AccessToken = strings.TrimPrefix(item, "access_token=")
					return
				}
			}
		}
		err = errors.New("获取授权失败，请使用QQ重新获取授权登录")
	}
	return
}

//获取OpenId
func GetQQOpenId(token QQAccessToken) (openid string, err error) {
	var resp string
	Api := beego.AppConfig.String("oauth::qqOpenId") + "?access_token=" + token.AccessToken
	req := httplib.Get(Api)
	if strings.HasPrefix(Api, "https") {
		req.SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true})
	}
	//更奇葩的是这里，我明明是PC的接口，但是返回的确实移动端的数据格式。。
	if resp, err = req.String(); err == nil {
		if debug {
			beego.Debug("获取QQ登录的openid", resp, Api)
		}
		if strings.Contains(resp, "callback(") { //callback( {"client_id":"YOUR_APPID","openid":"YOUR_OPENID"} );
			var data struct {
				Openid string `json:"openid"`
			}
			var js string
			if slice := strings.Split(strings.Split(resp, "}")[0]+"}", "{"); len(slice) == 2 {
				js = "{" + slice[1]
			}
			if err = json.Unmarshal([]byte(js), &data); err == nil {
				openid = data.Openid
				return
			} else {
				beego.Error("解析出错", err.Error())
			}
		} else {
			if slice := strings.Split(resp, "&"); len(slice) > 0 { //client_id=100222222&openid=1704************************878C
				for _, item := range slice {
					if strings.HasPrefix(item, "openid=") {
						openid = strings.TrimPrefix(item, "openid=")
						return
					}
				}
			} else {
				err = errors.New("获取授权失败，请使用QQ重新获取授权登录")
			}
		}
	}
	return
}

//获取用户信息
func GetQQUserInfo(accessToken, openid string) (info QQUser, err error) {
	var resp string
	Api := beego.AppConfig.String("oauth::qqUserInfo") + "?oauth_consumer_key=" + beego.AppConfig.String("oauth::qqClientId") + "&access_token=" + accessToken + "&openid=" + openid
	req := httplib.Get(Api)
	if strings.HasPrefix(Api, "https") {
		req.SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true})
	}
	if resp, err = req.String(); err == nil {
		if debug {
			beego.Debug("获取QQ登录的用户信息", resp, Api)
		}
		err = json.Unmarshal([]byte(resp), &info)
	}
	return
}
