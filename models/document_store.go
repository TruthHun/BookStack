package models

import (
	"time"

	"github.com/astaxie/beego"
	"github.com/astaxie/beego/orm"
)

var TableDocumentStore = "md_document_store"

// Document Store，文档存储，将大内容分发到专门的数据表里面
type DocumentStore struct {
	DocumentId int       `orm:"pk;auto;column(document_id)"` //文档id，对应Document中的document_id
	Markdown   string    `orm:"type(text);"`                 //markdown内容
	Content    string    `orm:"type(text);"`                 //文本内容
	UpdatedAt  time.Time `orm:"null"`
}

func NewDocumentStore() *DocumentStore {
	return &DocumentStore{}
}

//插入或者更新
func (this *DocumentStore) InsertOrUpdate(ds DocumentStore, fields ...string) (err error) {
	o := orm.NewOrm()

	var one DocumentStore

	// 全部要修改更新时间，除非用fields 参数指定不修改，即"-updated_at"
	// 这里要多加 1 秒的时间。因为在书籍导入的时候，这个时间跟文档的创建时间是一样的，在内容发布的时候会发布不了。
	ds.UpdatedAt = time.Now().Add(1 * time.Second)

	o.QueryTable(TableDocumentStore).Filter("document_id", ds.DocumentId).One(&one, "document_id")

	if one.DocumentId > 0 {

		if len(fields) > 0 {
			var updateFields []string
			withoutUpdatedAt := false
			for _, field := range fields {
				if field == "-updated_at" || field == "-UpdatedAt" {
					withoutUpdatedAt = true
					continue
				}

				if field == "updated_at" || field == "UpdatedAt" {
					continue
				}
				updateFields = append(updateFields, field)
			}

			fields = updateFields

			if withoutUpdatedAt == false {
				fields = append(fields, "updated_at")
			}
		}

		_, err = o.Update(&ds, fields...)
	} else {
		_, err = o.Insert(&ds)
	}
	return
}

//查询markdown内容或者content内容
func (this *DocumentStore) GetFiledById(docId interface{}, field string) string {
	var ds = DocumentStore{}
	if field != "markdown" {
		field = "content"
	}
	orm.NewOrm().QueryTable(TableDocumentStore).Filter("document_id", docId).One(&ds, field)
	if field == "content" {
		return ds.Content
	}
	return ds.Markdown
}

//查询markdown内容或者content内容
func (this *DocumentStore) DeleteById(docId ...interface{}) {
	if len(docId) > 0 {
		orm.NewOrm().QueryTable(TableDocumentStore).Filter("document_id__in", docId...).Delete()
	}
}

//查询markdown内容或者content内容
func (this *DocumentStore) GetById(docId interface{}) (ds DocumentStore, err error) {
	err = orm.NewOrm().QueryTable(TableDocumentStore).Filter("document_id", docId).One(&ds)
	if err != nil {
		beego.Error(err)
	}
	return
}
