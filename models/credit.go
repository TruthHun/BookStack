package models

import (
	"time"

	"github.com/astaxie/beego/orm"
)

type Credit struct {
}

const (
	cycleTypeOnce  int8 = 0 // 一次
	cycleTypeDay   int8 = 1 // 一天
	cycleTypeWeek  int8 = 2 // 一周
	cycleTypeMonth int8 = 3 // 一月
	cycleTypeYear  int8 = 4 // 一年
)

// CreditRule 规则
type CreditRule struct {
	Id          int    // 规则ID
	Identify    string `orm:"unique;size(32)"` // 积分标识，唯一
	Title       string // 规则名称
	Intro       string // 规则简介
	Score       int    // 周期内每次奖励的积分
	CycleType   int8   // 奖励周期
	RewardTimes int    // 周次内总共奖励的次数
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// CreditLog 积分变更日志记录
type CreditLog struct {
	Id        int
	Uid       int       `orm:"index"`          // 用户ID
	Identify  string    `orm:"index;size(32)"` // 规则标识
	Score     int       // 积分，有正有负数，负数表示被扣分，正数表示加分
	RewardBy  int       // 被谁奖励的，0 表示为系统奖励的，如果 Uid == RewardBy，表示用户自己充值的(如果有这个功能的话)，否则表示是管理员(需要是管理员身份)打赏的
	Log       string    `orm:"size(512)"` // 日志内容
	CreatedAt time.Time `orm:"auto_now"`
}

// InitCreditRule 如果还没存在积分规则，则初始化积分规则
func InitCreditRule() {
	// 积分榜单增加 壕榜(积分排行)
	// 一个勤奋的用户，每个月大概能获取到的奖励金额：0.005x30(签到, 0.15) + 0.005(注册,0.005) + 0.008x30(邀请注册,0.24) + 0.032x30(日榜上榜, 0.96) + 0.064x4(周榜上榜0.26) + 1.0(月榜前十) = 3
	rules := []CreditRule{
		{Identify: "sign", Title: "签到奖励", Intro: "用户每天签到的奖励", CycleType: cycleTypeDay, RewardTimes: 1, Score: 5},                      // 每天签到奖励， 0.005 元
		{Identify: "reg", Title: "注册奖励", Intro: "新用户注册奖励", CycleType: cycleTypeOnce, RewardTimes: 1, Score: 8},                        // 注册奖励
		{Identify: "read", Title: "阅读奖励", Intro: "重在参与，学习有奖！用户每天阅读时长超过5分钟的奖励", CycleType: cycleTypeDay, RewardTimes: 1, Score: 5},     // 重在参与，阅读有奖。 0.005 元
		{Identify: "invite", Title: "邀请注册", Intro: "邀请别人注册，获得奖励", CycleType: cycleTypeDay, RewardTimes: 1, Score: 8},                  // 0.008 元
		{Identify: "rank_day", Title: "阅读日榜前五十奖励", Intro: "用户阅读日榜上榜(前50)奖励", CycleType: cycleTypeDay, RewardTimes: 1, Score: 32},      // 0.032 x 50 x 30 = 48 / 月
		{Identify: "rank_week", Title: "阅读周榜前五十奖励", Intro: "用户阅读周榜上榜(前50)奖励", CycleType: cycleTypeWeek, RewardTimes: 1, Score: 64},    // 0.128 x 50 x 4 = 12.8 / 月
		{Identify: "rank_mon_top50", Title: "阅读月榜前五十奖励", Intro: "用户阅读月榜前五十奖励", CycleType: cycleTypeMonth, RewardTimes: 1, Score: 256}, // 0.256 x 30 = 7.68 / 月
		{Identify: "rank_mon_top20", Title: "阅读月榜前二十奖励", Intro: "用户阅读月榜前二十奖励", CycleType: cycleTypeMonth, RewardTimes: 1, Score: 512}, // 0.5 x 10 =  5 / 月
		{Identify: "rank_mon_top10", Title: "阅读月榜前十奖励", Intro: "用户阅读月榜前十奖励", CycleType: cycleTypeMonth, RewardTimes: 1, Score: 1024},  // 1.0 x 10 = 10 / 月

		{Identify: "donate", Title: "打赏书籍", Intro: "用户打赏书籍消费", CycleType: cycleTypeDay, RewardTimes: 10, Score: 0},          // score 为0，表示隐藏这条积分记录
		{Identify: "exchange", Title: "积分兑换", Intro: "用户积分兑换奖品", CycleType: cycleTypeMonth, RewardTimes: 20, Score: -10240}, // score 为0，表示隐藏这条积分记录

		{Identify: "submit", Title: "收录奖励", Intro: "提交未被收录的书籍获得的奖励", CycleType: cycleTypeDay, RewardTimes: 10, Score: 5}, // 收录奖励， 0.005 元
		{Identify: "charge", Title: "充值奖励", Intro: "用户充值", CycleType: cycleTypeDay, RewardTimes: 0, Score: 0},            // 手工充值 0.005 元

		{Identify: "market_value", Title: "网站积分市值", Intro: "整站所产生的积分", CycleType: cycleTypeOnce, RewardTimes: 0, Score: 0}, // 网站积分总市值  0.005 元
	}
	o := orm.NewOrm()
	for _, rule := range rules {
		o.Insert(&rule)
	}
}

func NewCreditRule() *CreditRule {
	return &CreditRule{}
}

func NewCreditLog() *CreditLog {
	return &CreditLog{}
}

func NewCredit() *Credit {
	return &Credit{}
}

// Insert 给用户新增积分
func (m *Credit) Insert(uid int, ruleIdentify string) {

}

// CreditCostPerMonth 每月成本估算，price 表示一元钱等于多少个积分。
// 计算结果计算网站一个月大概支出多少，以及一个勤奋的用户大概能拿到多少等价金额
// 注意：为避免积分规则设置错误，或者是积分规则设置者“滥发货币”引起不必要的“破产问题”，
//		建议积分每个月限定总额兑换机制，兑换完就自行等待下次兑换。具体可以参考微信支付出行奖励积分兑换机制玩法，
//		避免积分在跟实体金额兑换的时候玩死自己....
func (m *Credit) CreditCostPerMonth(price int) {
	// TODO
	_ = price
}
