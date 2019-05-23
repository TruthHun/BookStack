package controllers

import (
	"fmt"
	"strings"
	"time"

	"github.com/TruthHun/BookStack/conf"
	"github.com/TruthHun/BookStack/utils"

	"github.com/TruthHun/BookStack/models"
	"github.com/astaxie/beego"
)

type SearchController struct {
	BaseController
}

//搜索首页
func (this *SearchController) Search() {
	if wd := strings.TrimSpace(this.GetString("wd")); wd != "" {
		this.Redirect(beego.URLFor("LabelController.Index", ":key", wd), 302)
		return
	}
	this.Data["SeoTitle"] = "搜索 - " + this.Sitename
	this.Data["IsSearch"] = true
	this.TplName = "search/search.html"
}

// 搜索结果页
func (this *SearchController) Result() {

	totalRows := 0

	var ids []int

	wd := this.GetString("wd")
	if wd == "" {
		this.Redirect(beego.URLFor("SearchController.Search"), 302)
		return
	}

	now := time.Now()

	tab := this.GetString("tab", models.GetOptionValue("DEFAULT_SEARCH", "book"))
	isSearchDoc := false
	if tab == "doc" {
		isSearchDoc = true
	}

	page, _ := this.GetInt("page", 1)
	size := 10

	if page < 1 {
		page = 1
	}

	client := models.NewElasticSearchClient()

	if client.On { // elasticsearch 进行全文搜索
		result, err := models.NewElasticSearchClient().Search(wd, page, size, isSearchDoc)
		if err != nil {
			beego.Error(err.Error())
		} else { // 搜索结果处理
			totalRows = result.Hits.Total
			for _, item := range result.Hits.Hits {
				ids = append(ids, item.Source.Id)
			}
		}
	} else { //MySQL like 查询
		if isSearchDoc { //搜索文档
			docs, count, err := models.NewDocumentSearchResult().SearchDocument(wd, 0, page, size)
			totalRows = count
			if err != nil {
				beego.Error(err.Error())
			} else {
				for _, doc := range docs {
					ids = append(ids, doc.DocumentId)
				}
			}
		} else { //搜索书籍
			books, count, err := models.NewBook().SearchBook(wd, page, size)
			totalRows = count
			if err != nil {
				beego.Error(err.Error())
			} else {
				for _, book := range books {
					ids = append(ids, book.BookId)
				}
			}
		}
	}
	if len(ids) > 0 {
		if isSearchDoc {
			this.Data["Docs"], _ = models.NewDocumentSearchResult().GetDocsById(ids)
		} else {
			this.Data["Books"], _ = models.NewBook().GetBooksById(ids)
		}
		this.Data["Words"] = client.SegWords(wd)
	}

	this.Data["TotalRows"] = totalRows
	if totalRows > size {
		if totalRows > 1000 {
			totalRows = 1000
		}
		urlSuffix := fmt.Sprintf("&tab=%v&wd=%v", tab, wd)
		html := utils.NewPaginations(conf.RollPage, totalRows, size, page, beego.URLFor("SearchController.Result"), urlSuffix)
		this.Data["PageHtml"] = html
	} else {
		this.Data["PageHtml"] = ""
	}
	this.Data["SpendTime"] = fmt.Sprintf("%.3f", time.Since(now).Seconds())
	this.Data["Wd"] = wd
	this.Data["Tab"] = tab
	this.Data["IsSearch"] = true
	this.TplName = "search/result.html"
}
