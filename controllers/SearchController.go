package controllers

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/TruthHun/BookStack/conf"
	"github.com/TruthHun/BookStack/models"
	"github.com/TruthHun/BookStack/utils"
	"github.com/astaxie/beego"
)

type SearchController struct {
	BaseController
}

func (this *SearchController) Index() {
	this.TplName = "search/index.html"
	//如果没有开启你们访问则跳转到登录
	if !this.EnableAnonymous && this.Member == nil {
		this.Redirect(beego.URLFor("AccountController.Login"), 302)
		return
	}

	keyword := this.GetString("keyword")
	this.Redirect(beego.URLFor("LabelController.Index", ":key", keyword), 302)
	return

	//当搜索文档时，直接搜索标签
	pageIndex, _ := this.GetInt("page", 1)
	this.Data["BaseUrl"] = this.BaseUrl()

	if keyword != "" {
		this.Data["Keyword"] = keyword
		memberId := 0
		if this.Member != nil {
			memberId = this.Member.MemberId
		}
		searchResult, totalCount, err := models.NewDocumentSearchResult().FindToPager(keyword, pageIndex, conf.PageSize, memberId)
		if err != nil {
			beego.Error(err)
			return
		}

		if totalCount > 0 {
			html := utils.GetPagerHtml(this.Ctx.Request.RequestURI, pageIndex, conf.PageSize, totalCount)
			this.Data["PageHtml"] = html
		} else {
			this.Data["PageHtml"] = ""
		}

		if len(searchResult) > 0 {
			for _, item := range searchResult {
				item.DocumentName = strings.Replace(item.DocumentName, keyword, "<em>"+keyword+"</em>", -1)
				if item.Description != "" {
					src := item.Description

					//将HTML标签全转换成小写
					re, _ := regexp.Compile("\\<[\\S\\s]+?\\>")
					src = re.ReplaceAllStringFunc(src, strings.ToLower)

					//去除STYLE
					re, _ = regexp.Compile("\\<style[\\S\\s]+?\\</style\\>")
					src = re.ReplaceAllString(src, "")

					//去除SCRIPT
					re, _ = regexp.Compile("\\<script[\\S\\s]+?\\</script\\>")
					src = re.ReplaceAllString(src, "")

					//去除所有尖括号内的HTML代码，并换成换行符
					re, _ = regexp.Compile("\\<[\\S\\s]+?\\>")
					src = re.ReplaceAllString(src, "\n")

					//去除连续的换行符
					re, _ = regexp.Compile("\\s{2,}")
					src = re.ReplaceAllString(src, "\n")

					r := []rune(src)

					if len(r) > 100 {
						src = string(r[:100])
					} else {
						src = string(r)
					}
					item.Description = strings.Replace(src, keyword, "<em>"+keyword+"</em>", -1)
				}

				if item.Identify == "" {
					item.Identify = strconv.Itoa(item.DocumentId)
				}
				if item.ModifyTime.IsZero() {
					item.ModifyTime = item.CreateTime
				}
			}
		}
		this.Data["Lists"] = searchResult
	}
}
