package models

type RankType string

// 会员榜单
const (
	RankTypeTotalSignIn      RankType = "total-sign-in"      // 累计签到天数排行
	RankTypeContinuousSignIn RankType = "continuous-sign-in" // 连续签到天数排行
	RankTypeLatestReading    RankType = "latest-reading"     // 最近半小时或者一两个消失内的阅读排行
	RankTypeDailyReading     RankType = "daily-reading"      // 今日阅读排行
	RankTypeWeeklyReading    RankType = "weekly-reading"     // 本周阅读排行
	RankTypeMothReading      RankType = "month-reading"      // 本月阅读排行
)

type Rank struct {
	Id       int
	RankType string `orm:"size(30);index"` // 排名类型
	RankNO   int    // 名次
	Uid      int    // 用户id
	Score    int    // 分数。如果是阅读排行，则是阅读的秒数；如果是签到排行，则是签到的天数
}
