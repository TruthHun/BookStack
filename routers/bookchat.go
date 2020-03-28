package routers

import (
	"strings"

	"github.com/astaxie/beego/plugins/cors"

	"github.com/TruthHun/BookStack/controllers/api"
	"github.com/astaxie/beego"
)

func bookChatRouters() {
	prefix := strings.TrimSpace(beego.AppConfig.DefaultString("apiPrefix", "/bookchat"))
	prefix = "/" + strings.Trim(prefix, "./")

	if beego.AppConfig.DefaultBool("allowCors", true) {
		beego.InsertFilter(prefix+"/*", beego.BeforeRouter, cors.Allow(&cors.Options{
			AllowAllOrigins: true,
			AllowMethods:    []string{"PUT", "POST", "OPTIONS", "GET", "DELETE"},
		}))
	}

	// finished
	beego.Router(prefix+"/api/v1/version", &api.CommonController{}, "get:LatestVersion")
	beego.Router(prefix+"/api/v1/register", &api.CommonController{}, "get,post:Register")
	beego.Router(prefix+"/api/v1/login", &api.CommonController{}, "post:Login")
	beego.Router(prefix+"/api/v1/login-by-wechat", &api.CommonController{}, "post:LoginByWechat")
	beego.Router(prefix+"/api/v1/login-bind-wechat", &api.CommonController{}, "post:LoginBindWechat")
	beego.Router(prefix+"/api/v1/logout", &api.LoginedController{}, "get:Logout")
	beego.Router(prefix+"/api/v1/banners", &api.CommonController{}, "get:Banners")
	beego.Router(prefix+"/api/v1/rank", &api.CommonController{}, "get:Rank")
	beego.Router(prefix+"/api/v1/book/categories", &api.CommonController{}, "get:Categories")
	beego.Router(prefix+"/api/v1/book/lists", &api.CommonController{}, "get:BookLists")
	beego.Router(prefix+"/api/v1/book/lists-by-cids", &api.CommonController{}, "get:BookListsByCids")
	beego.Router(prefix+"/api/v1/book/info", &api.CommonController{}, "get:BookInfo")
	beego.Router(prefix+"/api/v1/book/menu", &api.CommonController{}, "get:BookMenu")
	beego.Router(prefix+"/api/v1/search/book", &api.CommonController{}, "get:SearchBook")
	beego.Router(prefix+"/api/v1/search/doc", &api.CommonController{}, "get:SearchDoc")
	beego.Router(prefix+"/api/v1/book/bookmark", &api.LoginedController{}, "get:GetBookmarks")
	beego.Router(prefix+"/api/v1/book/bookmark", &api.LoginedController{}, "put:SetBookmarks")
	beego.Router(prefix+"/api/v1/book/bookmark", &api.LoginedController{}, "delete:DeleteBookmarks")
	beego.Router(prefix+"/api/v1/book/download", &api.CommonController{}, "get:Download")
	beego.Router(prefix+"/api/v1/book/read", &api.CommonController{}, "get:Read")
	beego.Router(prefix+"/api/v1/user/info", &api.CommonController{}, "get:UserInfo")
	beego.Router(prefix+"/api/v1/user/release", &api.CommonController{}, "get:UserReleaseBook")
	beego.Router(prefix+"/api/v1/user/fans", &api.CommonController{}, "get:UserFans")
	beego.Router(prefix+"/api/v1/user/follow", &api.CommonController{}, "get:UserFollow")
	beego.Router(prefix+"/api/v1/user/bookshelf", &api.CommonController{}, "get:Bookshelf")
	beego.Router(prefix+"/api/v1/book/comment", &api.CommonController{}, "get:GetComments")
	beego.Router(prefix+"/api/v1/book/history", &api.CommonController{}, "get:HistoryReadBook")
	beego.Router(prefix+"/api/v1/book/comment", &api.LoginedController{}, "post:PostComment")
	beego.Router(prefix+"/api/v1/book/star", &api.LoginedController{}, "get,put:Star")
	beego.Router(prefix+"/api/v1/book/related", &api.CommonController{}, "get:RelatedBook")
	beego.Router(prefix+"/api/v1/user/change-avatar", &api.LoginedController{}, "post:ChangeAvatar")
	beego.Router(prefix+"/api/v1/user/change-password", &api.LoginedController{}, "post:ChangePassword")
	beego.Router(prefix+"/api/v1/user/sign", &api.LoginedController{}, "post:SignToday")
	beego.Router(prefix+"/api/v1/user/sign", &api.LoginedController{}, "get:SignStatus")
	beego.Router(prefix+"/api/v1/user/more-info", &api.CommonController{}, "get:GetUserMoreInfo")

	// developing
	//beego.Router(prefix+"/api/v1/user/find-password", &api.CommonController{}, "get:TODO")
}
