package models

import "github.com/astaxie/beego/orm"

type Dashboard struct {
	BookNumber       int64 `json:"book_number"`
	DocumentNumber   int64 `json:"document_number"`
	MemberNumber     int64 `json:"member_number"`
	CommentNumber    int64 `json:"comment_number"`
	AttachmentNumber int64 `json:"attachment_number"`
}

func NewDashboard() *Dashboard {
	return &Dashboard{}
}

func (m *Dashboard) Query() *Dashboard {
	o := orm.NewOrm()

	bookNumber, _ := o.QueryTable(NewBook().TableNameWithPrefix()).Count()

	m.BookNumber = bookNumber
	m.DocumentNumber, _ = o.QueryTable(NewDocument().TableNameWithPrefix()).Count()

	m.MemberNumber, _ = o.QueryTable(NewMember().TableNameWithPrefix()).Count()

	//comment_number,_ := o.QueryTable(NewComment().TableNameWithPrefix()).Count()
	m.CommentNumber = 0

	m.AttachmentNumber, _ = o.QueryTable(NewAttachment().TableNameWithPrefix()).Count()

	return m
}
