package models

import (
	"time"

	"github.com/TruthHun/BookStack/conf"
	"github.com/astaxie/beego/orm"
)

type DocumentHistory struct {
	HistoryId    int       `orm:"column(history_id);pk;auto;unique" json:"history_id"`
	Action       string    `orm:"column(action);size(255)" json:"action"`
	ActionName   string    `orm:"column(action_name);size(255)" json:"action_name"`
	DocumentId   int       `orm:"column(document_id);type(int);index" json:"doc_id"`
	DocumentName string    `orm:"column(document_name);size(500)" json:"doc_name"`
	ParentId     int       `orm:"column(parent_id);type(int);index;default(0)" json:"parent_id"`
	MemberId     int       `orm:"column(member_id);type(int);index" json:"member_id"`
	ModifyTime   time.Time `orm:"column(modify_time);type(datetime);auto_now" json:"modify_time"`
	ModifyAt     int       `orm:"column(modify_at);type(int)" json:"-"`
	Version      int64     `orm:"type(bigint);column(version)" json:"version"`
}

type DocumentHistorySimpleResult struct {
	HistoryId  int       `json:"history_id"`
	ActionName string    `json:"action_name"`
	MemberId   int       `json:"member_id"`
	Account    string    `json:"account"`
	Nickname   string    `json:"nickname"`
	ModifyAt   int       `json:"modify_at"`
	ModifyName string    `json:"modify_name"`
	ModifyTime time.Time `json:"modify_time"`
	Version    int64     `json:"version"`
}

// TableName 获取对应数据库表名.
func (m *DocumentHistory) TableName() string {
	return "document_history"
}

// TableEngine 获取数据使用的引擎.
func (m *DocumentHistory) TableEngine() string {
	return "INNODB"
}

func (m *DocumentHistory) TableNameWithPrefix() string {
	return conf.GetDatabasePrefix() + m.TableName()
}

func NewDocumentHistory() *DocumentHistory {
	return &DocumentHistory{}
}
func (m *DocumentHistory) Find(id int) (*DocumentHistory, error) {
	o := orm.NewOrm()
	err := o.QueryTable(m.TableNameWithPrefix()).Filter("history_id", id).One(m)
	return m, err
}

//清空指定文档的历史.
func (m *DocumentHistory) Clear(docId int) error {
	o := orm.NewOrm()
	_, err := o.Raw("DELETE from md_document_history WHERE document_id = ?", docId).Exec()
	if err == nil {
		m.DeleteByDocumentId(docId)
	}
	return err
}

//删除历史.
func (m *DocumentHistory) Delete(historyId, docId int) error {
	o := orm.NewOrm()
	_, err := o.QueryTable(m.TableNameWithPrefix()).Filter("history_id", historyId).Filter("document_id", docId).Delete()
	return err
}

func (m *DocumentHistory) InsertOrUpdate() (history *DocumentHistory, err error) {
	o := orm.NewOrm()
	history = m
	if history.HistoryId > 0 {
		_, err = o.Update(history)
	} else {
		_, err = o.Insert(history)
	}
	return
}

//分页查询指定文档的历史.
func (m *DocumentHistory) FindToPager(docId, pageIndex, pageSize int) (docs []*DocumentHistorySimpleResult, totalCount int, err error) {

	o := orm.NewOrm()

	offset := (pageIndex - 1) * pageSize

	totalCount = 0

	sql := `SELECT history.*,m1.account,m1.nickname,m2.account as modify_name
FROM md_document_history AS history
LEFT JOIN md_members AS m1 ON history.member_id = m1.member_id
LEFT JOIN md_members AS m2 ON history.modify_at = m2.member_id
WHERE history.document_id = ? ORDER BY history.history_id DESC LIMIT ?,?;`

	_, err = o.Raw(sql, docId, offset, pageSize).QueryRows(&docs)

	if err != nil {
		return
	}
	var count int64
	count, err = o.QueryTable(m.TableNameWithPrefix()).Filter("document_id", docId).Count()

	if err != nil {
		return
	}
	totalCount = int(count)

	return
}

//恢复指定历史的文档.
func (history *DocumentHistory) Restore(historyId, docId, uid int) (err error) {
	o := orm.NewOrm()

	o.Begin()
	defer func() {
		if err != nil {
			o.Rollback()
		} else {
			o.Commit()
		}
	}()

	err = o.QueryTable(history.TableNameWithPrefix()).Filter("history_id", historyId).Filter("document_id", docId).One(history)
	if err != nil {
		return err
	}

	var doc *Document

	doc, err = NewDocument().Find(history.DocumentId)
	if err != nil {
		return err
	}

	ds := DocumentStore{DocumentId: docId}
	if err = o.Read(&ds); err != nil {
		return err
	}

	vc := NewVersionControl(docId, history.Version)

	html := vc.GetVersionContent(true)
	md := vc.GetVersionContent(false)

	ds.Markdown = md                        //markdown内容
	ds.Content = html                       //HTML内容
	doc.Release = html                      //HTML内容
	doc.DocumentName = history.DocumentName //文件名
	doc.Version = time.Now().Unix()         //版本

	_, err = o.Update(doc)
	if err != nil {
		return
	}
	_, err = o.Update(&ds)

	return err
}

// 根据文档id删除记录
func (history *DocumentHistory) DeleteByDocumentId(docId int) (err error) {
	var histories []DocumentHistory
	o := orm.NewOrm()
	filter := o.QueryTable(history.TableNameWithPrefix()).Filter("document_id", docId)
	filter.All(&histories)
	for _, item := range histories {
		ver := NewVersionControl(docId, item.Version)
		ver.DeleteVersion() //删除版本文件
	}
	_, err = filter.Delete()
	return
}

// 根据history id 删除记录
func (history *DocumentHistory) DeleteByHistoryId(historyId int) (err error) {
	history, err = history.Find(historyId)
	if history.Version > 0 {
		ver := NewVersionControl(history.DocumentId, history.Version)
		ver.DeleteVersion()
	}
	_, err = orm.NewOrm().QueryTable(history.TableNameWithPrefix()).Filter("history_id", historyId).Delete()
	return
}

// 根据文档id删除
func (history *DocumentHistory) DeleteByLimit(docId, limit int) (err error) {
	if limit <= 0 {
		return
	}

	filter := orm.NewOrm().QueryTable(history.TableNameWithPrefix()).Filter("document_id", docId)

	var cnt int64
	cnt, err = filter.Count()
	if err != nil {
		return
	}

	l := int64(limit)

	if cnt > l {

		var histories []DocumentHistory
		var historyIds []interface{}

		filter2 := filter.OrderBy("history_id").Limit(cnt - l)
		filter2.All(&histories, "document_id", "version", "history_id")

		for _, item := range histories {
			ver := NewVersionControl(item.DocumentId, item.Version)
			ver.DeleteVersion()
			historyIds = append(historyIds, item.HistoryId)
		}
		filter.Filter("history_id__in", historyIds).Delete()
	}
	return
}
