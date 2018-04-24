package models

//增加一个重新阅读的功能，即重置阅读，清空所有阅读记录

//阅读记录.用于记录阅读的文档，以及阅读进度统计
type ReadRecord struct {
	Id       int //自增主键
	BookId   int `orm:"index"` //书籍id
	DocId    int //文档id
	Uid      int `orm:"index"` //用户id
	CreateAt int //记录创建时间，也就是内容阅读时间
}

//阅读统计
type ReadCount struct {
	Id     int //自增主键
	BookId int //书籍
	Uid    int //用户id
	Cnt    int //阅读的文档数
}
