package controllers

type ErrorController struct {
	BaseController
}

func (this *ErrorController) Error404() {
	this.TplName = "errors/404.html"
}

func (this *ErrorController) Error403() {
	this.TplName = "errors/403.html"
}

func (this *ErrorController) Error500() {
	this.TplName = "errors/error.html"
}
