package api

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/astaxie/beego/orm"

	"github.com/TruthHun/BookStack/utils"

	"github.com/TruthHun/BookStack/models"

	"github.com/astaxie/beego"
)

type BaseController struct {
	beego.Controller
	Token   string
	Version string
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
	Score       int       `json:"score"`       // 书籍评分，默认40，即4.0星
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

type UserMoreInfo struct {
	MemberId              int `json:"uid"`
	SignedAt              int `json:"signed_at"`               // 签到时间
	CreatedAt             int `json:"created_at"`              // 注册时间
	TotalSign             int `json:"total_sign"`              // 总签到天数
	TotalContinuousSign   int `json:"total_continuous_sign"`   // 总连续签到天数
	HistoryContinuousSign int `json:"history_continuous_sign"` // 历史连续签到天数
	TodayReading          int `json:"today_reading"`           // 今日阅读时长
	MonthReading          int `json:"month_reading"`           // 本月阅读时长
	TotalReading          int `json:"total_reading"`           // 总阅读时长
}

type APIDocV2 struct {
	DocumentId   int         `json:"id"`
	ParentId     int         `json:"pid"`
	DocumentName string      `json:"title"`
	Identify     string      `json:"identify"`
	BookId       int         `json:"book_id"`
	BookName     string      `json:"book_name"`
	OrderSort    int         `json:"sort"`
	Release      interface{} `json:"content,omitempty"`
	CreateTime   time.Time   `json:"created_at,omitempty"`
	MemberId     int         `json:"uid"`
	ModifyTime   time.Time   `json:"updated_at,omitempty"`
	Vcnt         int         `json:"vcnt"`
	Readed       bool        `json:"readed"`
	Bookmark     bool        `json:"bookmark"`
}

type WechatForm struct {
	UserInfo string `form:"userInfo"`
	Code     string `form:"code"`
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
	messageMustLogin               = "内容暂时不允许游客访问，请先登录"
	messageForbidRegister          = "网站已经关闭了注册功能，暂时不允许注册"
	maxPageSize                    = 30
)

// 微信小程序支持的 HTML 标签：https://developers.weixin.qq.com/miniprogram/dev/component/rich-text.html
var weixinTags = []string{"a", "abbr", "address", "article", "aside", "b", "bdi", "bdo", "big", "blockquote", "br", "caption", "center", "cite", "code", "col", "colgroup", "dd", "del", "div", "dl", "dt", "em", "fieldset", "font", "footer", "h1", "h2", "h3", "h4", "h5", "h6", "header", "hr", "i", "img", "ins", "label", "legend", "li", "mark", "nav", "ol", "p", "pre", "q", "rt", "ruby", "s", "section", "small", "span", "strong", "sub", "sup", "table", "tbody", "td", "tfoot", "th", " thead", "tr", "tt", "u", "ul"}
var appTags = []string{"a", "abbr", "b", "blockquote", "br", "code", "col", "colgroup", "dd", "del", "div", "dl", "dt", "em", "fieldset", "h1", "h2", "h3", "h4", "h5", "h6", "header", "hr", "i", "img", "ins", "label", "legend", "li", "ol", "p", "q", "span", "strong", "sub", "sup", "table", "tbody", "td", "tfoot", "th", "thead", "tr", "tt", "ul"}

var weixinTagsMap, appTagsMap sync.Map

func init() {
	for _, tag := range weixinTags {
		weixinTagsMap.Store(tag, true)
	}
	for _, tag := range appTags {
		appTagsMap.Store(tag, true)
	}
}

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
	if appId != "" && limitReferer && beego.AppConfig.String("runmode") != "dev" { // 限定请求的微信小程序的appid
		prefix := fmt.Sprintf("https://servicewechat.com/%v/", appId)
		if !strings.HasPrefix(strings.ToLower(this.Ctx.Request.Referer()), prefix) {
			this.Response(http.StatusNotFound, "not found")
		}
	}
	this.Token = this.Ctx.Request.Header.Get("Authorization")
	this.Version = this.Ctx.Request.Header.Get("x-version")

	if !models.AllowVisitor && this.isLogin() == 0 { // 不允许游客访问，则除了部分涉及登录的API外，一律提示先登录
		allowAPIs := map[string]bool{
			beego.URLFor("CommonController.Login"):         true,
			beego.URLFor("CommonController.LoginByWechat"): true,
			//beego.URLFor("CommonController.LoginBindWechat"): true, // 这个接口属于绑定信息的，属于注册接口。
		}
		if _, ok := allowAPIs[this.Ctx.Request.URL.Path]; !ok {
			this.Response(http.StatusBadRequest, messageMustLogin)
		}
	}

	if !models.AllowRegister { // 如果不允许注册，则不允许用户访问注册相关的接口，其他接口可以访问
		denyAPIs := map[string]bool{
			beego.URLFor("CommonController.LoginBindWechat"): true, // 这个接口属于绑定信息的，属于注册接口。
			beego.URLFor("CommonController.Register"):        true, // 这个接口属于绑定信息的，属于注册接口。
		}
		if _, ok := denyAPIs[this.Ctx.Request.URL.Path]; ok {
			this.Response(http.StatusBadRequest, messageForbidRegister)
		}
	}

}

func (this *BaseController) isLogin() (uid int) {
	return models.NewAuth().GetByToken(this.Token).Uid
}

func (this *BaseController) completeLink(path string) string {
	if path == "" {
		return ""
	}
	return utils.JoinURL(models.GetAPIStaticDomain(), strings.ReplaceAll(path, "\\", "/"))
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
