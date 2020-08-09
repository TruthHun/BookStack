package models

import (
	"time"

	"fmt"

	"errors"

	"github.com/astaxie/beego/orm"
)

//书签
type Bookmark struct {
	Id       int
	BookId   int `orm:"index"` //书籍id，主要是为了方便根据书籍id查询书签
	Uid      int //用户id
	DocId    int //文档id
	CreateAt int //创建时间
}

//书签列表
type bookmarkList struct {
	Id           int       `json:"id,omitempty"`
	Title        string    `json:"title"`
	Identify     string    `json:"identify"`
	BookId       int       `json:"book_id"`
	Uid          int       `json:"uid"`
	DocId        int       `json:"doc_id"`
	CreateAt     int       `json:"-"`
	CreateAtTime time.Time `json:"created_at"`
}

var tableBookmark = "md_bookmark"

// 多字段唯一键
func (m *Bookmark) TableUnique() [][]string {
	return [][]string{
		[]string{"Uid", "DocId"},
	}
}

func NewBookmark() *Bookmark {
	return &Bookmark{}
}

//添加或移除书签（如果书签不存在，则添加书签，如果书签存在，则移除书签）
func (m *Bookmark) InsertOrDelete(uid, docId int) (insert bool, err error) {
	if uid*docId == 0 {
		err = errors.New("用户id和文档id均不能为空")
		return
	}

	o := orm.NewOrm()

	var (
		bookmark Bookmark
		doc      = new(Document)
	)

	o.QueryTable(tableBookmark).Filter("uid", uid).Filter("doc_id", docId).One(&bookmark, "id")
	if bookmark.Id > 0 { //删除书签
		_, err = o.QueryTable(tableBookmark).Filter("id", bookmark.Id).Delete()
		return
	}

	//新增书签
	//查询文档id是属于哪个书籍
	o.QueryTable(doc).Filter("document_id", docId).One(doc, "book_id")
	bookmark.BookId = doc.BookId
	bookmark.CreateAt = int(time.Now().Unix())
	bookmark.Uid = uid
	bookmark.DocId = docId
	_, err = o.Insert(&bookmark)
	insert = true
	return
}

//查询书签是否存在
func (m *Bookmark) Exist(uid, docId int) (exist bool) {
	if uid*docId > 0 {
		bk := new(Bookmark)
		orm.NewOrm().QueryTable(bk).Filter("uid", uid).Filter("doc_id", docId).One(bk, "id")
		return bk.Id > 0
	}
	return
}

//删除书签
//1、只有 bookId > 0，则删除bookId所有书签【用于书籍被删除的情况】
//2、bookId>0 && uid > 0 ，删除用户的书籍书签【用户用户清空书签的情况】
//3、uid > 0 && docId>0 ，删除指定书签【用于删除某条书签】
//4、其余情况不做处理
func (m *Bookmark) Delete(uid, bookId, docId int) (err error) {
	q := orm.NewOrm().QueryTable(tableBookmark)
	if bookId > 0 {
		_, err = q.Filter("book_id", bookId).Delete()
	} else if bookId > 0 && uid > 0 {
		_, err = q.Filter("book_id", bookId).Filter("uid", uid).Delete()
	} else if uid > 0 && docId > 0 {
		_, err = q.Filter("doc_id", docId).Filter("uid", uid).Delete()
	}
	return
}

//查询书签列表
func (m *Bookmark) List(uid, bookId int) (bl []bookmarkList, rows int64, err error) {
	o := orm.NewOrm()
	fields := "b.id,d.document_name title,d.identify,b.book_id,b.uid,b.doc_id,b.create_at"
	sql := "select %v from md_bookmark b left join md_documents d on b.doc_id=d.document_id where b.uid=? and b.book_id=? order by b.id desc limit 1000"
	sql = fmt.Sprintf(sql, fields)
	rows, err = o.Raw(sql, uid, bookId).QueryRows(&bl)
	return
}
