package models

// 阅读时长
type ReadingTime struct {
	Id       int
	Uid      int `orm:"index"`
	Day      int `orm:"index"` // 日期，如 20191212
	Duration int // 每天的阅读时长
}

func NewReadingTime() *ReadingTime {
	return &ReadingTime{}
}
