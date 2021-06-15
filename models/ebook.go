package models

import (
	"errors"
	"time"

	"github.com/astaxie/beego"
	"github.com/astaxie/beego/orm"
)

type Ebook struct {
	Id            int
	Title         string    // 电子书名称
	Keywords      string    // 关键字
	Description   string    // 摘要
	Path          string    // 文件路径。如果是网站生成的电子书，则为电子书的路径，否则为URL地址
	BookID        int       `orm:"default(0);column(book_id);index"` // 所属书籍ID
	Ext           string    `orm:"size(8);index"`                    // 文件扩展名
	Status        int       `orm:"default(0);index"`                 // 0：待处理； 1: 转换中；2: 转换完成
	Size          int64     `orm:"default(0)"`                       // 电子书大小
	DownloadCount int       `orm:"default(0)"`                       // 电子书被下载次数
	CreatedAt     time.Time `orm:"auto_now_add;type(datetime)"`
	UpdatedAt     time.Time `orm:"auto_now;type(datetime)"`
}

var convert2ebookRunning = false

const (
	EBookStatusPending     = 0 // 待处理
	EBookStatusProccessing = 1 // 处理中
	EBookStatusSuccess     = 2 // 转换成功
	EBookStatusFailure     = 3 // 失败
)

func NewEbook() *Ebook {
	return &Ebook{}
}

func (m *Ebook) GetEBookByBookID(bookID int) (books []Ebook) {
	if bookID <= 0 {
		return
	}

	if _, err := orm.NewOrm().QueryTable(m).Filter("book_id", bookID).All(&books); err != nil && err != orm.ErrNoRows {
		beego.Error(err)
	}
	return
}

func (m *Ebook) GetEBook(id int) (book Ebook) {
	if id <= 0 {
		return
	}
	err := orm.NewOrm().QueryTable(m).Filter("id", id).One(&book)
	if err != nil {
		beego.Error(err)
	}
	return
}

// 添加书籍到电子书生成队列
func (m *Ebook) AddToGenerate(bookID int) (err error) {
	var (
		ebooks []Ebook
		exts   = []string{".pdf", ".mobi", ".epub"}
	)

	b, _ := NewBook().Find(bookID)
	if b == nil || b.BookId == 0 {
		return errors.New("书籍不存在")
	}
	for _, ext := range exts {
		ebooks = append(ebooks, Ebook{
			Title:       b.BookName,
			Keywords:    b.Label,
			Description: b.Description,
			BookID:      bookID,
			Ext:         ext,
			Status:      EBookStatusPending,
		})
	}

	if _, err = orm.NewOrm().InsertMulti(len(ebooks), &ebooks); err != nil {
		beego.Error(err)
	}
	return
}

// 电子书状态（最新的状态）
func (m *Ebook) Stats(bookID int) (stats map[string]Ebook) {
	var (
		ebooks []Ebook
		limit  = 4 // 先默认为4，即四个扩展名：.pdf,.epub,.mobi,.docx
	)
	stats = make(map[string]Ebook)
	o := orm.NewOrm()
	o.QueryTable(m).Filter("book_id", bookID).OrderBy("-id").Limit(limit).All(&ebooks)
	if len(ebooks) == 0 {
		return
	}

	for _, ebook := range ebooks {
		if _, ok := stats[ebook.Ext]; !ok {
			stats[ebook.Ext] = ebook
		}
	}
	return
}

// 查询书籍是否处于完成状态。失败也是完成状态的一种。
func (m *Ebook) IsFinish(bookID int) (ok bool) {
	count, err := orm.NewOrm().QueryTable(m).Filter("book_id", bookID).Filter("status__in", EBookStatusPending, EBookStatusProccessing).Count()
	if err != nil {
		beego.Error(err)
		return
	}
	return count == 0
}

// 生成电子书
func (m *Ebook) convert2ebook() {
	if convert2ebookRunning {
		return
	}
	convert2ebookRunning = true
	o := orm.NewOrm()
	o.QueryTable(m).Filter("book_id__gt", 0).Filter("status", EBookStatusProccessing).Update(orm.Params{"status": EBookStatusPending})
	for {
		var ebook Ebook
		o.QueryTable(m).Filter("book_id__gt", 0).Filter("status", EBookStatusPending).OrderBy("id").One(&ebook)
		if ebook.Id > 0 {
			// 根据电子书的ID，查找现有的电子书的队列
		}
		time.Sleep(5 * time.Second)
	}
}
