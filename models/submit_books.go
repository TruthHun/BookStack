package models

import (
	"fmt"
	"time"

	"errors"

	"github.com/astaxie/beego"

	"github.com/TruthHun/gotil/cryptil"
	"github.com/astaxie/beego/orm"
)

type SubmitBooks struct {
	Id  int
	Uid int `orm:"index"`
	// 注意: nickname 和 account 是不存储用户昵称和账户的。orm:"-" 的时候，beego自带的orm没法将用户的数据匹配到字段上来，所以用这种勉强的方式
	Nickname     string `orm:"default();size(1)"`
	Account      string `orm:"default();size(1)"`
	Title        string `form:"title"`
	Url          string `form:"url"`
	UrlMd5       string `orm:"size(32);unique"`
	Message      string `orm:"size(512)" form:"message"`
	Status       bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
	CreatedAtStr string `orm:"-"`
}

func NewSubmitBooks() *SubmitBooks {
	return &SubmitBooks{}
}

func (m *SubmitBooks) Add() (err error) {
	o := orm.NewOrm()
	m.UrlMd5 = cryptil.Md5Crypt(m.Url)
	m.CreatedAt = time.Now()
	m.UpdatedAt = m.CreatedAt
	exist := &SubmitBooks{}
	o.QueryTable(m).Filter("url_md5", m.UrlMd5).One(exist)
	if exist.Id > 0 { //已存在，直接返回
		return
	}

	o.QueryTable(m).Filter("uid", m.Uid).OrderBy("-id").One(exist)
	if exist.Id > 0 && m.CreatedAt.Unix()-exist.CreatedAt.Unix() < 60 { // TODO: 后期从配置中查询
		err = errors.New("提交频率过快，1分钟内仅限提交一次")
		return
	}

	_, err = o.Insert(m)
	if err != nil {
		beego.Error(err.Error())
		err = errors.New("数据写入失败")
	}

	return
}

func (m *SubmitBooks) Lists(page, size int, status ...bool) (books []SubmitBooks, total int64, err error) {
	o := orm.NewOrm()
	q := o.QueryTable(m)
	where := ""
	if len(status) > 0 {
		q = q.Filter("status", status[0])
		if status[0] == true {
			where = " where status = 1 "
		} else {
			where = "where status = 0 "
		}
	}
	total, err = q.Count()
	if err != nil {
		beego.Error(err)
		return
	}
	if total == 0 {
		return
	}

	querySQL := fmt.Sprintf("select b.*,u.nickname,u.account from md_submit_books b left join md_members u on u.member_id = b.uid %v order by b.id desc limit %v offset %v",
		where, size, (page-1)*size,
	)
	_, err = o.Raw(querySQL).QueryRows(&books)
	if err != nil {
		beego.Error(err.Error())
	}
	for idx, book := range books {
		book.CreatedAtStr = book.CreatedAt.Format("2006-01-02 15:04:05")
		books[idx] = book
	}
	return
}
