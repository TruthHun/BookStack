package controllers

import (
	"math"

	"strings"

	"strconv"

	"github.com/TruthHun/BookStack/conf"
	"github.com/TruthHun/BookStack/models"
	"github.com/TruthHun/BookStack/utils"
	"github.com/astaxie/beego"
)

type HomeController struct {
	BaseController
}

func (this *HomeController) Index() {
	//tab
	var (
		tab       string
		cid       int //分类，如果只是一级分类，则忽略，二级分类，则根据二级分类查找内容
		urlPrefix = beego.URLFor("HomeController.Index")
		cate      models.Category
		lang      = this.GetString("lang")
		tabName   = map[string]string{"recommend": "网站推荐", "latest": "最新发布", "popular": "热门书籍"}
	)

	tab = strings.ToLower(this.GetString("tab"))
	switch tab {
	case "recommend", "popular", "latest":
	default:
		tab = "latest"
	}

	ModelCate := new(models.Category)
	cates, _ := ModelCate.GetCates(-1, 1)
	cid, _ = this.GetInt("cid")
	pid := cid
	if cid > 0 {
		for _, item := range cates {
			if item.Id == cid {
				if item.Pid > 0 {
					pid = item.Pid
				}
				this.Data["Cate"] = item
				cate = item
				break
			}
		}
	}
	this.Data["Cates"] = cates
	this.Data["Cid"] = cid
	this.Data["Pid"] = pid
	this.TplName = "home/index.html"
	this.Data["IsHome"] = true

	pageIndex, _ := this.GetInt("page", 1)
	//每页显示24个，为了兼容Pad、mobile、PC
	pageSize := 24
	books, totalCount, err := models.NewBook().HomeData(pageIndex, pageSize, models.BookOrder(tab), lang, cid)
	if err != nil {
		beego.Error(err)
		this.Abort("404")
	}
	if totalCount > 0 {
		urlSuffix := "&tab=" + tab
		if cid > 0 {
			urlSuffix = urlSuffix + "&cid=" + strconv.Itoa(cid)
		}
		urlSuffix = urlSuffix + "&lang=" + lang
		html := utils.NewPaginations(conf.RollPage, totalCount, pageSize, pageIndex, urlPrefix, urlSuffix)
		this.Data["PageHtml"] = html
	} else {
		this.Data["PageHtml"] = ""
	}

	this.Data["TotalPages"] = int(math.Ceil(float64(totalCount) / float64(pageSize)))
	this.Data["Lists"] = books
	this.Data["Tab"] = tab
	this.Data["Lang"] = lang
	title := this.Sitename

	desc := this.Sitename + "专注于文档在线写作、协作、分享、阅读与托管，让每个人更方便地发布、分享和获得知识。"
	if cid > 0 {
		title = "[发现] " + cate.Title + " - " + tabName[tab] + " - " + title
		if strings.TrimSpace(cate.Intro) != "" {
			desc = cate.Title + "，" + cate.Intro + " - " + this.Sitename
		}
	} else {
		title = "探索，发现新世界，畅想新知识 - " + this.Sitename
	}

	this.Data["Cate"] = cate

	this.GetSeoByPage("index", map[string]string{
		"title":       title,
		"keywords":    "文档托管,在线创作,文档在线管理,在线知识管理,文档托管平台,在线写书,文档在线转换,在线编辑,在线阅读,开发手册,api手册,文档在线学习,技术文档,在线编辑",
		"description": desc,
	})
}
