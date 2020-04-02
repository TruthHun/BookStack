package models

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/astaxie/beego"
	"github.com/astaxie/beego/orm"
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
	AdsPositionBeforeFriendLink        = "global-before-friend-link"
	AdsPositionGlobalFooter            = "global-footer"
	AdsPositionUnderLatestRecommend    = "index-under-latest-recommend"
	AdsPositionSearchRight             = "search-right"
	AdsPositionSearchTop               = "search-top"
	AdsPositionSearchBottom            = "search-bottom"
	AdsPositionUnderBookName           = "intro-under-book-name"
	AdsPositionBeforeMenu              = "intro-before-menu"
	AdsPositionBeforeRelatedBooks      = "intro-before-related-books"
	AdsPositionUnderExploreNav         = "explore-under-nav"
	AdsPositionBeforeExplorePagination = "explore-before-pagination"
	AdsPositionUnderExplorePagination  = "explore-under-pagination"
	AdsPositionContentTop              = "content-top"
	AdsPositionContentBottom           = "content-bottom"
)

var (
	adsCache      sync.Map // map[pid][]AdsCont
	positionCache sync.Map // map[positionIdentify-isMobile]=pid
)

func InstallAdsPosition() {
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
			Title:    "[书籍介绍页]书籍名称下方",
			Identify: AdsPositionUnderBookName,
		},
		{
			IsMobile: false,
			Title:    "[书籍介绍页]文档概述上方",
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
			Title:    "[发现页]导航栏下方",
			Identify: AdsPositionUnderExploreNav,
		},
		{
			IsMobile: false,
			Title:    "[发现页]分页上方",
			Identify: AdsPositionBeforeExplorePagination,
		},
		{
			IsMobile: false,
			Title:    "[发现页]分页下方",
			Identify: AdsPositionUnderExplorePagination,
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
			Title:    "[书籍介绍页]书籍名称下方",
			Identify: AdsPositionUnderBookName,
		},
		{
			IsMobile: true,
			Title:    "[书籍介绍页]文档概述上方",
			Identify: AdsPositionBeforeMenu,
		},
		{
			IsMobile: true,
			Title:    "[书籍介绍页]相关书籍上方",
			Identify: AdsPositionBeforeRelatedBooks,
		},
		{
			IsMobile: true,
			Title:    "[发现页]导航栏下方",
			Identify: AdsPositionUnderExploreNav,
		},
		{
			IsMobile: true,
			Title:    "[发现页]分页上方",
			Identify: AdsPositionBeforeExplorePagination,
		},
		{
			IsMobile: true,
			Title:    "[发现页]分页下方",
			Identify: AdsPositionUnderExplorePagination,
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

func initAdsCache() {
	o := orm.NewOrm()
	var pos []AdsPosition
	o.QueryTable(&AdsPosition{}).All(&pos)
	for _, item := range pos {
		key := fmt.Sprintf("%v-%v", item.Identify, item.IsMobile)
		positionCache.Store(key, item.Id)
	}
	UpdateAdsCache()
}

func UpdateAdsCache() {
	var (
		ads   []AdsCont
		cache sync.Map
		data  = make(map[int][]AdsCont)
	)
	now := time.Now().Unix()
	orm.NewOrm().QueryTable(&AdsCont{}).Filter("status", 1).Filter("start__lt", now).Filter("end__gt", now).All(&ads)
	for _, item := range ads {
		data[item.Pid] = append(data[item.Pid], item)
	}
	if beego.AppConfig.String("runmode") == "dev" {
		beego.Info(" =============== update ads cache =============== ")
	}
	for pid, arr := range data {
		cache.Store(pid, arr)
	}
	adsCache = cache
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

func GetAdsCode(positionIdentify string, isMobile bool) (code string) {
	if beego.AppConfig.String("runmode") == "dev" {
		beego.Debug("getAdsCode", positionIdentify, isMobile)
	}
	key := fmt.Sprintf("%v-%v", positionIdentify, isMobile)
	pid, ok := positionCache.Load(key)
	if !ok {
		return
	}
	data, ok := adsCache.Load(pid.(int))
	if !ok {
		return
	}
	var ads []AdsCont
	nowSec := int(time.Now().Unix())
	for _, ad := range data.([]AdsCont) {
		if ad.Start <= nowSec && ad.End >= nowSec {
			ads = append(ads, ad)
		}
	}
	lenAds := len(ads)

	if lenAds == 0 {
		return
	} else if lenAds == 1 {
		return ads[0].Code
	}
	rand.Seed(time.Now().UnixNano())
	return ads[rand.Intn(lenAds)].Code
}
