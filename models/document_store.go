package models

import "github.com/astaxie/beego/orm"

var TableDocumentStore = "md_document_store"

// Document Store，文档存储，将大内容分发到专门的数据表里面
type DocumentStore struct {
	DocumentId int    `orm:"pk;auto;column(document_id)"` //文档id，对应Document中的document_id
	Markdown   string `orm:"type(text);"`                 //markdown内容
	Content    string `orm:"type(text);"`                 //文本内容
}

//插入或者更新
func (this *DocumentStore) InsertOrUpdate(ds DocumentStore, fields ...string) (err error) {
	o := orm.NewOrm()
	var one DocumentStore
	o.QueryTable(TableDocumentStore).Filter("document_id", ds.DocumentId).One(&one, "document_id")

	if one.DocumentId > 0 {
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
