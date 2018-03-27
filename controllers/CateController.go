package controllers

import (
	"github.com/TruthHun/BookStack/models"
	"github.com/astaxie/beego"
)

type CateController struct {
	BaseController
}

//分类
func (this *CateController) List() {
	this.Data["IsCate"] = true
	if cates, err := new(models.Category).GetCates(-1, 1); err == nil {
		this.Data["Cates"] = cates
	} else {
		beego.Error(err.Error())
	}
	this.TplName = "cates/list.html"
}
