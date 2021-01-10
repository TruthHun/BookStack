package models

import (
	"strings"

	"github.com/TruthHun/BookStack/utils"

	"github.com/astaxie/beego"

	"github.com/astaxie/beego/orm"
)

var (
	staticDomain     string
	maxLoginTerminal = 10 //允许最大的登录数（这里暂时写死，后面再设置为可以从数据库中进行配置）
)

func initAPI() {
	if strings.ToLower(utils.StoreType) == utils.StoreOss {
		staticDomain = strings.TrimSpace(beego.AppConfig.String("oss::Domain"))
	}

	if strings.TrimRight(staticDomain, "/") == "" {
		staticDomain = beego.AppConfig.DefaultString("static_domain", "")
	}

	staticDomain = strings.TrimRight(staticDomain, "/") + "/"
}

func GetAPIStaticDomain() string {
	return staticDomain
}

type Auth struct {
	Id    int
	Token string `orm:"size(32);unique"`
	Uid   int    `orm:"index"`
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
	m.clearMoreThanLimit(uid)
	return
}

func (m *Auth) clearMoreThanLimit(uid int) {
	if maxLoginTerminal <= 0 {
		return
	}
	q := orm.NewOrm().QueryTable(m)
	var auths []Auth
	q.Filter("uid", uid).OrderBy("-id").Limit(maxLoginTerminal).All(&auths, "id")
	if len(auths) == maxLoginTerminal {
		q.Filter("uid", uid).Filter("id__lt", auths[maxLoginTerminal-1].Id).Delete()
	}
}

func (m *Auth) GetByToken(token string) (auth Auth) {
	orm.NewOrm().QueryTable(m).Filter("token", token).One(&auth)
	return
}

func (m *Auth) DeleteByToken(token string) {
	orm.NewOrm().QueryTable(m).Filter("token", token).Delete()
}
