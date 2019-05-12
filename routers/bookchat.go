package routers

import (
	"github.com/TruthHun/BookStack/controllers"
	"github.com/astaxie/beego"
)

func bookChatRouters() {
	beego.Router("/bookchat/api/v1/login", &controllers.APIController{}, "post:Login")
}
