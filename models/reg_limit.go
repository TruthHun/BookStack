package models

import (
	"strconv"
	"time"

	"github.com/astaxie/beego"

	"github.com/astaxie/beego/orm"
)

type RegLimit struct {
	Id          int
	Ip          string    `orm:"size(15);index"`
	CreatedAt   time.Time `orm:"index"`
	DailyRegNum int       `orm:"-"`
	HourRegNum  int       `orm:"-"`
	RealIPField string    `orm:"-"`
}

func NewRegLimit() (rl *RegLimit) {
	rl = &RegLimit{}
	var options []Option
	orm.NewOrm().QueryTable(NewOption()).Filter("option_name__in", "REAL_IP_FIELD", "HOUR_REG_NUM", "DAILY_REG_NUM").All(&options, "option_name", "option_value")
	for _, item := range options {
		switch item.OptionName {
		case "REAL_IP_FIELD":
			rl.RealIPField = item.OptionValue
		case "HOUR_REG_NUM":
			rl.HourRegNum, _ = strconv.Atoi(item.OptionValue)
		case "DAILY_REG_NUM":
			rl.DailyRegNum, _ = strconv.Atoi(item.OptionValue)
		}
	}
	return rl
}

func (rl *RegLimit) CheckIPIsAllowed(ip string) (allowHour, allowDaily bool) {
	now := time.Now()
	o := orm.NewOrm()
	if rl.HourRegNum > 0 {
		hourBefore := now.Add(-1 * time.Hour)
		cnt, _ := o.QueryTable(rl).Filter("ip", ip).Filter("created_at__gt", hourBefore).Filter("created_at__lt", now).Count()
		if int(cnt) >= rl.HourRegNum {
			return false, true
		}
	}

	DayBefore := now.Add(-24 * time.Hour)
	if rl.DailyRegNum > 0 {
		cnt, _ := o.QueryTable(rl).Filter("ip", ip).Filter("created_at__gt", DayBefore).Filter("created_at__lt", now).Count()
		if int(cnt) >= rl.DailyRegNum {
			return true, false
		}
	}
	o.QueryTable(rl).Filter("created_at__lt", DayBefore).Delete()
	return true, true
}

func (rl *RegLimit) Insert(ip string) (err error) {
	rl.Ip = ip
	rl.CreatedAt = time.Now()
	if _, err = orm.NewOrm().Insert(rl); err != nil {
		beego.Error(err)
	}
	return
}
