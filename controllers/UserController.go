package controllers

import (
	"time"

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
	rt := models.NewReadingTime()
	this.Data["IsSelf"] = this.UcenterMember.MemberId == this.Member.MemberId
	this.Data["User"] = this.UcenterMember
	this.Data["JoinedDays"] = int(time.Now().Sub(this.UcenterMember.CreateTime).Seconds()/(24*3600)) + 1
	this.Data["TotalReading"] = utils.FormatReadingTime(this.UcenterMember.TotalReadingTime)
	this.Data["MonthReading"] = utils.FormatReadingTime(rt.GetReadingTime(this.UcenterMember.MemberId, models.PeriodMonth))
	this.Data["WeekReading"] = utils.FormatReadingTime(rt.GetReadingTime(this.UcenterMember.MemberId, models.PeriodWeek))
	this.Data["TodayReading"] = utils.FormatReadingTime(rt.GetReadingTime(this.UcenterMember.MemberId, models.PeriodDay))
	this.Data["Tab"] = "share"
	this.Data["IsSign"] = false
	if this.Member != nil && this.Member.MemberId > 0 {
		this.Data["LatestSign"] = models.NewSign().LatestOne(this.Member.MemberId)
	}
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
	this.GetSeoByPage("ucenter-share", map[string]string{
		"title":       "分享 - " + this.UcenterMember.Nickname,
		"keywords":    "用户主页," + this.UcenterMember.Nickname,
		"description": this.Sitename + "专注于文档在线写作、协作、分享、阅读与托管，让每个人更方便地发布、分享和获得知识。",
	})

	this.TplName = "user/index.html"
}

//收藏
func (this *UserController) Collection() {
	page, _ := this.GetInt("page")
	cid, _ := this.GetInt("cid")
	pageSize := 10
	if page < 1 {
		page = 1
	}

	totalCount, books, _ := new(models.Star).List(this.UcenterMember.MemberId, page, pageSize, cid)
	this.Data["Books"] = books

	if totalCount > 0 {
		html := utils.NewPaginations(conf.RollPage, int(totalCount), pageSize, page, beego.URLFor("UserController.Collection", ":username", this.UcenterMember.Account), "")
		this.Data["PageHtml"] = html
	} else {
		this.Data["PageHtml"] = ""
	}
	this.GetSeoByPage("ucenter-collection", map[string]string{
		"title":       "收藏 - " + this.UcenterMember.Nickname,
		"keywords":    "用户收藏," + this.UcenterMember.Nickname,
		"description": this.Sitename + "专注于文档在线写作、协作、分享、阅读与托管，让每个人更方便地发布、分享和获得知识。",
	})
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
	this.GetSeoByPage("ucenter-follow", map[string]string{
		"title":       "关注 - " + this.UcenterMember.Nickname,
		"keywords":    "用户关注," + this.UcenterMember.Nickname,
		"description": this.Sitename + "专注于文档在线写作、协作、分享、阅读与托管，让每个人更方便地发布、分享和获得知识。",
	})
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
	this.GetSeoByPage("ucenter-fans", map[string]string{
		"title":       "粉丝 - " + this.UcenterMember.Nickname,
		"keywords":    "用户粉丝," + this.UcenterMember.Nickname,
		"description": this.Sitename + "专注于文档在线写作、协作、分享、阅读与托管，让每个人更方便地发布、分享和获得知识。",
	})
	this.Data["Fans"] = fans
	this.Data["Tab"] = "fans"
	this.TplName = "user/fans.html"
}
