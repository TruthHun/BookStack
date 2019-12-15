package controllers

import "github.com/TruthHun/BookStack/models"

type RankController struct {
	BaseController
}

func (this *RankController) Index() {
	limit := 10
	sign := models.NewSign()
	this.Data["TotalReadingUsers"] = sign.SortedContinuousSign(limit)
	this.Data["IsRank"] = true
	this.TplName = "rank/index.html"
}
