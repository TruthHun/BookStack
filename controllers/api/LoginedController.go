package api

import (
	"net/http"

	"github.com/TruthHun/BookStack/models"
)

// 登录之后才能调用的接口放这里
type LoginedController struct {
	BaseController
}

func (this *LoginedController) Prepare() {
	this.BaseController.Prepare()
	if models.NewAuth().GetByToken(this.Token).Uid == 0 {
		this.Response(http.StatusUnauthorized, messageRequiredLogin)
	}
}

func (this *LoginedController) UserInfo() {

}

func (this *LoginedController) Logout() {
	models.NewAuth().DeleteByToken(this.Token)
	this.Response(http.StatusOK, messageLogoutSuccess)
}

func (this *LoginedController) GetBookmarks() {
	bookId, _ := this.GetInt("book_id")
	if bookId <= 0 {
		this.Response(http.StatusOK, messageSuccess, []string{})
	}

	lists, _, _ := models.NewBookmark().List(this.isLogin(), bookId)
	if len(lists) > 0 {
		this.Response(http.StatusOK, messageSuccess, lists)
	}
	this.Response(http.StatusOK, messageSuccess, []string{})
}

func (this *LoginedController) SetBookmarks() {
	docId, _ := this.GetInt("doc_id")
	if docId <= 0 {
		this.Response(http.StatusBadRequest, messageBadRequest)
	}
	bm := models.NewBookmark()
	if !bm.Exist(this.isLogin(), docId) {
		bm.InsertOrDelete(this.isLogin(), docId)
	}
	this.Response(http.StatusOK, messageSuccess)
}

func (this *LoginedController) DeleteBookmarks() {
	docId, _ := this.GetInt("doc_id")
	if docId <= 0 {
		this.Response(http.StatusBadRequest, messageBadRequest)
	}
	bm := models.NewBookmark()
	if bm.Exist(this.isLogin(), docId) {
		bm.InsertOrDelete(this.isLogin(), docId)
	}
	this.Response(http.StatusOK, messageSuccess)
}
