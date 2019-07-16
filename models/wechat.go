package models

import (
	"fmt"

	"github.com/TruthHun/gotil/cryptil"
	"github.com/astaxie/beego/orm"
)

type Wechat struct {
	Id        int
	MemberId  int    //绑定的用户id
	Openid    string `orm:"unique;size(50)"`
	Unionid   string `orm:"size(50)"`
	AvatarURL string `orm:"column(avatar_url)"`
	Nickname  string `orm:"size(30)"`
	SessKey   string `orm:"size(50);unique"`
}

func NewWechat() *Wechat {
	return &Wechat{}
}

//根据giteeid获取用户的gitee数据。这里可以查询用户是否绑定了或者数据是否在库中存在
func (m *Wechat) GetUserByOpenid(openid string, cols ...string) (user Wechat, err error) {
	qs := orm.NewOrm().QueryTable(m).Filter("openid", openid)
	if len(cols) > 0 {
		err = qs.One(&user, cols...)
	} else {
		err = qs.One(&user)
	}
	return
}

//根据giteeid获取用户的gitee数据。这里可以查询用户是否绑定了或者数据是否在库中存在
func (m *Wechat) GetUserBySess(sessKey string, cols ...string) (user Wechat, err error) {
	qs := orm.NewOrm().QueryTable(m).Filter("sess_key", sessKey)
	if len(cols) > 0 {
		err = qs.One(&user, cols...)
	} else {
		err = qs.One(&user)
	}
	return
}

func (m *Wechat) Insert() (err error) {
	o := orm.NewOrm()
	exist := &Wechat{}
	o.QueryTable(m).Filter("openid", m.Openid).One(exist)
	if exist.Id > 0 {
		exist.SessKey = m.SessKey
		_, err = o.Update(exist)
	} else {
		_, err = o.Insert(m)
	}
	return
}

//绑定用户
func (m *Wechat) Bind(openid, memberId interface{}) (err error) {
	_, err = orm.NewOrm().QueryTable(m).Filter("openid", openid).Filter("member_id", 0).Update(orm.Params{"member_id": memberId, "sess_key": cryptil.Md5Crypt(fmt.Sprint(openid))})
	return
}
