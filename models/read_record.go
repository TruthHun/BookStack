package models

import (
	"time"

	"strconv"

	"github.com/astaxie/beego/orm"
)

//增加一个重新阅读的功能，即重置阅读，清空所有阅读记录

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
	var doc Document
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
		var r ReadRecord
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

//查询阅读记录，返回书籍、文档等相关信息

//查询最近阅读的书籍

//查询收藏书籍(书架上的书籍)的阅读进度(统计等)
