// Package models .
package models

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
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

// DoesICanDownload 用户是否可以下载电子书
// availableTimes 表示剩余可下载次数。负数表示不限制，否则表示限制可下载的次数
func (m *DownloadCounter) DoesICanDownload(uid int, wecode string) (availableTimes int, err error) {
	availableTimes = -1
	downCode := GetOptionValue("DOWNLOAD_WECODE", "")
	if wecode != "" {
		if strings.TrimSpace(wecode) == strings.TrimSpace(downCode) {
			return
		}
		err = errors.New("下载码不正确：请重新输入或重新获取下载码")
		return
	}

	minute, _ := strconv.Atoi(strings.TrimSpace(GetOptionValue("DOWNLOAD_INTERVAL", "0")))
	if minute <= 0 { // 不限制下载
		return
	}

	// 查询用户今日阅读时长
	seconds := NewReadingTime().GetReadingTime(uid, PeriodDay)
	availableTimes = seconds / (minute * 60) // 可下载次数
	if availableTimes == 0 {
		err = fmt.Errorf("每天每阅读学习 %v 分钟可下载1个离线文档。您今日阅读时长不足，无法再下载。请输入【下载码】进行下载或者继续阅读以增加阅读时长。", minute)
		return
	}

	orm.NewOrm().QueryTable(m).Filter("uid", uid).Filter("date", time.Now().Format("20060102")).One(m)

	if availableTimes > m.Total {
		availableTimes = availableTimes - m.Total
		return
	}

	err = fmt.Errorf("每天每阅读学习 %v 分钟可下载1个离线文档。您今日阅读时长不足，无法再下载。请输入【下载码】进行下载或者继续阅读以增加阅读时长。", minute)
	return
}
