package models

import (
	"strings"

	"github.com/TruthHun/BookStack/conf"
	"github.com/astaxie/beego/orm"
)

type Label struct {
	LabelId    int    `orm:"column(label_id);pk;auto;unique;" json:"label_id"`
	LabelName  string `orm:"column(label_name);size(50);unique" json:"label_name"`
	BookNumber int    `orm:"column(book_number)" json:"book_number"`
}

// TableName 获取对应数据库表名.
func (m *Label) TableName() string {
	return "label"
}

// TableEngine 获取数据使用的引擎.
func (m *Label) TableEngine() string {
	return "INNODB"
}

func (m *Label) TableNameWithPrefix() string {
	return conf.GetDatabasePrefix() + m.TableName()
}

func NewLabel() *Label {
	return &Label{}
}

func (m *Label) FindFirst(field string, value interface{}) (*Label, error) {
	o := orm.NewOrm()

	err := o.QueryTable(m.TableNameWithPrefix()).Filter(field, value).One(m)

	return m, err
}

//插入或更新标签.
func (m *Label) InsertOrUpdate(labelName string) (err error) {
	o := orm.NewOrm()
	count, _ := o.QueryTable(NewBook().TableNameWithPrefix()).Filter("label", labelName).Count()
	m.BookNumber = int(count) + 1
	m.LabelName = labelName
	if count == 0 {
		_, err = o.Insert(m)
	} else {
		_, err = o.Update(m)
	}
	return
}

//批量插入或更新标签.
func (m *Label) InsertOrUpdateMulti(labels string) {
	if labels != "" {
		labelArray := strings.Split(labels, ",")
		for _, label := range labelArray {
			if label != "" {
				NewLabel().InsertOrUpdate(strings.TrimSpace(label))
			}
		}
	}
}

//分页查找标签.
func (m *Label) FindToPager(pageIndex, pageSize int, word ...string) (labels []*Label, totalCount int, err error) {
	var count int64
	o := orm.NewOrm()
	q := o.QueryTable(m.TableNameWithPrefix()).OrderBy("-book_number")
	if len(word) > 0 {
		q = q.Filter("label_name__icontains", word[0])
	}
	count, err = q.Count()
	if err != nil {
		return
	}
	totalCount = int(count)

	offset := (pageIndex - 1) * pageSize

	_, err = q.Offset(offset).Limit(pageSize).All(&labels)

	return
}
