package controllers

import (
	"github.com/TruthHun/BookStack/models"
	"github.com/astaxie/beego"
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
	if docId > 0 {
		if insert, err := new(models.Bookmark).InsertOrDelete(this.Member.MemberId, docId); err == nil {
			if insert {
				this.JsonResult(0, "添加书签成功", insert)
			} else {
				this.JsonResult(0, "删除书签成功", insert)
			}
		} else {
			beego.Error(err.Error())
			if insert {
				this.JsonResult(1, "添加书签失败", insert)
			} else {
				this.JsonResult(1, "移除书签失败", insert)
			}
		}
	} else {
		this.JsonResult(1, "收藏失败，文档id参数错误")
	}
}

//获取书签列表
func (this *BookmarkController) List() {
	bookId, _ := this.GetInt(":book_id")
	if bookId > 0 {
		if bl, rows, err := new(models.Bookmark).List(this.Member.MemberId, bookId); err != nil {
			beego.Error(err.Error())
			this.JsonResult(1, "获取书签列表失败")
		} else {
			this.JsonResult(0, "获取书签列表成功", map[string]interface{}{
				"count": rows,
				"list":  bl,
			})
		}
	} else {
		this.JsonResult(1, "获取书签列表失败：参数错误")
	}
}
