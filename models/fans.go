package models

import (
	"fmt"

	"github.com/astaxie/beego/orm"
)

var tableFans = "md_fans"

type FansResult struct {
	Uid      int
	Nickname string
	Avatar   string
	Account  string
}

//粉丝表
type Fans struct {
	Id     int //自增主键
	Uid    int `orm:"index"` //被关注的用户id
	FansId int `orm:"index"` //粉丝id
}

// 多字段唯一键
func (this *Fans) TableUnique() [][]string {
	return [][]string{
		[]string{"Uid", "FansId"},
	}
}

//关注和取消关注
func (this *Fans) FollowOrCancel(uid, fansId int) (cancel bool, err error) {
	var fans Fans
	o := orm.NewOrm()
	qs := o.QueryTable(tableFans).Filter("uid", uid).Filter("fans_id", fansId)
	qs.One(&fans)
	if fans.Id > 0 { //已关注，则取消关注
		_, err = qs.Delete()
		cancel = true
	} else { //未关注，则新增关注
		fans.Uid = uid
		fans.FansId = fansId
		_, err = o.Insert(&fans)
	}
	return
}

//查询是否已经关注了用户
func (this *Fans) Relation(uid, fansId interface{}) (ok bool) {
	var fans Fans
	orm.NewOrm().QueryTable(tableFans).Filter("uid", uid).Filter("fans_id", fansId).One(&fans)
	return fans.Id != 0
}

//查询用户的粉丝（用户id作为被关注对象）
func (this *Fans) GetFansList(uid, page, pageSize int) (fans []FansResult, total int64, err error) {
	o := orm.NewOrm()
	total, _ = o.QueryTable(tableFans).Filter("uid", uid).Count()
	if total > 0 {
		sql := fmt.Sprintf(
			"select m.member_id uid,m.avatar,m.account,m.nickname from md_members m left join md_fans f on m.member_id=f.fans_id where f.uid=?  order by f.id desc limit %v offset %v",
			pageSize, (page-1)*pageSize,
		)
		_, err = o.Raw(sql, uid).QueryRows(&fans)
	}
	return
}

//查询用户的关注（用户id作为fans_id）
func (this *Fans) GetFollowList(fansId, page, pageSize int) (fans []FansResult, total int64, err error) {
	o := orm.NewOrm()
	total, _ = o.QueryTable(tableFans).Filter("fans_id", fansId).Count()
	if total > 0 {
		sql := fmt.Sprintf(
			"select m.member_id uid,m.avatar,m.account,m.nickname from md_members m left join md_fans f on m.member_id=f.uid where f.fans_id=?  order by f.id desc limit %v offset %v",
			pageSize, (page-1)*pageSize,
		)
		_, err = o.Raw(sql, fansId).QueryRows(&fans)
	}
	return
}
