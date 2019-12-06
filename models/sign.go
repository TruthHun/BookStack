package models

import (
	"strconv"
	"time"
)

// 会员签到表
type Sign struct {
	Id        int
	Uid       int // 签到的用户id
	Day       int // 签到日期，如20200101
	Reward    int // 奖励的阅读秒数
	CreatedAt time.Time
}

type Rule struct {
	BasicReward      int
	ContinuousReward int
	AppReward        int
	ContinuousDay    int
}

var _rule = &Rule{}

func NewSign() *Sign {
	return &Sign{}
}

// 多字段唯一键
func (m *Sign) TableUnique() [][]string {
	return [][]string{
		[]string{"uid", "day"},
	}
}

// TODO:
// 执行签到。使用事务
func (m *Sign) Sign(uid int) {
	// 1. 检测用户有没有签到
	// 2. 检测用户签到了多少天
	// 3. 查询奖励规则
	// 4. 更新用户签到记录、签到天数和连续签到天数
}

// 获取签到奖励规则
func (m *Sign) GetSignRule() (r *Rule) {
	return _rule
}

// 更新签到奖励规则
func (m *Sign) UpdateSignRule() {
	ops := []string{"SIGN_BASIC_REWARD", "SIGN_APP_REWARD", "SIGN_CONTINUOUS_REWARD", "SIGN_CONTINUOUS_DAY"}
	for _, op := range ops {
		num, _ := strconv.Atoi(GetOptionValue(op, ""))
		switch op {
		case "SIGN_BASIC_REWARD":
			_rule.BasicReward = num
		case "SIGN_APP_REWARD":
			_rule.AppReward = num
		case "SIGN_CONTINUOUS_REWARD":
			_rule.ContinuousReward = num
		case "SIGN_CONTINUOUS_DAY":
			_rule.ContinuousDay = num
		}
	}
}
