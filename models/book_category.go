package models

import (
	"strconv"

	"fmt"
	"strings"

	"github.com/astaxie/beego/orm"
)

//文档项目与分类关联表，一个文档项目可以属于多个分类
type BookCategory struct {
	Id         int //自增主键
	BookId     int //书籍id
	CategoryId int //分类id
}

// 多字段唯一键
func (this *BookCategory) TableUnique() [][]string {
	return [][]string{
		[]string{"BookId", "CategoryId"},
	}
}

//根据书籍id查询分类id
func (this *BookCategory) GetByBookId(book_id int) (cates []Category, rows int64, err error) {
	o := orm.NewOrm()
	sql := "select c.* from md_category c left join md_book_category bc on c.id=bc.category_id where bc.book_id=?"
	rows, err = o.Raw(sql, book_id).QueryRows(&cates)
	return
}

//处理书籍分类
func (this *BookCategory) SetBookCates(book_id int, cids []string) {
	var (
		cates []Category
		bc    []BookCategory

		oldCids           []string
		tableCategory     = "md_category"
		tableBookCategory = "md_book_category"
	)
	o := orm.NewOrm()
	//1、查找当前分类的父级分类
	o.QueryTable(tableCategory).Filter("id__in", cids).All(&cates, "id", "pid")
	for _, cate := range cates {
		cids = append(cids, strconv.Itoa(cate.Pid))
	}
	//2、删除原有的分类关系，并减少计数
	qs := o.QueryTable(tableBookCategory).Filter("book_id", book_id)
	qs.All(&bc) //查询
	qs.Delete() //删除
	//减少计数
	for _, c := range bc {
		oldCids = append(oldCids, strconv.Itoa(c.CategoryId))
	}
	SetIncreAndDecre(tableCategory, "cnt", fmt.Sprintf("id in(%v)", strings.Join(oldCids, ",")), false)
	//3、新增现在的分类关系，并增加计数
	SetIncreAndDecre(tableCategory, "cnt", fmt.Sprintf("id in(%v)", strings.Join(cids, ",")), true) //计算增加
	for _, cid := range cids {                                                                      //这里逐条添加记录，不是批量添加，因为设置了唯一键，批量添加可能会导致全部都添加失败
		cidNum, _ := strconv.Atoi(cid)
		var bookCate = BookCategory{
			CategoryId: cidNum,
			BookId:     book_id,
		}
		o.Insert(&bookCate)
	}
}
