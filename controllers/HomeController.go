package controllers

import (
	"math"

	"github.com/TruthHun/BookStack/conf"
	"github.com/TruthHun/BookStack/models"
	"github.com/TruthHun/BookStack/utils"
	"github.com/astaxie/beego"
)

type HomeController struct {
	BaseController
}

func (this *HomeController) Index() {
	this.TplName = "home/index.html"
	this.Data["IsHome"] = true
	//如果没有开启匿名访问，则跳转到登录页面
	if !this.EnableAnonymous && this.Member == nil {
		this.Redirect(beego.URLFor("AccountController.Login"), 302)
	}

	pageIndex, _ := this.GetInt("page", 1)
	//每页显示24个，为了兼容Pad、mobile、PC
	pageSize := 24

	member_id := 0

	//首页，无论是否已登录，都只显示公开的文档。用户个人的私有文档，在项目管理里面查看
	//if this.Member != nil {
	//	member_id = this.Member.MemberId
	//}
	books, totalCount, err := models.NewBook().FindForHomeToPager(pageIndex, pageSize, member_id)

	if err != nil {
		beego.Error(err)
		this.Abort("500")
	}
	if totalCount > 0 {
		//html := utils.GetPagerHtml(this.Ctx.Request.RequestURI, pageIndex, pageSize, totalCount)
		html := utils.NewPaginations(conf.RollPage, totalCount, pageSize, pageIndex, "/", "")
		this.Data["PageHtml"] = html
	} else {
		this.Data["PageHtml"] = ""
	}
	this.Data["TotalPages"] = int(math.Ceil(float64(totalCount) / float64(pageSize)))

	this.Data["Lists"] = books

	labels, totalCount, err := models.NewLabel().FindToPager(1, 10)

	if err != nil {
		this.Data["Labels"] = make([]*models.Label, 0)
	} else {
		this.Data["Labels"] = labels
	}

	this.GetSeoByPage("index", map[string]string{
		"title":       this.Sitename,
		"keywords":    "文档托管,在线创作,文档在线管理,在线知识管理,文档托管平台,在线写书,文档在线转换,在线编辑,在线阅读,开发手册,api手册,文档在线学习,技术文档,在线编辑",
		"description": this.Sitename + "专注于文档在线写作、协作、分享、阅读与托管，让每个人更方便地发布、分享和获得知识。",
	})

}
