package models

import (
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/astaxie/beego"
	"github.com/astaxie/beego/orm"
	"github.com/hashicorp/go-version"
)

// 版本
type Version struct {
	Id         int           `json:"id"`
	Title      string        `json:"title" orm:"unique;size(128)"`                 // 版本名称
	OrderIndex int           `json:"order_index,omitempty" orm:"default(0);index"` // 排序，值越大越靠前
	Versions   []VersionItem `json:"versions" orm:"-"`
	CreatedAt  time.Time     `json:"created_at" orm:"auto_now_add;type(datetime)"`
	UpdatedAt  time.Time     `json:"updated_at" orm:"auto_now;type(datetime)"`
}

type VersionItem struct {
	Id           int    `json:"id"`
	VersionId    int    `json:"version_id" orm:"index"`
	BookName     string `json:"book_name`
	BookIdentify string `json:"book_identify" orm:"unique;size(128)"`
	VersionNO    string `json:"version_no,omitempty" orm:"column(version_no);size(64)"`
}

// NewVersion 版本
func NewVersion() *Version {
	return &Version{}
}

// NewVersionItem 版本列表
func NewVersionItem() *VersionItem {
	return &VersionItem{}
}

// InsertOrUpdate 新增或更新版本标识
func (m *Version) InsertOrUpdate() (err error) {
	exist := m.FindByTitle(m.Title)
	o := orm.NewOrm()
	if exist.Id > 0 { // 版本名称已存在
		if m.Id == 0 || (m.Id > 0 && exist.Id != m.Id) {
			err = errors.New("版本名称已存在")
			return
		}
		_, err = o.Update(m)
	} else {
		_, err = o.Insert(m)
	}
	return
}

func (m *Version) FindByTitle(title string) (version Version) {
	orm.NewOrm().QueryTable(m).Filter("title", title).One(&version)
	return
}

// 删除版本
func (m *Version) Delete(id int) (err error) {
	o := orm.NewOrm()
	o.Begin()
	defer func() {
		if err != nil {
			o.Rollback()
		} else {
			o.Commit()
		}
	}()

	if _, err = o.QueryTable(m).Filter("id", id).Delete(); err != nil {
		beego.Error(err)
		return
	}

	if _, err = o.QueryTable(NewVersionItem()).Filter("version_id", id).Delete(); err != nil {
		beego.Error(err)
	}
	return
}

// GetVersionItems 查询所有公共版本
func (m *Version) GetPublicVersionItems(bookIdentify string) (items []VersionItem) {
	item := VersionItem{}
	o := orm.NewOrm()
	o.QueryTable(item).Filter("book_identify", bookIdentify).One(&item)
	if item.Id == 0 {
		return
	}
	sql := "select vi.*,b.book_name from md_version_item vi left join md_books b on vi.book_identify = b.identify where vi.version_id = ? and privately_owned = 0"
	o.Raw(sql, item.VersionId).QueryRows(&items)

	var available []VersionItem
	for _, item := range items {
		if item.VersionNO != "" {
			available = append(available, item)
		}
	}

	if len(available) == 0 {
		return
	}

	items = m.SortVersionItems(available)
	return
}

func (m *Version) GetVersionItems(versionId int) (items []VersionItem) {
	orm.NewOrm().QueryTable(NewVersionItem()).Filter("version_id", versionId).All(&items)
	return
}

// InsertOrUpdateVersionItem 添加版本项
func (m *Version) InsertOrUpdateVersionItem(versionId int, bookIdentify, bookName string, versionNO string) (err error) {
	o := orm.NewOrm()
	if versionId <= 0 {
		if bookIdentify != "" {
			o.QueryTable(NewVersionItem()).Filter("book_identify", bookIdentify).Delete()
		}
		return
	}

	if versionId > 0 && versionNO == "" {
		return
	}

	versionNO = strings.TrimLeft(strings.ToLower(strings.TrimSpace(versionNO)), "v")

	item := &VersionItem{
		VersionId:    versionId,
		BookIdentify: bookIdentify,
		BookName:     bookName,
		VersionNO:    versionNO,
	}
	o.Begin()
	defer func() {
		if err != nil {
			o.Rollback()
		} else {
			o.Commit()
		}
	}()

	if _, err = o.QueryTable(item).Filter("book_identify", bookIdentify).Delete(); err != nil {
		beego.Error(err)
		return
	}

	if _, err = o.Insert(item); err != nil {
		beego.Error(err)
	}
	return
}

func (m *Version) All() (versions []Version) {
	orm.NewOrm().QueryTable(m).Limit(100000).OrderBy("title").All(&versions, "id", "title")
	return
}

func (m *Version) GetVersionItem(bookIdentify string) (versionItem VersionItem) {
	orm.NewOrm().QueryTable(NewVersionItem()).Filter("book_identify", bookIdentify).One(&versionItem)
	return
}

func (m *Version) Lists(page, size int, wd string) (versions []Version, total int64) {
	o := orm.NewOrm()
	q := o.QueryTable(m)
	wd = strings.TrimSpace(wd)
	if wd != "" {
		q = q.Filter("title__icontains", wd)
	}

	total, _ = q.Count()
	if total == 0 {
		return
	}

	q.Limit(size).Offset((page - 1) * size).OrderBy("-id").All(&versions)

	l := len(versions)
	if l == 0 {
		return
	}

	var (
		ids     []interface{}
		items   []VersionItem
		itemMap = make(map[int][]VersionItem)
	)

	for _, ver := range versions {
		ids = append(ids, ver.Id)
	}

	o.QueryTable(NewVersionItem()).Filter("version_id__in", ids...).Limit(l * 1000).OrderBy("version_id").All(&items)
	for _, item := range items {
		itemMap[item.VersionId] = append(itemMap[item.VersionId], item)
	}

	for idx, ver := range versions {
		if v, ok := itemMap[ver.Id]; ok {
			ver.Versions = m.SortVersionItems(v)
			versions[idx] = ver
		}
	}

	return
}

// DeleteVersionItem 删除版本项
func (m *Version) DeleteVersionItem(id int) (err error) {
	_, err = orm.NewOrm().QueryTable(NewVersionItem()).Filter("id", id).Delete()
	if err != nil {
		beego.Error(err)
	}
	return
}

// SortVersionItems 版本排序
func (m *Version) SortVersionItems(versionItems []VersionItem) (vers []VersionItem) {
	if len(versionItems) == 0 {
		return
	}

	var versions []*version.Version
	itemMap := make(map[string][]VersionItem)
	for _, item := range versionItems {
		v, err := version.NewVersion(item.VersionNO)
		if err != nil {
			beego.Error(err)
			continue
		}
		if _, ok := itemMap[item.VersionNO]; !ok {
			versions = append(versions, v)
		}
		itemMap[item.VersionNO] = append(itemMap[item.VersionNO], item)
	}

	sort.Sort(
		sort.Reverse(
			version.Collection(versions),
		),
	)

	for _, v := range versions {
		if vv, ok := itemMap[v.Original()]; ok {
			vers = append(vers, vv...)
		}
	}

	return vers
}
