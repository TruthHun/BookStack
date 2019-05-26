package routers

import (
	"github.com/TruthHun/BookStack/controllers/api"
	"github.com/astaxie/beego"
)

func bookChatRouters() {
	prefix := "/bookchat"

	// finished
	beego.Router(prefix+"/api/v1/register", &api.CommonController{}, "post:Register")
	beego.Router(prefix+"/api/v1/login", &api.CommonController{}, "post:Login")
	beego.Router(prefix+"/api/v1/logout", &api.LoginedController{}, "get:Logout")
	beego.Router(prefix+"/api/v1/banners", &api.CommonController{}, "get:Banners")
	beego.Router(prefix+"/api/v1/book/categories", &api.CommonController{}, "get:Categories")
	beego.Router(prefix+"/api/v1/book/lists", &api.CommonController{}, "get:BookLists")
	beego.Router(prefix+"/api/v1/book/info", &api.CommonController{}, "get:BookInfo")
	beego.Router(prefix+"/api/v1/book/menu", &api.CommonController{}, "get:BookMenu")
	beego.Router(prefix+"/api/v1/search/book", &api.CommonController{}, "get:SearchBook")
	beego.Router(prefix+"/api/v1/search/doc", &api.CommonController{}, "get:SearchDoc")
	beego.Router(prefix+"/api/v1/book/bookmark", &api.LoginedController{}, "get:GetBookmarks")
	beego.Router(prefix+"/api/v1/book/bookmark", &api.LoginedController{}, "Put:SetBookmarks")
	beego.Router(prefix+"/api/v1/book/bookmark", &api.LoginedController{}, "delete:DeleteBookmarks")
	beego.Router(prefix+"/api/v1/book/download", &api.CommonController{}, "get:Download")
	beego.Router(prefix+"/api/v1/book/read", &api.CommonController{}, "get:Read")

	// developing
	beego.Router(prefix+"/api/v1/about-us", &api.CommonController{}, "get:TODO")
	beego.Router(prefix+"/api/v1/user/find-password", &api.CommonController{}, "get:TODO")
	beego.Router(prefix+"/api/v1/user/change-password", &api.CommonController{}, "get:TODO")
	beego.Router(prefix+"/api/v1/user/change-avatar", &api.CommonController{}, "get:TODO")
	beego.Router(prefix+"/api/v1/user/info", &api.CommonController{}, "get:TODO")
	beego.Router(prefix+"/api/v1/user/star", &api.CommonController{}, "get:TODO")
	beego.Router(prefix+"/api/v1/user/release", &api.CommonController{}, "get:TODO")
	beego.Router(prefix+"/api/v1/user/fans", &api.CommonController{}, "get:TODO")
	beego.Router(prefix+"/api/v1/user/follow", &api.CommonController{}, "get:TODO")
	beego.Router(prefix+"/api/v1/book/comment", &api.CommonController{}, "get:TODO")
	beego.Router(prefix+"/api/v1/book/comment", &api.CommonController{}, "post:TODO")
	beego.Router(prefix+"/api/v1/book/process", &api.CommonController{}, "get:TODO")
	beego.Router(prefix+"/api/v1/book/reset-process", &api.CommonController{}, "get:TODO")
}
