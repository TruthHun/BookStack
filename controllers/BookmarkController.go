package controllers

import (
	"time"

	"github.com/TruthHun/BookStack/models"
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/orm"
)

type BookmarkController struct {
	BaseController
}

func (this *BookmarkController) Prepare() {
	this.BaseController.Prepare()
	if this.Member.MemberId == 0 {
		this.JsonResult(1, "请先登录")
	}
}

//添加或者移除书签
func (this *BookmarkController) Bookmark() {
	docId, _ := this.GetInt(":id")
	if docId <= 0 {
		this.JsonResult(1, "收藏失败，文档id参数错误")
	}

	insert, err := new(models.Bookmark).InsertOrDelete(this.Member.MemberId, docId)
	if err != nil {
		beego.Error(err.Error())
		if insert {
			this.JsonResult(1, "添加书签失败", insert)
		}
		this.JsonResult(1, "移除书签失败", insert)
	}

	if insert {
		this.JsonResult(0, "添加书签成功", insert)
	}
	this.JsonResult(0, "移除书签成功", insert)
}

//获取书签列表
func (this *BookmarkController) List() {
	bookId, _ := this.GetInt(":book_id")
	if bookId <= 0 {
		this.JsonResult(1, "获取书签列表失败：参数错误")
	}

	bl, rows, err := new(models.Bookmark).List(this.Member.MemberId, bookId)
	if err != nil {
		beego.Error(err.Error())
		this.JsonResult(1, "获取书签列表失败")
	}

	var (
		book  = new(models.Book)
		lists []map[string]interface{}
	)

	orm.NewOrm().QueryTable(book).Filter("book_id", bookId).One(book, "identify")
	for _, item := range bl {
		var list = make(map[string]interface{})
		list["url"] = beego.URLFor("DocumentController.Read", ":key", book.Identify, ":id", item.Identify)
		list["title"] = item.Title
		list["doc_id"] = item.DocId
		list["del"] = beego.URLFor("BookmarkController.Bookmark", ":id", item.DocId)
		list["time"] = time.Unix(int64(item.CreateAt), 0).Format("01-02 15:04")
		lists = append(lists, list)
	}

	this.JsonResult(0, "获取书签列表成功", map[string]interface{}{
		"count":   rows,
		"book_id": bookId,
		"list":    lists,
	})
}
