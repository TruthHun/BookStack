package controllers

import (
	"github.com/TruthHun/BookStack/conf"
	"github.com/TruthHun/BookStack/models"
	"github.com/TruthHun/BookStack/utils"
	"github.com/astaxie/beego"
)

type UserController struct {
	BaseController
	UcenterMember models.Member
}

func (this *UserController) Prepare() {
	this.BaseController.Prepare()
	username := this.GetString(":username")
	this.UcenterMember, _ = new(models.Member).GetByUsername(username)
	if this.UcenterMember.MemberId == 0 {
		this.Abort("404")
		return
	}

	this.Data["IsSelf"] = this.UcenterMember.MemberId == this.Member.MemberId
	this.Data["User"] = this.UcenterMember
	this.Data["Tab"] = "share"
}

//首页
func (this *UserController) Index() {

	page, _ := this.GetInt("page")
	pageSize := 10
	if page < 1 {
		page = 1
	}
	books, totalCount, _ := models.NewBook().FindToPager(page, pageSize, this.UcenterMember.MemberId, 0)
	this.Data["Books"] = books

	if totalCount > 0 {
		html := utils.NewPaginations(conf.RollPage, totalCount, pageSize, page, beego.URLFor("UserController.Index", ":username", this.UcenterMember.Account), "")
		this.Data["PageHtml"] = html
	} else {
		this.Data["PageHtml"] = ""
	}
	this.Data["Total"] = totalCount

	this.TplName = "user/index.html"
}

//收藏
func (this *UserController) Collection() {
	page, _ := this.GetInt("page")
	pageSize := 10
	if page < 1 {
		page = 1
	}

	totalCount, books, _ := new(models.Star).List(this.UcenterMember.MemberId, page, pageSize)
	this.Data["Books"] = books

	if totalCount > 0 {
		html := utils.NewPaginations(conf.RollPage, int(totalCount), pageSize, page, beego.URLFor("UserController.Collection", ":username", this.UcenterMember.Account), "")
		this.Data["PageHtml"] = html
	} else {
		this.Data["PageHtml"] = ""
	}
	this.Data["Total"] = totalCount
	this.Data["Tab"] = "collection"
	this.TplName = "user/collection.html"
}

//关注
func (this *UserController) Follow() {
	page, _ := this.GetInt("page")
	pageSize := 18
	if page < 1 {
		page = 1
	}
	fans, totalCount, _ := new(models.Fans).GetFollowList(this.UcenterMember.MemberId, page, pageSize)
	if totalCount > 0 {
		html := utils.NewPaginations(conf.RollPage, int(totalCount), pageSize, page, beego.URLFor("UserController.Follow", ":username", this.UcenterMember.Account), "")
		this.Data["PageHtml"] = html
	} else {
		this.Data["PageHtml"] = ""
	}
	this.Data["Fans"] = fans
	this.Data["Tab"] = "follow"
	this.TplName = "user/fans.html"
}

//粉丝和关注
func (this *UserController) Fans() {
	page, _ := this.GetInt("page")
	pageSize := 18
	if page < 1 {
		page = 1
	}
	fans, totalCount, _ := new(models.Fans).GetFansList(this.UcenterMember.MemberId, page, pageSize)
	if totalCount > 0 {
		html := utils.NewPaginations(conf.RollPage, int(totalCount), pageSize, page, beego.URLFor("UserController.Fans", ":username", this.UcenterMember.Account), "")
		this.Data["PageHtml"] = html
	} else {
		this.Data["PageHtml"] = ""
	}
	this.Data["Fans"] = fans
	this.Data["Tab"] = "fans"
	this.TplName = "user/fans.html"
}
