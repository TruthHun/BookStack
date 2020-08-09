package models

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/astaxie/beego/orm"
)

type DocumentSearchResult struct {
	DocumentId   int       `json:"doc_id"`
	BookId       int       `json:"book_id"`
	DocumentName string    `json:"doc_name"`
	Identify     string    `json:"identify"` // Identify 文档唯一标识
	Description  string    `json:"description"`
	Author       string    `json:"author"`
	BookName     string    `json:"book_name"`
	BookIdentify string    `json:"book_identify"`
	ModifyTime   time.Time `json:"modify_time"`
	CreateTime   time.Time `json:"create_time"`
}

// 文档结果
type DocResult struct {
	DocumentId   int       `json:"doc_id"`
	DocumentName string    `json:"doc_name"`
	Identify     string    `json:"identify"` // Identify 文档唯一标识
	Release      string    `json:"release"`  // Release 发布后的Html格式内容.
	Vcnt         int       `json:"vcnt"`     //书籍被浏览次数
	CreateTime   time.Time `json:"create_time"`
	BookId       int       `json:"book_id"`
	BookIdentify string    `json:"book_identify"`
	BookName     string    `json:"book_name"`
}

func NewDocumentSearchResult() *DocumentSearchResult {
	return &DocumentSearchResult{}
}

//分页全局搜索.
func (m *DocumentSearchResult) FindToPager(keyword string, pageIndex, pageSize, memberId int) (searchResult []*DocumentSearchResult, totalCount int, err error) {
	o := orm.NewOrm()

	offset := (pageIndex - 1) * pageSize
	keyword = "%" + keyword + "%"

	if memberId <= 0 {
		sql1 := `SELECT count(doc.document_id) as total_count FROM md_documents AS doc
  LEFT JOIN md_books as book ON doc.book_id = book.book_id
WHERE book.privately_owned = 0 AND (doc.document_name LIKE ? OR doc.release LIKE ?) `

		sql2 := `SELECT doc.document_id,doc.modify_time,doc.create_time,doc.document_name,doc.identify,doc.release as description,doc.modify_time,book.identify as book_identify,book.book_name,rel.member_id,member.account AS author FROM md_documents AS doc
  LEFT JOIN md_books as book ON doc.book_id = book.book_id
  LEFT JOIN md_relationship AS rel ON book.book_id = rel.book_id AND rel.role_id = 0
  LEFT JOIN md_members as member ON rel.member_id = member.member_id
WHERE book.privately_owned = 0 AND (doc.document_name LIKE ? OR doc.release LIKE ?)
 ORDER BY doc.document_id DESC LIMIT ?,? `

		err = o.Raw(sql1, keyword, keyword).QueryRow(&totalCount)
		if err != nil {
			return
		}
		_, err = o.Raw(sql2, keyword, keyword, offset, pageSize).QueryRows(&searchResult)
		if err != nil {
			return
		}
	} else {
		sql1 := `SELECT count(doc.document_id) as total_count FROM md_documents AS doc
  LEFT JOIN md_books as book ON doc.book_id = book.book_id
  LEFT JOIN md_relationship AS rel ON doc.book_id = rel.book_id AND rel.role_id = 0
  LEFT JOIN md_relationship AS rel1 ON doc.book_id = rel1.book_id AND rel1.member_id = ?
WHERE (book.privately_owned = 0 OR rel1.relationship_id > 0)  AND (doc.document_name LIKE ? OR doc.release LIKE ?) `

		sql2 := `SELECT doc.document_id,doc.modify_time,doc.create_time,doc.document_name,doc.identify,doc.release as description,doc.modify_time,book.identify as book_identify,book.book_name,rel.member_id,member.account AS author FROM md_documents AS doc
  LEFT JOIN md_books as book ON doc.book_id = book.book_id
  LEFT JOIN md_relationship AS rel ON book.book_id = rel.book_id AND rel.role_id = 0
  LEFT JOIN md_members as member ON rel.member_id = member.member_id
  LEFT JOIN md_relationship AS rel1 ON doc.book_id = rel1.book_id AND rel1.member_id = ?
WHERE (book.privately_owned = 0 OR rel1.relationship_id > 0)  AND (doc.document_name LIKE ? OR doc.release LIKE ?)
 ORDER BY doc.document_id DESC LIMIT ?,? `

		err = o.Raw(sql1, memberId, keyword, keyword).QueryRow(&totalCount)
		if err != nil {
			return
		}
		_, err = o.Raw(sql2, memberId, keyword, keyword, offset, pageSize).QueryRows(&searchResult)
		if err != nil {
			return
		}
	}
	return
}

//书籍内搜索.
func (m *DocumentSearchResult) SearchDocument(keyword string, bookId int, page, size int) (docs []*DocumentSearchResult, cnt int, err error) {
	o := orm.NewOrm()

	fields := []string{"document_id", "document_name", "identify", "book_id"}
	sql := "SELECT %v FROM md_documents WHERE book_id = " + strconv.Itoa(bookId) + " AND (document_name LIKE ? OR `release` LIKE ?) "
	sqlCount := fmt.Sprintf(sql, "count(document_id) cnt")
	sql = fmt.Sprintf(sql, strings.Join(fields, ",")) + " order by vcnt desc"
	if bookId == 0 {
		// bookId 为 0 的时候，只搜索公开的书籍的文档
		sql = "SELECT %v FROM md_documents d left join md_books b on d.book_id=b.book_id WHERE b.privately_owned=0 and (d.document_name LIKE ? OR d.`release` LIKE ? )"
		sqlCount = fmt.Sprintf(sql, "count(d.document_id) cnt")
		sql = fmt.Sprintf(sql, "d."+strings.Join(fields, ",d.")) + " order by d.vcnt desc"
	}

	keyword = "%" + keyword + "%"

	var count struct {
		Cnt int
	}

	o.Raw(sqlCount, keyword, keyword).QueryRow(&count)
	cnt = count.Cnt

	limit := fmt.Sprintf(" limit %v offset %v", size, (page-1)*size)
	if cnt > 0 {
		_, err = o.Raw(sql+limit, keyword, keyword).QueryRows(&docs)
	}
	return
}

// 根据id查询搜索结果
func (m *DocumentSearchResult) GetDocsById(id []int, withoutCont ...bool) (docs []DocResult, err error) {
	if len(id) == 0 {
		return
	}

	var idArr []string
	for _, i := range id {
		idArr = append(idArr, fmt.Sprint(i))
	}

	fields := []string{
		"d.document_id", "d.document_name", "d.identify", "d.vcnt", "d.create_time", "b.book_id",
	}

	// 不返回内容
	if len(withoutCont) == 0 || !withoutCont[0] {
		fields = append(fields, "b.identify book_identify", "d.release", "b.book_name")
	}

	sqlFmt := "select " + strings.Join(fields, ",") + " from md_documents d left join md_books b on d.book_id=b.book_id where d.document_id in(%v)"
	sql := fmt.Sprintf(sqlFmt, strings.Join(idArr, ","))

	var rows []DocResult
	var cnt int64

	cnt, err = orm.NewOrm().Raw(sql).QueryRows(&rows)
	if cnt > 0 {
		docMap := make(map[int]DocResult)
		for _, row := range rows {
			docMap[row.DocumentId] = row
		}
		client := NewElasticSearchClient()
		for _, i := range id {
			if doc, ok := docMap[i]; ok {
				doc.Release = client.html2Text(doc.Release)
				docs = append(docs, doc)
			}
		}
	}

	return
}
