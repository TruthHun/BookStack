package models

import (
	"strings"
	"time"

	"github.com/astaxie/beego"

	"github.com/astaxie/beego/orm"
)

var (
	staticDomain string
)

func initAPI() {
	staticDomain = strings.TrimRight(beego.AppConfig.DefaultString("static_domain", "https://static.bookstack.cn/"), "/") + "/"
}

func GetAPIStaticDomain() string {
	return staticDomain
}

type Auth struct {
	Id        int
	Token     string `orm:"size(32);unique"`
	Uid       int
	CreatedAt time.Time
}

func NewAuth() *Auth {
	return &Auth{}
}

func (m *Auth) Insert(token string, uid int) (err error) {
	m.DeleteByToken(token)
	var auth = Auth{Token: token, Uid: uid}
	_, err = orm.NewOrm().Insert(&auth)
	if err != nil {
		beego.Error(err.Error())
		return
	}
	return
}

func (m *Auth) GetByToken(token string) (auth Auth) {
	orm.NewOrm().QueryTable(m).Filter("token", token).One(&auth)
	return
}

func (m *Auth) DeleteByToken(token string) {
	orm.NewOrm().QueryTable(m).Filter("token", token).Delete()
}
