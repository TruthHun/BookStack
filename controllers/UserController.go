package controllers

type UserController struct {
	BaseController
}

//首页
func (this *UserController) Index() {
	this.TplName = "user/index.html"
}

//收藏
func (this *UserController) Collection() {

}

//粉丝和关注
func (this *UserController) Follow() {

}

//粉丝和关注
func (this *UserController) Fans() {

}
