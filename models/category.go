package models

import (
	"strings"

	"github.com/TruthHun/BookStack/utils"
	"github.com/astaxie/beego/orm"
	"github.com/kataras/iris/core/errors"
)

var tableCategory = "md_category"

// 分类
type Category struct {
	Id     int    //自增主键
	Pid    int    //分类id
	Title  string `orm:"size(30);unique"` //分类名称
	Intro  string //介绍
	Icon   string //分类icon
	Cnt    int    //分类下的文档项目统计
	Sort   int    //排序
	Status bool   //分类状态，true表示显示，否则表示隐藏
}

//新增分类
func (this *Category) AddCates(pid int, cates string) (err error) {
	if slice := strings.Split(cates, "\n"); len(slice) > 0 {
		o := orm.NewOrm()
		for _, item := range slice {
			if item = strings.TrimSpace(item); item != "" {
				var cate = Category{
					Pid:    pid,
					Title:  item,
					Status: true,
				}
				if o.Read(&cate, "title"); cate.Id == 0 {
					_, err = orm.NewOrm().Insert(&cate)
				}
			}
		}
	}
	return
}

//删除分类（如果分类下的文档项目不为0，则不允许删除）
func (this *Category) Del(id int) (err error) {
	o := orm.NewOrm()
	var cate = Category{Id: id}
	if err = o.Read(&cate); cate.Cnt > 0 { //当前分类下文档项目数量不为0，不允许删除
		return errors.New("删除失败，当前分类下的问下项目不为0，不允许删除")
	}
	if _, err = o.Delete(&cate, "id"); err == nil {
		_, err = o.QueryTable(tableCategory).Filter("pid", id).Delete()
	}
	if err == nil { //删除分类图标
		switch utils.StoreType {
		case utils.StoreOss:
			ModelStoreOss.DelFromOss(cate.Icon)
		case utils.StoreLocal:
			ModelStoreLocal.DelFiles(cate.Icon)
		}
	}
	return
}

//查询所有分类
//@param            pid         -1表示不限（即查询全部），否则表示查询指定pid的分类
//@param            status      -1表示不限状态(即查询所有状态的分类)，0表示关闭状态，1表示启用状态
func (this *Category) GetCates(pid int, status int) (cates []Category, err error) {
	qs := orm.NewOrm().QueryTable(tableCategory)
	if pid > -1 {
		qs = qs.Filter("pid", pid)
	}
	if status == 0 || status == 1 {
		qs = qs.Filter("status", status)
	}
	_, err = qs.OrderBy("-status", "sort", "title").All(&cates)
	return
}

//根据字段更新内容
func (this *Category) UpdateByField(id int, field, val string) (err error) {
	_, err = orm.NewOrm().QueryTable(tableCategory).Filter("id", id).Update(orm.Params{field: val})
	return
}

//查询单个分类
func (this *Category) Find(id int) (cate Category) {
	cate.Id = id
	orm.NewOrm().Read(&cate)
	return cate
}
