package models

import (
	"github.com/astaxie/beego/orm"
)

const (
	QUEUE_STATUS_PADDING   int8 = 0 //待导出
	QUEUE_STATUS_EXPORTING int8 = 1 //导出中
	QUEUE_STATUS_FINISHED  int8 = 2 //导出完成
	QUEUE_STATUS_FAILED    int8 = 3 //导出失败
)

//导出离线文档队列
type Queue struct {
	Id     int  //自增主键
	BookId int  `orm:"unique"`     //书籍ID
	Status int8 `orm:"default(0)"` //导出状态，0待导出，1导出中，2导出完成，3导出失败
}

var tableQueue = "md_queue"

//从队列中获取一项
func (*Queue) one(status int8) (q Queue, err error) {
	err = orm.NewOrm().QueryTable(tableQueue).Filter("status", status).One(&q)
	return
}

//设置
func (*Queue) set(bookId int, status int8) (err error) {
	_, err = orm.NewOrm().QueryTable(tableQueue).Filter("status", status).Update(orm.Params{"status": status})
	return
}
