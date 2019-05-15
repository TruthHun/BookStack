package routers

import (
	"github.com/TruthHun/BookStack/controllers/api"
	"github.com/astaxie/beego"
)

func bookChatRouters() {
	prefix := "/bookchat"
	beego.Router(prefix+"/api/v1/login", &api.CommonController{}, "post:Login")
	beego.Router(prefix+"/api/v1/logout", &api.LoginedController{}, "get:Logout")
}
