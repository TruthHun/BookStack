package models

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/TruthHun/BookStack/models/store"

	"github.com/TruthHun/BookStack/utils"
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/orm"
)

// 微信小程序码记录表
type WechatCode struct {
	Id     int
	BookId int    `orm:"unique"`
	Path   string `orm:"default()"`
}

func NewWechatCode() (code *WechatCode) {
	return &WechatCode{}
}

func (this *WechatCode) GetCode(bookId int) (path string) {
	m := NewWechatCode()
	orm.NewOrm().QueryTable(m).Filter("book_id", bookId).One(m)
	return m.Path
}

func (this *WechatCode) Delete(bookId int) {
	orm.NewOrm().QueryTable(this).Filter("book_id", bookId).Delete()
}

// 调用分钟频率受限（5000次/分钟），如需大量小程序码，建议预生成\
// 生成微信小程序码
func (this *WechatCode) CreateWechatCode(bookId int) {
	o := orm.NewOrm()
	m := NewWechatCode()
	o.QueryTable(m).Filter("book_id", bookId).One(m)
	if m.Id > 0 {
		return
	}
	// 先写入一条空数据，如果生成小程序码失败，则再把记录删除
	m.BookId = bookId
	_, err := o.Insert(m)
	if err != nil {
		beego.Error(err.Error())
		return
	}
	defer func() {
		if err != nil {
			o.QueryTable(m).Filter("id", m.Id).Delete()
		}
	}()
	book := NewBook()
	o.QueryTable(book).Filter("book_id", bookId).One(book, "identify", "book_id")
	if book.BookId == 0 {
		err = errors.New("书籍不存在")
		return
	}

	// 生成小程序二维码
	tmpFile := ""
	accessToken := utils.GetAccessToken()
	tmpFile, err = utils.GetBookWXACode(accessToken.AccessToken, bookId)
	if err != nil {
		beego.Error(err.Error())
		return
	}
	defer os.Remove(tmpFile) // 删除临时文件
	ext := filepath.Ext(tmpFile)
	m.Path = fmt.Sprintf("projects/%v/wxacode%v", book.Identify, ext)
	switch utils.StoreType {
	case utils.StoreOss: //oss存储
		err = store.ModelStoreOss.MoveToOss(tmpFile, m.Path, true, false)
		if err != nil {
			beego.Error(err.Error())
			return
		}
	case utils.StoreLocal: //本地存储
		m.Path = "uploads/" + m.Path
		err = store.ModelStoreLocal.MoveToStore(tmpFile, m.Path)
		if err != nil {
			beego.Error(err.Error())
			return
		}
	}
	m.Path = "/" + m.Path
	_, err = o.Update(m)
	if err != nil {
		beego.Error(err.Error())
	}
	return
}
