package models

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/astaxie/beego/orm"
)

type BookCounter struct {
	Id      int
	Bid     int //bookId
	Day     int // 20060102
	StarCnt int
	ViewCnt int
}

type SortedBook struct {
	Id       int    `json:"id"`
	BookId   int    `json:"book_id"`
	Identify string `json:"identify"`
	Cover    string `json:"cover"`
	BookName string `json:"book_name"`
	Cnt      int    `json:"cnt"`
}

const (
	bookCounterCacheDir = "cache/rank/book-counter"
	bookCounterCacheFmt = "cache/rank/book-counter/%v-%v.json"
)

func init() {
	if _, err := os.Stat(bookCounterCacheDir); err != nil {
		err = os.MkdirAll(bookCounterCacheDir, os.ModePerm)
		if err != nil {
			panic(err)
		}
	}
}
func NewBookCounter() *BookCounter {
	return &BookCounter{}
}

func (*BookCounter) TableUnique() [][]string {
	return [][]string{[]string{"bid", "day"}}
}

func (*BookCounter) Increase(bookId int, isPV bool) {
	sc := NewBookCounter()
	today, _ := strconv.Atoi(time.Now().Format(dateFormat))
	o := orm.NewOrm()
	o.QueryTable(sc).Filter("bid", bookId).Filter("day", today).One(sc)
	if sc.Id == 0 {
		sc = &BookCounter{
			Bid: bookId,
			Day: today,
		}
		if isPV {
			sc.ViewCnt = 1
		} else {
			sc.StarCnt = 1
		}
		o.Insert(sc)
	} else {
		if isPV {
			sc.ViewCnt += 1
		} else {
			sc.StarCnt += 1
		}
		o.Update(sc)
	}
}

func (*BookCounter) Decrease(bookId int, isPV bool) {
	sc := NewBookCounter()
	today, _ := strconv.Atoi(time.Now().Format(dateFormat))
	o := orm.NewOrm()
	o.QueryTable(sc).Filter("bid", bookId).Filter("day", today).One(sc)
	if sc.Id == 0 {
		sc = &BookCounter{
			Bid: bookId,
			Day: today,
		}
		if isPV {
			sc.ViewCnt = 1
		} else {
			sc.StarCnt = 1
		}
		o.Insert(sc)
	} else {
		if isPV {
			if sc.ViewCnt > 0 {
				sc.ViewCnt -= 1
			}
		} else {
			if sc.StarCnt > 0 {
				sc.StarCnt -= 1
			}
		}
		o.Update(sc)
	}
}

func (m *BookCounter) StarSort(prd period, limit int, withCache ...bool) (books []SortedBook) {
	return m._sort(prd, limit, "star", withCache...)
}

func (m *BookCounter) PageViewSort(prd period, limit int, withCache ...bool) (books []SortedBook) {
	return m._sort(prd, limit, "vcnt", withCache...)
}

func (*BookCounter) _sort(prd period, limit int, orderField string, withCache ...bool) (books []SortedBook) {
	field := "vcnt" // 浏览
	if orderField != "vcnt" {
		field = "star" // 收藏
	}

	if prd == PeriodAll {
		books2 := NewBook().Sorted(limit, field)
		for _, book := range books2 {
			cnt := book.Vcnt
			if field != "vcnt" {
				cnt = book.Star
			}
			books = append(books, SortedBook{
				BookId:   book.BookId,
				Identify: book.Identify,
				Cover:    strings.ReplaceAll(book.Cover, "\\", "/"),
				BookName: book.BookName,
				Cnt:      cnt,
			})
		}
		return
	}

	var b []byte

	cache := false
	if len(withCache) > 0 {
		cache = withCache[0]
	}

	file := fmt.Sprintf(bookCounterCacheFmt, string(prd)+"-"+field, limit)

	if cache {
		if info, err := os.Stat(file); err == nil && time.Now().Sub(info.ModTime()).Seconds() <= cacheTime {
			// 文件存在，且在缓存时间内
			if b, err = ioutil.ReadFile(file); err == nil {
				json.Unmarshal(b, &books)
				if len(books) > 0 {
					return
				}
			}
		}
	}

	sqlSort := "SELECT sum(c.view_cnt) as cnt,b.book_id,b.identify,b.cover,b.book_name  FROM `md_book_counter` c left JOIN md_books b on b.book_id=c.bid WHERE c.day>=? and c.day<=? and b.order_index>=0 and b.privately_owned=0 GROUP BY c.bid ORDER BY cnt desc limit ?"
	if field == "star" {
		sqlSort = "SELECT sum(c.star_cnt) as cnt,b.book_id,b.identify,b.cover,b.book_name  FROM `md_book_counter` c left JOIN md_books b on b.book_id=c.bid WHERE c.day>=? and c.day<=?  and b.order_index>=0 and b.privately_owned=0 GROUP BY c.bid ORDER BY cnt desc limit ?"
	}

	start, end := getTimeRange(time.Now(), prd)
	orm.NewOrm().Raw(sqlSort, start, end, limit).QueryRows(&books)

	if cache && len(books) > 0 {
		b, _ = json.Marshal(books)
		ioutil.WriteFile(file, b, os.ModePerm)
	}

	return
}
