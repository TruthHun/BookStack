package oauth

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"time"

	"github.com/astaxie/beego"

	"github.com/astaxie/beego/httplib"

	"github.com/xlstudio/wxbizdatacrypt"
)

type SessKey struct {
	OpenId     string `json:"openid"`
	SessionKey string `json:"session_key"`
	UnionId    string `json:"unionid"`
	ErrCode    int    `json:"errcode"`
	ErrMsg     string
}

type WechatUser struct {
	Openid    string `json:"openid"`
	Unionid   string `json:"unionid"`
	AvatarURL string `json:"avatarUrl"`
	City      string `json:"city"`
	Country   string `json:"country"`
	Gender    int    `json:"gender"`
	Language  string `json:"language"`
	NickName  string `json:"nickName"`
	Province  string `json:"province"`
}

func GetWechatSessKey(appId, secret, code string) (sess *SessKey, err error) {
	var resp string
	api := fmt.Sprintf("https://api.weixin.qq.com/sns/jscode2session?appid=%v&secret=%v&js_code=%v&grant_type=authorization_code", appId, secret, code)
	resp, err = httplib.Get(api).SetTimeout(60*time.Second, 60*time.Second).SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true}).String()
	if err != nil {
		beego.Error(err.Error())
		return
	}
	sess = &SessKey{}
	json.Unmarshal([]byte(resp), sess)
	return
}

func DecodeWechatUserInfo(appId, sessKey, encryptedData, iv string) {
	pc := wxbizdatacrypt.WxBizDataCrypt{AppID: appId, SessionKey: sessKey}
	result, err := pc.Decrypt(encryptedData, iv, true) // 第三个参数解释： 需要返回 JSON 数据类型时 使用 true, 需要返回 map 数据类型时 使用 false
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(result)
	}
}
