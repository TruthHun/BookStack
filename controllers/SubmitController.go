package controllers

type SubmitController struct {
	BaseController
}

func (this *SubmitController) Index() {
	this.Data["Title"] = "开源书籍和文档收录"
	this.Data["IsSubmit"] = true
	this.TplName = "submit/index.html"
}

func (this *SubmitController) Post() {

}
