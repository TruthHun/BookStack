// Package models .
package models

import (
	"strconv"
	"time"

	"github.com/astaxie/beego/orm"
)

type DownloadCounter struct {
	Id    int
	Uid   int `orm:"index"`
	Date  int `orm:"index"`
	Total int
}

func NewDownloadCounter() *DownloadCounter {
	return &DownloadCounter{}
}

func (m *DownloadCounter) Increase(uid int) (err error) {
	now, _ := strconv.Atoi(time.Now().Format("20060102"))
	o := orm.NewOrm()
	o.QueryTable(m).Filter("uid", uid).Filter("date", now).One(m)
	if m.Id == 0 {
		m.Total = 1
		m.Uid = uid
		m.Date = now
		_, err = o.Insert(m)
	} else {
		m.Total = m.Total + 1
		_, err = o.Update(m)
	}
	return
}

// 大于0，表示还可以下载多少个文档
// 小于0，表示没有限制
func (m *DownloadCounter) DoesICanDownload(uid int) (times int, min int) {
	if uid == 0 {
		return
	}

	min, _ = strconv.Atoi(GetOptionValue("DOWNLOAD_INTERVAL", "0"))
	if min <= 0 { // 不限制下载
		return -1, min
	}

	// 查询用户今日阅读时长
	seconds := NewReadingTime().GetReadingTime(uid, PeriodDay)
	times = seconds / (min * 60) // 可下载次数

	if times == 0 {
		return
	}

	orm.NewOrm().QueryTable(m).Filter("uid", uid).Filter("date", time.Now().Format("20060102")).One(m)

	if times > m.Total {
		return times - m.Total, min
	}

	return 0, min
}
