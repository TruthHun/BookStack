package models

import (
	"github.com/astaxie/beego/orm"
	"time"
)

// 阅读时长
type ReadingTime struct {
	Id       int
	Uid      int `orm:"index"`
	Day      int `orm:"index"` // 日期，如 20191212
	Duration int // 每天的阅读时长
}

type period string

const (
	PeriodDay      period = "day"
	PeriodWeek     period = "week"
	PeriodLastWeek period = "last-week"
	PeriodMonth    period = "month"
	PeriodLastMoth period = "last-month"
	PeriodAll      period = "all"
	PeriodYear     period = "year"
)

const dateFormat = "20060102"

type sum struct {
	SumVal int
}

type ReadingSortedUser struct {
	Uid      int
	Username string
	Nickname string
	Avatar   string
	SumTime  int
}

func NewReadingTime() *ReadingTime {
	return &ReadingTime{}
}

func (r *ReadingTime) GetReadingTime(uid int, prd period) int {
	sum := &sum{}
	o := orm.NewOrm()
	sqlSum := "select sum(duration) sum_val from md_reading_time where uid = ? and day>=? and day<=? limit 1"
	now := time.Now()
	if prd == PeriodAll {
		m := NewMember()
		o.QueryTable(m).Filter("member_id", uid).One(m, "total_reading_time")
		return m.TotalReadingTime
	}
	start, end := r.getTimeRange(now, prd)
	o.Raw(sqlSum, uid, start, end).QueryRow(sum)
	return sum.SumVal
}

func (r *ReadingTime) Sort(prd period, limit int) (users []ReadingSortedUser) {
	sqlSort := "SELECT t.uid,sum(t.duration) sum_time,m.account,m.avatar,m.nickname FROM `md_reading_time` t left JOIN md_members m on t.uid=m.member_id WHERE t.day>=? and t.day<=? GROUP BY t.uid ORDER BY sum_time desc limit ?"
	start, end := r.getTimeRange(time.Now(), prd)
	orm.NewOrm().Raw(sqlSort, start, end, limit).QueryRows(&users)
	return
}

func (r *ReadingTime) getTimeRange(t time.Time, prd period) (start, end string) {
	switch prd {
	case PeriodWeek:
		start, end = r.getWeek(t)
	case PeriodLastWeek:
		start, end = r.getWeek(t.AddDate(0, 0, -7))
	case PeriodMonth:
		start, end = r.getWeek(t)
	case PeriodLastMoth:
		start, end = r.getWeek(t.AddDate(0, -1, 0))
	case PeriodAll:
		start = "20060102"
		end = "20401231"
	case PeriodDay:
		start = t.Format(dateFormat)
		end = start
	case PeriodYear:
		start, end = r.getWeek(t.AddDate(0, -1, 0))
	default:
		start = t.Format(dateFormat)
		end = start
	}
	return
}

func (*ReadingTime) getWeek(t time.Time) (start, end string) {
	if t.Weekday() == 0 {
		start = t.Add(-7 * 24 * time.Hour).Format(dateFormat)
		end = t.Format(dateFormat)
	} else {
		s := t.Add(-time.Duration(t.Weekday()-1) * 24 * time.Hour)
		start = s.Format(dateFormat)
		end = s.Add(6 * 24 * time.Hour).Format(dateFormat)
	}
	return
}

func (*ReadingTime) getYear(t time.Time) (start, end string) {
	month := time.Date(t.Year(), 1, 1, 0, 0, 0, 0, time.Local)
	start = month.Format(dateFormat)
	end = month.AddDate(0, 12, 0).Add(-24 * time.Hour).Format(dateFormat)
	return
}

func (*ReadingTime) getMonth(t time.Time) (start, end string) {
	month := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.Local)
	start = month.Format(dateFormat)
	end = month.AddDate(0, 1, 0).Add(-24 * time.Hour).Format(dateFormat)
	return
}
