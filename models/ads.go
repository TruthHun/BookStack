package models

import (
	"github.com/astaxie/beego/orm"
	"sync"
	"time"
)

// 广告位
type AdsPosition struct {
	Id       int
	Title    string
	Identify string `orm:"index;size(32)"`
	IsMobile bool   `orm:"index"`
}

// 广告
type AdsCont struct {
	Id        int
	Pid       int `orm:"index"`
	Title     string
	Code      string `orm:"size(4096)"`
	Start     int
	StartTime string `orm:"-"`
	End       int
	EndTime   string `orm:"-"`
	Status    bool
}

func NewAdsPosition() *AdsPosition {
	return &AdsPosition{}
}

func NewAdsCont() *AdsCont {
	return &AdsCont{}
}

const (
	AdsPositionBeforeFriendLink     = "global-before-friend-link"
	AdsPositionGlobalFooter         = "global-footer"
	AdsPositionUnderLatestRecommend = "index-under-latest-recommend"
	AdsPositionSearchRight          = "search-right"
	AdsPositionSearchTop            = "search-top"
	AdsPositionSearchBottom         = "search-bottom"
	AdsPositionBeforeMenu           = "intro-before-menu"
	AdsPositionBeforeRelatedBooks   = "intro-before-related-books"
	AdsPositionUnderExploreCate     = "explore-under-cate"
	AdsPositionContentTop           = "content-top"
	AdsPositionContentBottom        = "content-bottom"
)

var adsCache sync.Map // map[positionIdentify][]AdsCont

func InitAdsPosition() {
	positions := []AdsPosition{
		{
			IsMobile: false,
			Title:    "[全局]页面底部",
			Identify: AdsPositionGlobalFooter,
		},
		{
			IsMobile: false,
			Title:    "[友链]顶部",
			Identify: AdsPositionBeforeFriendLink,
		},
		{
			IsMobile: false,
			Title:    "[首页]最新推荐下方",
			Identify: AdsPositionUnderLatestRecommend,
		},
		{
			IsMobile: false,
			Title:    "[搜索页]搜索结果右侧",
			Identify: AdsPositionSearchRight,
		},
		{
			IsMobile: false,
			Title:    "[搜索页]搜索结果上方",
			Identify: AdsPositionSearchTop,
		},
		{
			IsMobile: false,
			Title:    "[搜索页]搜索结果下方",
			Identify: AdsPositionSearchBottom,
		},
		{
			IsMobile: false,
			Title:    "[书籍介绍页]菜单上方",
			Identify: AdsPositionBeforeMenu,
		},
		{
			IsMobile: false,
			Title:    "[书籍介绍页]相关书籍上方",
			Identify: AdsPositionBeforeRelatedBooks,
		},
		{
			IsMobile: false,
			Title:    "[内容阅读页]内容上方",
			Identify: AdsPositionContentTop,
		},
		{
			IsMobile: false,
			Title:    "[内容阅读页]内容下方",
			Identify: AdsPositionContentBottom,
		},
		{
			IsMobile: false,
			Title:    "[发现页]分类下方",
			Identify: AdsPositionUnderExploreCate,
		},
		{
			IsMobile: true,
			Title:    "[全局]页面底部",
			Identify: AdsPositionGlobalFooter,
		},
		{
			IsMobile: true,
			Title:    "[友链]顶部",
			Identify: AdsPositionBeforeFriendLink,
		},
		{
			IsMobile: true,
			Title:    "[首页]最新推荐下方",
			Identify: AdsPositionUnderLatestRecommend,
		},
		{
			IsMobile: true,
			Title:    "[搜索页]搜索结果上方",
			Identify: AdsPositionSearchTop,
		},
		{
			IsMobile: true,
			Title:    "[搜索页]搜索结果下方",
			Identify: AdsPositionSearchBottom,
		},
		{
			IsMobile: true,
			Title:    "[书籍介绍页]菜单上方",
			Identify: AdsPositionBeforeMenu,
		},
		{
			IsMobile: true,
			Title:    "[书籍介绍页]相关书籍上方",
			Identify: AdsPositionBeforeRelatedBooks,
		},
		{
			IsMobile: true,
			Title:    "[发现页]分类下方",
			Identify: AdsPositionUnderExploreCate,
		},
		{
			IsMobile: true,
			Title:    "[内容阅读页]内容上方",
			Identify: AdsPositionContentTop,
		},
		{
			IsMobile: true,
			Title:    "[内容阅读页]内容下方",
			Identify: AdsPositionContentBottom,
		},
	}
	o := orm.NewOrm()
	for _, position := range positions {
		table := &AdsPosition{}
		o.QueryTable(table).Filter("is_mobile", position.IsMobile).Filter("identify", position.Identify).One(table)
		if table.Id == 0 {
			o.Insert(&position)
		}
	}
}

func (m *AdsCont) GetPositions() []AdsPosition {
	var positions []AdsPosition
	orm.NewOrm().QueryTable(NewAdsPosition()).OrderBy("is_mobile").All(&positions)
	return positions
}

func (m *AdsCont) Lists(isMobile bool, status ...bool) (ads []AdsCont) {
	var (
		positions     []AdsPosition
		pids          []interface{}
		tablePosition = NewAdsPosition()
		tableAds      = NewAdsCont()
	)
	o := orm.NewOrm()
	o.QueryTable(tablePosition).Filter("is_mobile", isMobile).All(&positions)
	for _, p := range positions {
		pids = append(pids, p.Id)
	}

	q := o.QueryTable(tableAds).Filter("pid__in", pids...)
	if len(status) > 0 {
		q = q.Filter("status", status[0])
	}
	q.All(&ads)
	layout := "2006-01-02"
	for idx, ad := range ads {
		ad.StartTime = time.Unix(int64(ad.Start), 0).Format(layout)
		ad.EndTime = time.Unix(int64(ad.End), 0).Format(layout)
		ads[idx] = ad
	}
	return
}

func (m *AdsCont) GetAdsCode(positionIdentify string, isMobile bool) (code string) {
	position := NewAdsPosition()
	o := orm.NewOrm()
	o.QueryTable(position).Filter("is_mobile", isMobile).Filter("Identify", positionIdentify).One(&position)
	if position.Id > 0 {
		var ads []AdsCont
		now := int(time.Now().Unix())
		o.QueryTable(NewAdsCont()).Filter("start__lt", now).Filter("end__gt", now).Filter("pid", position.Id).Filter("status", 1).All(&ads)
		if l := len(ads); l > 0 {
			if l == 1 {
				return ads[0].Code
			} else {
				return ads[time.Now().UnixNano()%int64(l)].Code
			}
		}
	}
	return
}
