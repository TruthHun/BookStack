package models

import (
	"github.com/astaxie/beego/orm"
	"strconv"
	"time"
)

type BookCounter struct {
	Id      int
	Bid     int //bookId
	Day     int // 20060102
	StarCnt int
	ViewCnt int
}

type SortedBook struct {
	Id       int
	BookId   int
	Cover    string
	BookName string
	Cnt      int
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

func (*BookCounter) StarSort(prd period, limit int, withCache ...bool) (books []SortedBook) {
	sqlSort := "SELECT sum(c.star_cnt) as cnt,b.book_id,b.cover,b.book_name  FROM `md_star_counter` c left JOIN md_books b on b.book_id=c.bid WHERE c.day>=? and c.day<=?  and b.order_index>=0 GROUP BY c.bid ORDER BY cnt desc limit ?"
	start, end := getTimeRange(time.Now(), prd)
	orm.NewOrm().Raw(sqlSort, start, end, limit).QueryRows(&books)
	return
}

func (*BookCounter) PageViewSort(prd period, limit int, withCache ...bool) (books []SortedBook) {
	sqlSort := "SELECT sum(c.view_cnt) as cnt,b.book_id,b.cover,b.book_name  FROM `md_star_counter` c left JOIN md_books b on b.book_id=c.bid WHERE c.day>=? and c.day<=? and b.order_index>=0 GROUP BY c.bid ORDER BY cnt desc limit ?"
	start, end := getTimeRange(time.Now(), prd)
	orm.NewOrm().Raw(sqlSort, start, end, limit).QueryRows(&books)
	return
}
