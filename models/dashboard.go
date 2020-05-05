package models

import (
	"github.com/astaxie/beego/orm"
	"time"
)

type Dashboard struct {
	BookNumber          int64 `json:"book_number"`
	BookNumberToday     int64 `json:"book_number_today"`
	DocumentNumber      int64 `json:"document_number"`
	DocumentNumberToday int64 `json:"document_number_today"`
	MemberNumber        int64 `json:"member_number"`
	MemberNumberToday   int64 `json:"member_number_today"`
	CommentNumber       int64 `json:"comment_number"`
	CommentNumberToday  int64 `json:"comment_number_today"`
	AttachmentNumber    int64 `json:"attachment_number"`
}

func NewDashboard() *Dashboard {
	return &Dashboard{}
}

func (m *Dashboard) Query() *Dashboard {
	var (
		o       = orm.NewOrm()
		doc     = NewDocument()
		member  = NewMember()
		comment = NewComments()
		book    = NewBook()
		max     = 1000
		layout  = "2006-01-02 00:00:00"
		today   = time.Now().Format(layout)
	)

	bookNumber, _ := o.QueryTable(NewBook().TableNameWithPrefix()).Count()

	m.BookNumber = bookNumber
	m.DocumentNumber, _ = o.QueryTable(NewDocument().TableNameWithPrefix()).Count()
	m.MemberNumber, _ = o.QueryTable(NewMember().TableNameWithPrefix()).Count()
	m.CommentNumber, _ = o.QueryTable(NewComments()).Count()

	//m.AttachmentNumber, _ = o.QueryTable(NewAttachment().TableNameWithPrefix()).Count()

	o.QueryTable(doc).OrderBy("-document_id").One(doc, "document_id")
	o.QueryTable(member).OrderBy("-member_id").One(member, "member_id")
	o.QueryTable(comment).OrderBy("-id").One(comment, "id")
	o.QueryTable(book).OrderBy("-book_id").One(book, "book_id")
	m.DocumentNumberToday, _ = o.QueryTable(doc).Filter("document_id__gte", doc.DocumentId-max).Filter("create_time__gte", today).Count()
	m.MemberNumberToday, _ = o.QueryTable(member).Filter("member_id__gte", member.MemberId-max).Filter("create_time__gte", today).Count()
	m.CommentNumberToday, _ = o.QueryTable(comment).Filter("id__gte", comment.Id-max).Filter("time_create__gte", today).Count()
	m.BookNumberToday, _ = o.QueryTable(book).Filter("book_id__gte", book.BookId-max).Filter("create_time__gte", today).Count()
	return m
}
