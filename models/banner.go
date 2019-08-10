package models

import (
	"time"

	"github.com/TruthHun/BookStack/utils"

	"github.com/astaxie/beego/orm"
)

type Banner struct {
	Id        int       `json:"id"`
	Type      string    `orm:"size(30);index" json:"type" description:"横幅类型，如 wechat（小程序）,pc（PC端）,mobi（移动端）等"`
	Title     string    `json:"title" orm:"size(100)"`
	Link      string    `json:"link"`
	Image     string    `json:"image"`
	Sort      int       `json:"sort"`
	Status    bool      `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

func NewBanner() *Banner {
	return &Banner{}
}

func (m *Banner) Lists(t string) (banners []Banner, err error) {
	_, err = orm.NewOrm().QueryTable(m).Filter("type", t).Filter("status", true).OrderBy("-sort", "-id").All(&banners)
	if err == orm.ErrNoRows {
		err = nil
	}
	return
}

func (m *Banner) All() (banners []Banner, err error) {
	_, err = orm.NewOrm().QueryTable(m).OrderBy("-sort", "-status").All(&banners)
	if err == orm.ErrNoRows {
		err = nil
	}
	return
}

func (m *Banner) Update(id int, field string, value interface{}) (err error) {
	_, err = orm.NewOrm().QueryTable(m).Filter("id", id).Update(orm.Params{field: value})
	if err == orm.ErrNoRows {
		err = nil
	}
	return
}

func (m *Banner) Delete(id int) (err error) {
	var banner Banner
	q := orm.NewOrm().QueryTable(m).Filter("id", id)
	q.One(&banner)
	if banner.Id > 0 {
		_, err = q.Delete()
		if err == nil {
			utils.DeleteFile(banner.Image)
		}
	}
	return
}
