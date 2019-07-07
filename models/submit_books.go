package models

import (
	"time"

	"github.com/astaxie/beego"
	"golang.org/x/tools/go/ssa/interp/testdata/src/errors"

	"github.com/TruthHun/gotil/cryptil"
	"github.com/astaxie/beego/orm"
)

type SubmitBooks struct {
	Id        int
	Uid       int    `orm:"index"`
	Title     string `form:"title"`
	Url       string `form:"url"`
	UrlMd5    string `orm:"size(32);unique"`
	Message   string `orm:"size(512)" form:"message"`
	Status    bool
	CreatedAt time.Time
	UpdatedAt time.Time
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
