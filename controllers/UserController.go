package controllers

import (
	"github.com/TruthHun/BookStack/conf"
	"github.com/TruthHun/BookStack/models"
	"github.com/TruthHun/BookStack/utils"
	"github.com/astaxie/beego"
)

type UserController struct {
	BaseController
}

func (this *UserController) Prepare() {
	this.BaseController.Prepare()
	this.Data["Tab"] = "share"
}

//首页
func (this *UserController) Index() {
	username := this.GetString(":username")
	member, _ := new(models.Member).GetByUsername(username)
	page, _ := this.GetInt("page")
	pageSize := 10
	if page < 1 {
		page = 1
	}

	if member.MemberId == 0 {
		this.Abort("404")
		return
	}
	books, totalCount, _ := models.NewBook().FindToPager(page, pageSize, member.MemberId, 0)
	this.Data["Books"] = books

	if totalCount > 0 {
		html := utils.NewPaginations(conf.RollPage, totalCount, pageSize, page, beego.URLFor("UserController.Index", ":username", member.Account), "")
		this.Data["PageHtml"] = html
	} else {
		this.Data["PageHtml"] = ""
	}
	this.Data["Total"] = totalCount
	this.Data["User"] = member
	this.TplName = "user/index.html"
}

//收藏
func (this *UserController) Collection() {
	username := this.GetString(":username")
	member, _ := new(models.Member).GetByUsername(username)
	page, _ := this.GetInt("page")
	pageSize := 10
	if page < 1 {
		page = 1
	}

	if member.MemberId == 0 {
		this.Abort("404")
		return
	}

	totalCount, books, _ := new(models.Star).List(member.MemberId, page, pageSize)
	this.Data["Books"] = books

	if totalCount > 0 {
		html := utils.NewPaginations(conf.RollPage, int(totalCount), pageSize, page, beego.URLFor("UserController.Collection", ":username", member.Account), "")
		this.Data["PageHtml"] = html
	} else {
		this.Data["PageHtml"] = ""
	}
	this.Data["Total"] = totalCount
	this.Data["User"] = member
	this.Data["Tab"] = "collection"
	this.TplName = "user/collection.html"
}

//粉丝和关注
func (this *UserController) Follow() {

}

//粉丝和关注
func (this *UserController) Fans() {

}
