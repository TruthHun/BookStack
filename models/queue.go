package models

import (
	"github.com/astaxie/beego/orm"
)

//导出离线文档队列
type Queue struct {
	Id     int  //自增主键
	BookId int  `orm:"index"`      //书籍ID
	Status int8 `orm:"default(0)"` //导出状态，0待导出，1导出中，2导出完成
}

var tableQueue = "md_queue"

//从队列中获取一项
func (*Queue) One(status int8) (q Queue, err error) {
	err = orm.NewOrm().QueryTable(tableQueue).Filter("status", status).One(&q)
	return
}
