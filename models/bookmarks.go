package models

//书签
type BookMarks struct {
	Id       int
	BookId   int //书籍id
	Uid      int //用户id
	DocId    int //文档id
	CreateAt int //创建时间
}
