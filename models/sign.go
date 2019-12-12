package models

import (
	"errors"
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/orm"
	"strconv"
	"time"
)

// 会员签到表
type Sign struct {
	Id        int
	Uid       int // 签到的用户id
	Day       int // 签到日期，如20200101
	Reward    int // 奖励的阅读秒数
	FromApp   bool
	CreatedAt time.Time
}

type Rule struct {
	BasicReward         int
	ContinuousReward    int
	AppReward           int
	MaxContinuousReward int
}

var (
	_rule = &Rule{}
)

const (
	signDayLayout       = "20060102"
	messageSigned       = "您今日已签到"
	messageNotExistUser = "您的账户不存在"
	messageSignInnerErr = "签到失败，内部错误"
)

func NewSign() *Sign {
	return &Sign{}
}

// 多字段唯一键
func (m *Sign) TableUnique() [][]string {
	return [][]string{
		[]string{"uid", "day"},
	}
}

// 今天是否已签到
func (m *Sign) IsSignToday(uid int) bool {
	s := &Sign{}
	orm.NewOrm().QueryTable(m).Filter("uid", uid).Filter("day", time.Now().Format(signDayLayout)).One(s, "id")
	return s.Id > 0
}

// 是否未断签
func (m *Sign) IsContinuousSign(uid int) bool {
	s := &Sign{}
	now := time.Now()
	yesterday := now.Add(-24 * time.Hour).Format(signDayLayout)
	orm.NewOrm().QueryTable(m).Filter("uid", uid).Filter("day", yesterday).One(s)
	return s.Id > 0
}

// 执行签到。使用事务
func (m *Sign) Sign(uid int, fromApp bool) (err error) {
	s := &Sign{}
	o := orm.NewOrm()
	now := time.Now()
	day, _ := strconv.Atoi(now.Format(signDayLayout))
	// 1. 检测用户有没有签到
	o.QueryTable(s).Filter("uid", uid).Filter("day", day).One(s)
	if m.IsSignToday(uid) {
		return errors.New(messageSigned)
	}
	isContinuousSign := m.IsContinuousSign(uid) // 昨天有没有断签

	// 2. 查询用户签到了多少天
	user := NewMember()
	cols := []string{"member_id", "total_sign", "total_continuous_sign", "history_total_continuous_sign"}
	o.QueryTable(user).Filter("member_id", uid).One(user, cols...)
	if user.MemberId < 0 {
		return errors.New(messageNotExistUser)
	}
	// 3. 查询奖励规则
	rule := s.GetSignRule()
	// 4. 更新用户签到记录、签到天数和连续签到天数
	o.Begin()
	defer func() {
		if err != nil {
			beego.Error(err)
			err = errors.New(messageSignInnerErr)
			o.Rollback()
		} else {
			o.Commit()
		}
	}()
	user.TotalSign += 1
	s.Day = day
	s.Uid = uid
	s.CreatedAt = now
	s.FromApp = fromApp

	//  奖励计算
	if isContinuousSign { //连续签到
		user.TotalContinuousSign += 1
		extra := user.TotalContinuousSign * rule.ContinuousReward
		if extra >= rule.MaxContinuousReward {
			extra = rule.MaxContinuousReward
		}
		s.Reward = rule.BasicReward + extra
	} else { // 未连续签到
		user.TotalContinuousSign = 1
		s.Reward = rule.BasicReward + rule.ContinuousReward
	}

	if user.TotalContinuousSign > user.HistoryTotalContinuousSign {
		user.HistoryTotalContinuousSign = user.TotalContinuousSign
	}

	if fromApp {
		s.Reward = s.Reward + rule.AppReward
	}

	if _, err = o.Insert(s); err != nil {
		return
	}

	_, err = o.QueryTable(user).Filter("member_id", user.MemberId).Update(orm.Params{
		"total_sign":                    user.TotalSign,
		"total_continuous_sign":         user.TotalContinuousSign,
		"history_total_continuous_sign": user.HistoryTotalContinuousSign,
	})

	return
}

// 获取签到奖励规则
func (m *Sign) GetSignRule() (r *Rule) {
	return _rule
}

// 更新签到奖励规则
func (m *Sign) UpdateSignRule() {
	ops := []string{"SIGN_BASIC_REWARD", "SIGN_APP_REWARD", "SIGN_CONTINUOUS_REWARD", "SIGN_CONTINUOUS_MAX_REWARD"}
	for _, op := range ops {
		num, _ := strconv.Atoi(GetOptionValue(op, ""))
		switch op {
		case "SIGN_BASIC_REWARD":
			_rule.BasicReward = num
		case "SIGN_APP_REWARD":
			_rule.AppReward = num
		case "SIGN_CONTINUOUS_REWARD":
			_rule.ContinuousReward = num
		case "SIGN_CONTINUOUS_MAX_REWARD":
			_rule.MaxContinuousReward = num
		}
	}
}
