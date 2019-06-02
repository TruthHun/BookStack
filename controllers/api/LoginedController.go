package api

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/TruthHun/BookStack/models/store"
	"github.com/TruthHun/BookStack/utils"

	"github.com/astaxie/beego"

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

// 更换头像
func (this *LoginedController) ChangeAvatar() {
	_, fh, err := this.GetFile("avatar")
	if err != nil {
		beego.Error(err.Error())
		this.Response(http.StatusBadRequest, messageBadRequest)
	}
	ext := strings.ToLower(filepath.Ext(fh.Filename))
	extMap := map[string]bool{".jpg": true, ".png": true, ".gif": true, ".jpeg": true}
	if _, ok := extMap[ext]; !ok {
		this.Response(http.StatusBadRequest, "图片格式不正确")
	}
	uid := this.isLogin()
	now := time.Now()
	tmp := fmt.Sprintf("uploads/%v-%v%v", now.Format("2006/01"), now.Unix(), ext)
	os.MkdirAll(filepath.Dir(tmp), os.ModePerm)
	defer os.Remove(tmp)
	save := fmt.Sprintf("uploads/%v/%v-%v%v", now.Format("2006/01"), uid, now.Unix(), ext)
	os.MkdirAll(filepath.Dir(save), os.ModePerm)
	err = this.SaveToFile("avatar", tmp)
	if err != nil {
		beego.Error(err.Error())
		this.Response(http.StatusInternalServerError, messageInternalServerError)
	}

	member, err := models.NewMember().Find(uid)
	if err != nil {
		beego.Error(err.Error())
		this.Response(http.StatusInternalServerError, messageInternalServerError)
	}
	avatar := strings.TrimLeft(member.Avatar, "./")
	member.Avatar = "/" + save

	switch utils.StoreType {
	case utils.StoreOss: //oss存储
		err = store.ModelStoreOss.MoveToOss(tmp, strings.TrimLeft(save, "./"), true, false)
		if err != nil {
			beego.Error(err.Error())
		} else {
			store.ModelStoreOss.DelFromOss(avatar)
		}
	case utils.StoreLocal: //本地存储
		err = store.ModelStoreLocal.MoveToStore(tmp, save)
		if err != nil {
			beego.Error(err.Error())
		} else {
			os.Remove(avatar)
		}
	}

	if err = member.Update(); err != nil {
		beego.Error(err.Error())
		this.Response(http.StatusInternalServerError, messageInternalServerError)
	}
	this.Response(http.StatusOK, messageSuccess, map[string]string{"avatar": this.completeLink(save)})
}

func (this *LoginedController) PostComment() {
	content := this.GetString("content")
	if l := len(content); l < 5 || l > 256 {
		this.Response(http.StatusBadRequest, "点评内容限定 5 - 256 个字符")
	}
	bookId, _ := this.GetInt("book_id")

	if bookId <= 0 {
		this.Response(http.StatusBadRequest, messageBadRequest)
	}

	err := new(models.Comments).AddComments(this.isLogin(), bookId, content)
	if err != nil {
		this.Response(http.StatusBadRequest, err.Error())
	}

	// 点评成功之后，再写入评分。如果之前的评分已存在，则不会再重新写入
	score, _ := this.GetInt("score")
	if score > 0 && score <= 5 {
		new(models.Score).AddScore(this.isLogin(), bookId, score)
	}

	this.Response(http.StatusOK, messageSuccess)
}
