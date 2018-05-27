package models

//爬虫
type Spider struct {
	Id         int    //主键
	BookId     int    `orm:"index"` //归属于哪一本数
	Title      string //文档标题
	Content    string `orm:"type(text)"` //内容
	TimeCreate int    //创建时间
	TimeUpdate int    //更新时间
}
