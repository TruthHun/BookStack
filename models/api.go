package models

import (
	"sync"

	"github.com/astaxie/beego"

	"github.com/astaxie/beego/orm"
)

var (
	authCache    sync.Map
	staticDomain string
)

func initAPI() {
	beego.Info(" ===  init api data ===  ")
	authCache = NewAuth().AllFromDatabase()
	staticDomain = "http://localhost:8181/" //TODO: 从数据库中查询并初始化为全局变量
}

func GetAPIStaticDomain() string {
	return staticDomain
}

type Auth struct {
	Id    int
	Token string `orm:"size(32);unique"`
	Uid   int    `orm:"unique"`
}

func NewAuth() *Auth {
	return &Auth{}
}

func (m *Auth) Insert(token string, uid int) (err error) {
	m.DeleteByToken(token)
	m.DeleteByUID(uid)

	var auth = Auth{Token: token, Uid: uid}
	_, err = orm.NewOrm().Insert(&auth)
	if err != nil {
		beego.Error(err.Error())
		return
	}
	authCache.Store(token, auth)
	return
}

// map[token]Auth
func (m *Auth) AllFromCache() (auth sync.Map) {
	return authCache
}

// map[token]Auth
func (m *Auth) AllFromDatabase() (auth sync.Map) {
	o := orm.NewOrm()
	var a []Auth
	o.QueryTable(m).Limit(1000000).All(&a)
	for _, item := range a {
		auth.Store(item.Token, item)
	}
	return
}

func (m *Auth) GetByToken(token string) (auth Auth) {
	val, ok := authCache.Load(token)
	if ok {
		return val.(Auth)
	}
	return
}

func (m *Auth) GetByUID(uid int) (auth Auth) {
	orm.NewOrm().QueryTable(m).Filter("uid", uid).One(&auth)
	if auth.Id > 0 {
		authCache.Store(auth.Token, auth)
	}
	return
}

func (m *Auth) DeleteByToken(token string) {
	authCache.Delete(token)
	orm.NewOrm().QueryTable(m).Filter("token", token).Delete()
}

func (m *Auth) DeleteByUID(uid int) {
	q := orm.NewOrm().QueryTable(m).Filter("uid", uid)
	var auth Auth
	q.One(&auth)
	if auth.Id > 0 {
		q.Delete()
		authCache.Delete(auth.Token)
	}
}
