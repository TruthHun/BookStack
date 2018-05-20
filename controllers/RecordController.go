package controllers

import (
	"time"

	"github.com/TruthHun/BookStack/models"
	"github.com/astaxie/beego"
)

type RecordController struct {
	BaseController
}

func (this *RecordController) Prepare() {
	this.BaseController.Prepare()
	if this.Member.MemberId == 0 {
		this.JsonResult(1, "请先登录")
	}
}

//获取书签列表
func (this *RecordController) List() {
	var (
		lists   []map[string]interface{}
		err     error
		rl      []models.RecordList
		rp      models.ReadProgress
		errcode int
		message string = "数据查询成功"
		count   int64
	)
	bookId, _ := this.GetInt(":book_id")
	if bookId > 0 {
		m := new(models.ReadRecord)
		if rl, count, err = m.List(this.Member.MemberId, bookId); err == nil && len(rl) > 0 {
			rp, _ = m.Progress(this.Member.MemberId, bookId)
			for _, item := range rl {
				var list = make(map[string]interface{})
				list["title"] = item.Title
				list["url"] = beego.URLFor("DocumentController.Read", ":key", rp.BookIdentify, ":id", item.Identify)
				list["time"] = time.Unix(int64(item.CreateAt), 0).Format("01-02 15:04")
				lists = append(lists, list)
			}
		}
	}
	if len(lists) == 0 {
		errcode = 1
		message = "您当前没有阅读记录"
	}
	this.JsonResult(errcode, message, map[string]interface{}{
		"lists":    lists,
		"count":    count,
		"progress": rp,
		"clear":    beego.URLFor("RecordController.Clear", ":book_id", bookId),
	})
}

//重置阅读进度(清空阅读历史)
func (this *RecordController) Clear() {
	bookId, _ := this.GetInt(":book_id")
	if bookId > 0 {
		m := new(models.ReadRecord)
		m.Clear(this.Member.MemberId, bookId)
	}
	this.JsonResult(0, "阅读进度重置成功")
}
