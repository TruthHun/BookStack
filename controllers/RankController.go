package controllers

type RankController struct {
	BaseController
}

func (this *RankController) Index() {
	this.Data["IsRank"] = true
	this.TplName = "rank/index.html"
}
