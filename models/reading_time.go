package models

import (
	"github.com/astaxie/beego/orm"
	"time"
)

// 阅读时长
type ReadingTime struct {
	Id       int
	Uid      int
	Day      int // 日期，如 20191212
	Duration int // 每天的阅读时长
}

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

func (*ReadingTime) TableUnique() [][]string {
	return [][]string{[]string{"uid", "day"}}
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
	start, end := getTimeRange(now, prd)
	o.Raw(sqlSum, uid, start, end).QueryRow(sum)
	return sum.SumVal
}

func (r *ReadingTime) Sort(prd period, limit int, withCache ...bool) (users []ReadingSortedUser) {
	sqlSort := "SELECT t.uid,sum(t.duration) sum_time,m.account,m.avatar,m.nickname FROM `md_reading_time` t left JOIN md_members m on t.uid=m.member_id WHERE t.day>=? and t.day<=? GROUP BY t.uid ORDER BY sum_time desc limit ?"
	start, end := getTimeRange(time.Now(), prd)
	orm.NewOrm().Raw(sqlSort, start, end, limit).QueryRows(&users)
	return
}
