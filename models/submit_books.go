package models

import "time"

type SubmitBooks struct {
	Id        int
	Uid       int
	Title     string
	Url       string
	UrlMd5    string `orm:"size(32);unique"`
	Message   string `xorm:"size(512)"`
	Status    bool
	CreatedAt time.Time
	UpdatedAt time.Time
}
