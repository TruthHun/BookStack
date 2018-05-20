package models

import (
	"time"

	"strconv"

	"fmt"

	"github.com/astaxie/beego/orm"
	"github.com/kataras/iris/core/errors"
)

//增加一个重新阅读的功能，即重置阅读，清空所有阅读记录
//阅读记录，不允许单个删除，因为这样没意义

//阅读记录.用于记录阅读的文档，以及阅读进度统计
type ReadRecord struct {
	Id       int //自增主键
	BookId   int `orm:"index"` //书籍id
	DocId    int //文档id
	Uid      int `orm:"index"` //用户id
	CreateAt int //记录创建时间，也就是内容阅读时间
}

//阅读统计
type ReadCount struct {
	Id     int //自增主键
	BookId int //书籍
	Uid    int //用户id
	Cnt    int //阅读的文档数
}

//阅读记录列表（非表）
type RecordList struct {
	Title    string
	Identify string
	CreateAt int
}

//阅读进度(非表)
type ReadProgress struct {
	Cnt          int    `json:"cnt"`     //已阅读过的文档
	Total        int    `json:"total"`   //总文档
	Percent      string `json:"percent"` //占的百分比
	BookIdentify string `json:"book_identify"`
}

// 多字段唯一键
func (this *ReadCount) TableUnique() [][]string {
	return [][]string{
		[]string{"BookId", "Uid"},
	}
}

// 多字段唯一键
func (this *ReadRecord) TableUnique() [][]string {
	return [][]string{
		[]string{"DocId", "Uid"},
	}
}

var (
	tableReadRecord = "md_read_record"
	tableReadCount  = "md_read_count"
)

//添加阅读记录
func (this *ReadRecord) Add(docId, uid int) (err error) {
	//1、根据文档id查询书籍id
	//2、写入或者更新阅读记录
	//3、更新书籍被阅读的文档统计
	var (
		doc Document
		r   ReadRecord
	)
	o := orm.NewOrm()
	tableDoc := new(Document)
	err = o.QueryTable(tableDoc).Filter("document_id", docId).One(&doc, "book_id")
	if doc.BookId > 0 { //书籍id大于0
		record := ReadRecord{
			BookId:   doc.BookId,
			DocId:    docId,
			Uid:      uid,
			CreateAt: int(time.Now().Unix()),
		}

		o.QueryTable(tableReadRecord).Filter("doc_id", docId).Filter("uid", uid).One(&r, "id")
		readCnt := 1
		if r.Id > 0 { //先删再增，以便根据主键id索引的倒序查询列表
			o.QueryTable(tableReadRecord).Filter("id", r.Id).Delete()
			readCnt = 0 //如果是更新，则阅读次数
		}
		if _, err = o.Insert(&record); err == nil && readCnt == 1 {
			var rc = ReadCount{
				BookId: doc.BookId,
				Uid:    uid,
				Cnt:    1,
			}
			o.QueryTable(tableReadCount).Filter("uid", uid).Filter("book_id", doc.BookId).One(&rc, "id")
			if rc.Id > 0 { //更新统计
				err = SetIncreAndDecre(tableReadCount, "cnt", "id="+strconv.Itoa(rc.Id), true, 1)
			} else { //增加统计
				_, err = o.Insert(&rc)
			}
		}
	}
	return
}

//清空阅读记录
//当删除文档项目时，直接删除该文档项目的所有记录
func (this *ReadRecord) Clear(uid, bookId int) (err error) {
	o := orm.NewOrm()
	if bookId > 0 && uid > 0 {
		_, err = o.QueryTable(tableReadCount).Filter("uid", uid).Filter("book_id", bookId).Delete()
		if err == nil {
			_, err = o.QueryTable(tableReadRecord).Filter("uid", uid).Filter("book_id", bookId).Delete()
		}
	} else if uid == 0 && bookId > 0 {
		_, err = o.QueryTable(tableReadCount).Filter("book_id", bookId).Delete()
		if err == nil {
			_, err = o.QueryTable(tableReadRecord).Filter("book_id", bookId).Delete()
		}
	}
	return
}

//查询阅读记录
func (this *ReadRecord) List(uid, bookId int) (lists []RecordList, cnt int64, err error) {
	if uid*bookId == 0 {
		err = errors.New("用户id和项目id不能为空")
		return
	}
	fields := "r.create_at,d.document_name title,d.identify"
	sql := "select %v from %v r left join md_documents d on r.doc_id=d.document_id where r.book_id=? and r.uid=? order by r.id desc limit 5000"
	sql = fmt.Sprintf(sql, fields, tableReadRecord)
	cnt, err = orm.NewOrm().Raw(sql, bookId, uid).QueryRows(&lists)
	return
}

//查询阅读进度
func (this *ReadRecord) Progress(uid, bookId int) (rp ReadProgress, err error) {
	if uid*bookId == 0 {
		err = errors.New("用户id和书籍id均不能为空")
		return
	}
	var (
		rc   ReadCount
		book = new(Book)
	)
	o := orm.NewOrm()
	if err = o.QueryTable(tableReadCount).Filter("uid", uid).Filter("book_id", bookId).One(&rc, "cnt"); err == nil {
		if err = o.QueryTable(book).Filter("book_id", bookId).One(book, "doc_count", "identify"); err == nil {
			rp.Total = book.DocCount
		}
	}
	rp.Cnt = rc.Cnt
	rp.BookIdentify = book.Identify
	if rp.Total == 0 {
		rp.Percent = "0.00%"
	} else {
		if rp.Cnt > rp.Total {
			rp.Cnt = rp.Total
		}
		f := float32(rp.Cnt) / float32(rp.Total)
		rp.Percent = fmt.Sprintf("%.2f", f*100) + "%"
	}
	return
}
