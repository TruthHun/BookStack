package models

import (
	"github.com/astaxie/beego/orm"
	"time"
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
