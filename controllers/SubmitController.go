package controllers

import (
	"strings"

	"github.com/TruthHun/BookStack/models"
	"github.com/astaxie/beego"
)

type SubmitController struct {
	BaseController
}

func (this *SubmitController) Index() {
	this.Data["SeoTitle"] = "开源书籍和文档收录"
	this.Data["IsSubmit"] = true
	this.TplName = "submit/index.html"
}

func (this *SubmitController) Post() {
	uid := this.Member.MemberId
	if uid <= 0 {
		this.JsonResult(1, "请先登录")
	}

	form := &models.SubmitBooks{}
	err := this.ParseForm(form)
	if err != nil {
		beego.Error(err.Error())
		this.JsonResult(1, "数据解析失败")
	}

	lowerURL := strings.ToLower(form.Url)
	if !(strings.HasPrefix(lowerURL, "https://") || strings.HasPrefix(lowerURL, "http://")) {
		this.JsonResult(1, "URL链接地址格式不正确")
	}

	if form.Url == "" || form.Title == "" {
		this.JsonResult(1, "请填写必填项")
	}
	form.Uid = uid
	if err = form.Add(); err != nil {
		this.JsonResult(1, err.Error())
	}
	this.JsonResult(0, "提交成功，感谢您的分享。")
}
