package models

import (
	"strconv"

	"github.com/astaxie/beego/orm"
)

//书籍与分类关联表，一个书籍可以属于多个分类
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
func (this *BookCategory) SetBookCates(bookId int, cids []string) {

	if len(cids) == 0 {
		return
	}

	var (
		cates             []Category
		tableCategory     = "md_category"
		tableBookCategory = "md_book_category"
	)

	o := orm.NewOrm()
	o.QueryTable(tableCategory).Filter("id__in", cids).All(&cates, "id", "pid")

	cidMap := make(map[string]bool)
	for _, cate := range cates {
		cidMap[strconv.Itoa(cate.Pid)] = true
		cidMap[strconv.Itoa(cate.Id)] = true
	}
	cids = []string{}
	for cid, _ := range cidMap {
		cids = append(cids, cid)
	}

	o.QueryTable(tableBookCategory).Filter("book_id", bookId).Delete()
	var bcs []BookCategory
	for _, cid := range cids {
		cidNum, _ := strconv.Atoi(cid)
		bookCate := BookCategory{
			CategoryId: cidNum,
			BookId:     bookId,
		}
		bcs = append(bcs, bookCate)
	}
	if l := len(bcs); l > 0 {
		o.InsertMulti(l, &bcs)
	}
	go CountCategory()
}
