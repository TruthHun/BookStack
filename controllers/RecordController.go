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

//获取阅读记录列表
func (this *RecordController) List() {
	var (
		lists   []map[string]interface{}
		err     error
		rl      []models.RecordList
		rp      models.ReadProgress
		errCode int
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
				list["del"] = beego.URLFor("RecordController.Delete", ":doc_id", item.DocId)
				lists = append(lists, list)
			}
		}
	}
	if len(lists) == 0 {
		errCode = 1
		message = "您当前没有阅读记录"
	}
	this.JsonResult(errCode, message, map[string]interface{}{
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
		if err := m.Clear(this.Member.MemberId, bookId); err != nil {
			beego.Error(err)
		}
	}
	//不管删除是否成功，均返回成功
	this.JsonResult(0, "重置阅读进度成功")
}

//删除单条阅读历史
func (this *RecordController) Delete() {
	docId, _ := this.GetInt(":doc_id")
	if docId > 0 {
		if err := new(models.ReadRecord).Delete(this.Member.MemberId, docId); err != nil {
			beego.Error(err)
		}
	}
	this.JsonResult(0, "删除成功")
}
