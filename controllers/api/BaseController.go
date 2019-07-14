package api

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/astaxie/beego/orm"

	"github.com/TruthHun/BookStack/utils"

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
	Token       string `json:"token,omitempty"`
	Account     string `json:"username,omitempty"` //对应 member.account
	Nickname    string `json:"nickname"`
	Email       string `json:"email,omitempty"`
	Phone       string `json:"phone,omitempty"`
	Avatar      string `json:"avatar"`
	Description string `json:"intro,omitempty"`
}

type APIBook struct {
	BookId      int       `json:"book_id"`
	BookName    string    `json:"book_name"`
	Identify    string    `json:"identify"`
	OrderIndex  int       `json:"sort"`
	Description string    `json:"description"`
	Label       string    `json:"tags"`
	Vcnt        int       `json:"view"` // 阅读
	Star        int       `json:"star"` // 收藏
	Lang        string    `json:"lang"`
	Cover       string    `json:"cover"`
	Score       int       `json:"score"`       // 文档项目评分，默认40，即4.0星
	CntScore    int       `json:"cnt_score"`   // 评分个数
	CntComment  int       `json:"cnt_comment"` // 评论人数
	DocCount    int       `json:"cnt_doc"`     // 章节数量
	ReleaseTime time.Time `json:"updated_at"`  // 更新时间。这里用书籍的release_time 作为最后的更新时间。因为现有的更新时间不准
	CreateTime  time.Time `json:"created_at"`  // 新建时间
	MemberId    int       `json:"uid,omitempty"`
	User        string    `json:"user,omitempty"`       // 分享人
	Author      string    `json:"author,omitempty"`     // 原作者
	AuthorURL   string    `json:"author_url,omitempty"` // 原作者连接地址
	DocReaded   int       `json:"cnt_readed"`           //已读章节
	IsStar      bool      `json:"is_star"`              //是否已收藏到书架
	//PrivatelyOwned int       `json:"private"`
}

type APIRegister struct {
	Nickname   string `form:"nickname"`
	Account    string `form:"username"`
	Password   string `form:"password"`
	RePassword string `form:"re_password"`
	Email      string `form:"email"`
}

type APIDoc struct {
	DocumentId   int       `json:"id"`
	ParentId     int       `json:"pid"`
	DocumentName string    `json:"title"`
	Identify     string    `json:"identify"`
	BookId       int       `json:"book_id"`
	BookName     string    `json:"book_name"`
	OrderSort    int       `json:"sort"`
	Release      string    `json:"content,omitempty"`
	CreateTime   time.Time `json:"created_at,omitempty"`
	MemberId     int       `json:"uid"`
	ModifyTime   time.Time `json:"updated_at,omitempty"`
	Vcnt         int       `json:"vcnt"`
	Readed       bool      `json:"readed"`
	Bookmark     bool      `json:"bookmark"`
}

type WechatForm struct {
	IV            string `form:"iv"`
	EncryptedData string `form:"encryptedData"`
	Code          string `form:"code"`
}

type WechatBindForm struct {
	Username   string `form:"username"`
	Password   string `form:"password"`
	RePassword string `form:"re_password"`
	Nickname   string `form:"nickname"`
	Email      string `form:"email"`
	Sess       string `form:"sess"`
}

//###################################//

const (
	messageInternalServerError     = "服务内部错误，请联系管理员"
	messageUsernameOrPasswordError = "用户名或密码不正确"
	messageLoginSuccess            = "登录成功"
	messageRequiredLogin           = "您未登录或者您的登录已过期，请重新登录"
	messageLogoutSuccess           = "退出登录成功"
	messageSuccess                 = "操作成功"
	messageBadRequest              = "请求参数不正确"
	messageNotFound                = "资源不存在"
	messageEmailError              = "邮箱格式不正确"
	messageRequiredInput           = "请输入必填项"
	messageNotEqualTwicePassword   = "两次输入密码不一致"
	maxPageSize                    = 30
)

// 微信小程序支持的 HTML 标签：https://developers.weixin.qq.com/miniprogram/dev/component/rich-text.html
var richTextTags = map[string]bool{"a": true, "abbr": true, "address": true, "article": true, "aside": true, "b": true,
	"bdi": true, "bdo": true, "big": true, "blockquote": true, "br": true, "caption": true, "center": true,
	"cite": true, "code": true, "col": true, "colgroup": true, "dd": true, "del": true, "div": true, "dl": true, "dt": true, "em": true,
	"fieldset": true, "font": true, "footer": true, "h1": true, "h2": true, "h3": true, "h4": true, "h5": true, "h6": true,
	"header": true, "hr": true, "i": true, "img": true, "ins": true, "label": true, "legend": true, "li": true, "mark": true,
	"nav": true, "ol": true, "p": true, "pre": true, "q": true, "rt": true, "ruby": true,
	"s": true, "section": true, "small": true, "span": true, "strong": true, "sub": true, "sup": true,
	"table": true, "tbody": true, "td": true, "tfoot": true, "th": true, " thead": true, "tr": true, "tt": true, "u": true, "ul": true}

//###################################//

func (this *BaseController) Response(httpStatus int, message string, data ...interface{}) {
	resp := APIResponse{Message: message}
	if len(data) > 0 {
		resp.Data = data[0]
	}
	returnJSON, err := json.Marshal(resp)
	if err != nil {
		beego.Error(err)
	}
	this.Ctx.ResponseWriter.Header().Set("Content-Type", "application/json; charset=utf-8")
	if strings.Contains(strings.ToLower(this.Ctx.Request.Header.Get("Accept-Encoding")), "gzip") { //gzip压缩
		this.Ctx.ResponseWriter.Header().Set("Content-Encoding", "gzip")
		this.Ctx.ResponseWriter.WriteHeader(httpStatus)
		w := gzip.NewWriter(this.Ctx.ResponseWriter)
		defer w.Close()
		w.Write(returnJSON)
		w.Flush()
	} else {
		io.WriteString(this.Ctx.ResponseWriter, string(returnJSON))
	}
	this.StopRun()
}

// 验证access token
func (this *BaseController) Prepare() {
	//在微信小程序中：网络请求的 referer 是不可以设置的，格式固定为 https://servicewechat.com/{appid}/{version}/page-frame.html，其中 {appid} 为小程序的 appid，{version} 为小程序的版本号，版本号为 0 表示为开发版。
	appId := strings.ToLower(beego.AppConfig.DefaultString("appid", ""))
	limitReferer := beego.AppConfig.DefaultBool("limitReferer", false)
	if appId != "" && limitReferer { // 限定请求的微信小程序的appid
		prefix := fmt.Sprintf("https://servicewechat.com/%v/", appId)
		if !strings.HasPrefix(strings.ToLower(this.Ctx.Request.Referer()), prefix) {
			this.Response(http.StatusNotFound, "not found")
		}
	}
	this.Token = this.Ctx.Request.Header.Get("Authorization")
}

func (this *BaseController) isLogin() (uid int) {
	return models.NewAuth().GetByToken(this.Token).Uid
}

func (this *BaseController) completeLink(path string) string {
	if path == "" {
		return ""
	}
	return utils.JoinURL(models.GetAPIStaticDomain(), path)
}

// 根据标识查询书籍id，标识可以是数字也可以是字符串
func (this *BaseController) getBookIdByIdentify(identify string) (bookId int) {
	bookId, _ = strconv.Atoi(identify)
	if bookId > 0 {
		return
	}
	book := models.NewBook()
	orm.NewOrm().QueryTable(book).Filter("identify", identify).One(book, "book_id")
	return book.BookId
}
