package models

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/astaxie/beego"

	"github.com/astaxie/beego/orm"
)

type RelateBook struct {
	Id      int
	BookId  int `orm:"unique"`
	BookIds string
	Expire  int
}

func NewRelateBook() *RelateBook {
	return &RelateBook{}
}

func (r *RelateBook) Lists(bookId int, limit ...int) (books []Book) {

	length := 6
	if len(limit) > 0 && limit[0] > 0 {
		length = limit[0]
	}

	if GetOptionValue("ELASTICSEARCH_ON", "false") != "true" {
		return
	}

	day, _ := strconv.Atoi(GetOptionValue("RELATE_BOOK", "0"))
	if day <= 0 {
		return
	}

	var rb RelateBook
	var ids []int

	now := int(time.Now().Unix())

	o := orm.NewOrm()
	o.QueryTable(r).Filter("book_id", bookId).One(&rb)
	bookModel := NewBook()

	fields := []string{"book_id", "book_name", "cover", "identify"}

	if rb.BookId > 0 && rb.Expire > now {
		if slice := strings.Split(rb.BookIds, ","); len(slice) > 0 {
			for _, item := range slice {
				id, _ := strconv.Atoi(item)
				if id > 0 && len(ids) < length {
					ids = append(ids, id)
				}
			}
			books, _ = bookModel.GetBooksById(ids, fields...)
			return
		}
	}

	book, err := bookModel.Find(bookId)
	if err != nil {
		return
	}

	client := NewElasticSearchClient()
	client.IsRelateSearch = true
	client.Timeout = 1 * time.Second
	wd := book.Label
	if len(wd) == 0 {
		wd = book.BookName
	}
	res, err := client.Search(wd, 1, 12, false)
	if err != nil {
		beego.Error(err.Error())
		return
	}

	var bookIds []string
	for _, item := range res.Hits.Hits {
		if item.Source.Id != bookId {
			if len(ids) < length {
				ids = append(ids, item.Source.Id)
			}
			bookIds = append(bookIds, fmt.Sprint(item.Source.Id))
		}
	}
	books, _ = bookModel.GetBooksById(ids, fields...)
	rb.BookId = bookId
	rb.BookIds = strings.Join(bookIds, ",")
	rb.Expire = now + day*24*3600
	if rb.Id > 0 {
		o.Update(&rb)
	} else {
		o.Insert(&rb)
	}
	return
}
