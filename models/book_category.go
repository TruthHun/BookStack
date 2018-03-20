package models

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
