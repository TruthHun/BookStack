package models

import (
	"github.com/TruthHun/BookStack/oauth"
	"github.com/astaxie/beego/orm"
)

var ModelGithub = new(Github)

type Github struct {
	oauth.GithubUser
}

//gitee用户的登录流程是这样的
//1、获取gitee的用户信息，用gitee的用户id查询member_id是否大于0，大于0则表示已绑定了用户信息，直接登录
//2、未绑定用户，先把gitee数据入库，然后再跳转绑定页面

//根据giteeid获取用户的gitee数据。这里可以查询用户是否绑定了或者数据是否在库中存在
func (this *Github) GetUserByGithubId(id int, cols ...string) (user Github, err error) {
	qs := orm.NewOrm().QueryTable("md_github").Filter("id", id)
	if len(cols) > 0 {
		err = qs.One(&user, cols...)
	} else {
		err = qs.One(&user)
	}
	return
}

//绑定用户
func (this *Github) Bind(githubId, memberId interface{}) (err error) {
	_, err = orm.NewOrm().QueryTable("md_github").Filter("id", githubId).Update(orm.Params{"member_id": memberId})
	return
}
