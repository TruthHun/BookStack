package models

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/astaxie/beego/orm"
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
	Uid              int    `json:"uid"`
	Account          string `json:"account"`
	Nickname         string `json:"nickname"`
	Avatar           string `json:"avatar"`
	SumTime          int    `json:"sum_time"`
	TotalReadingTime int    `json:"total_reading_time"`
}

const (
	readingTimeCacheDir = "cache/rank/reading-time"
	readingTimeCacheFmt = "cache/rank/reading-time/%v-%v.json"
)

func init() {
	if _, err := os.Stat(readingTimeCacheDir); err != nil {
		err = os.MkdirAll(readingTimeCacheDir, os.ModePerm)
		if err != nil {
			panic(err)
		}
	}
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
	var b []byte
	cache := false
	if len(withCache) > 0 {
		cache = withCache[0]
	}
	file := fmt.Sprintf(readingTimeCacheFmt, prd, limit)
	if cache {
		if info, err := os.Stat(file); err == nil && time.Now().Sub(info.ModTime()).Seconds() <= cacheTime {
			// 文件存在，且在缓存时间内
			if b, err = ioutil.ReadFile(file); err == nil {
				json.Unmarshal(b, &users)
				if len(users) > 0 {
					return
				}
			}
		}
	}

	sqlSort := "SELECT t.uid,sum(t.duration) sum_time,m.account,m.avatar,m.nickname FROM `md_reading_time` t left JOIN md_members m on t.uid=m.member_id WHERE m.no_rank=0 and t.day>=? and t.day<=? GROUP BY t.uid ORDER BY sum_time desc limit ?"
	start, end := getTimeRange(time.Now(), prd)
	orm.NewOrm().Raw(sqlSort, start, end, limit).QueryRows(&users)

	if cache && len(users) > 0 {
		b, _ = json.Marshal(users)
		ioutil.WriteFile(file, b, os.ModePerm)
	}
	return
}
