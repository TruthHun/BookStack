package utils

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/astaxie/beego/httplib"

	"github.com/astaxie/beego"
)

type WechatToken struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int64  `json:"expires_in"`
	ErrCode     int    `json:"errcode"`
	ErrMsg      string `json:"errmsg"`
}

var wechatToken = &WechatToken{}

// 获取access_token
func GetAccessToken() (token *WechatToken) {
	now := time.Now().Unix()
	if now < wechatToken.ExpiresIn-600 { //提前10分钟失效
		return wechatToken
	}
	appId := beego.AppConfig.String("appId")
	appSecret := beego.AppConfig.String("appSecret")
	api := fmt.Sprintf("https://api.weixin.qq.com/cgi-bin/token?grant_type=client_credential&appid=%v&secret=%v", appId, appSecret)
	req := httplib.Get(api).SetTimeout(10*time.Second, 10*time.Second).SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true})
	resp, err := req.String()
	if err != nil {
		beego.Error(err.Error())
	}
	token = &WechatToken{}
	json.Unmarshal([]byte(resp), token)
	token.ExpiresIn = now + token.ExpiresIn
	wechatToken = token
	return
}

// 获取书籍页面小程序码
func GetBookWXACode(accessToken string, bookId int) (tmpFile string, err error) {
	api := fmt.Sprintf("https://api.weixin.qq.com/wxa/getwxacodeunlimit?access_token=" + accessToken)
	data := map[string]interface{}{"page": "pages/intro/intro", "scene": fmt.Sprint(bookId), "width": 280}
	req := httplib.Post(api).SetTimeout(10*time.Second, 10*time.Second).SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true})

	var resp *http.Response
	var b []byte

	b, err = json.Marshal(data)
	if err != nil {
		return
	}

	resp, err = req.Body(b).DoRequest()
	if err != nil {
		return
	}
	defer resp.Body.Close()
	b, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	if contentType := strings.ToLower(resp.Header.Get("content-type")); strings.HasPrefix(contentType, "image/") {
		tmpFile = fmt.Sprintf("book-id-%v.%v", bookId, strings.TrimPrefix(contentType, "image/"))
		err = ioutil.WriteFile(tmpFile, b, os.ModePerm)
	} else {
		err = errors.New(string(b))
	}
	return
}
