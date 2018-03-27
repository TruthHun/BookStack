package controllers

type CateController struct {
	BaseController
}

//分类
func (this *CateController) List() {
	this.Data["IsCate"] = true
	this.TplName = "cates/list.html"
}
