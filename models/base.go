package models

import (
	"fmt"

	"os"

	"strconv"
	"strings"

	"time"

	"net/url"

	"github.com/TruthHun/gotil/sitemap"
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/orm"
)

//设置增减
//@param            table           需要处理的数据表
//@param            field           字段
//@param            condition       条件
//@param            incre           是否是增长值，true则增加，false则减少
//@param            step            增或减的步长
func SetIncreAndDecre(table string, field string, condition string, incre bool, step ...int) (err error) {
	mark := "-"
	if incre {
		mark = "+"
	}
	s := 1
	if len(step) > 0 {
		s = step[0]
	}
	sql := fmt.Sprintf("update %v set %v=%v%v%v where %v", table, field, field, mark, s, condition)
	_, err = orm.NewOrm().Raw(sql).Exec()
	return
}

type SitemapDocs struct {
	DocumentId   int
	DocumentName string
	Identify     string
	BookId       int
}

//站点地图数据
func SitemapData(page, listRows int) (totalRows int64, sitemaps []SitemapDocs) {
	//获取公开的项目
	var (
		books   []Book
		docs    []Document
		maps    = make(map[int]string)
		booksId []interface{}
	)

	o := orm.NewOrm()
	o.QueryTable("md_books").Filter("privately_owned", 0).Limit(100000).All(&books, "book_id", "identify")
	if len(books) > 0 {
		for _, book := range books {
			booksId = append(booksId, book.BookId)
			maps[book.BookId] = book.Identify
		}
		q := o.QueryTable("md_documents").Filter("BookId__in", booksId...)
		totalRows, _ = q.Count()
		q.Limit(listRows).Offset((page-1)*listRows).All(&docs, "document_id", "document_name", "book_id")
		if len(docs) > 0 {
			for _, doc := range docs {
				sd := SitemapDocs{
					DocumentId:   doc.DocumentId,
					DocumentName: doc.DocumentName,
					BookId:       doc.BookId,
				}
				if v, ok := maps[doc.BookId]; ok {
					sd.Identify = v
				}
				sitemaps = append(sitemaps, sd)
			}
		}
	}
	return
}

func SitemapUpdate(domain string) {
	var (
		files   []string
		bookIds []interface{}
		bookMap = make(map[int]string)
		Sitemap = sitemap.NewSitemap("1.0", "utf-8")
		o       = orm.NewOrm()
		si      []sitemap.SitemapIndex
	)
	domain = strings.TrimSuffix(domain, "/")
	os.Mkdir("sitemap", os.ModePerm)
	//查询公开的项目
	qsBooks := o.QueryTable("md_books").Filter("privately_owned", 0)
	limit := 10000
	for i := 0; i < 10; i++ {
		var books []Book
		qsBooks.Limit(limit).Offset(i*limit).All(&books, "book_id", "identify", "release_time", "book_name")
		if len(books) > 0 {
			file := "sitemap/books-" + strconv.Itoa(i) + ".xml"
			files = append(files, file)
			var su []sitemap.SitemapUrl
			for _, book := range books {
				su = append(su, sitemap.SitemapUrl{
					Loc:        domain + beego.URLFor("DocumentController.Index", ":key", book.Identify),
					Lastmod:    book.ReleaseTime.Format("2006-01-02 15:04:05"),
					ChangeFreq: sitemap.WEEKLY,
					Priority:   0.9,
				})
				bookIds = append(bookIds, book.BookId)
				bookMap[book.BookId] = book.Identify
			}
			Sitemap.CreateSitemapContent(su, file)
		} else {
			i = 10
		}
	}
	qsDocs := o.QueryTable("md_documents").Filter("book_id__in", bookIds...)
	for i := 0; i < 100; i++ {
		var docs []Document
		qsDocs.Limit(limit).Offset(i*limit).All(&docs, "modify_time", "book_id", "document_name", "document_id", "identify")
		if len(docs) > 0 {
			file := "sitemap/docs-" + strconv.Itoa(i) + ".xml"
			files = append(files, file)
			var su []sitemap.SitemapUrl
			for _, doc := range docs {
				bookIdentify := ""
				if idtf, ok := bookMap[doc.BookId]; ok {
					bookIdentify = idtf
				}
				su = append(su, sitemap.SitemapUrl{
					Loc:        domain + beego.URLFor("DocumentController.Read", ":key", bookIdentify, ":id", url.QueryEscape(doc.Identify)),
					Lastmod:    doc.ModifyTime.Format("2006-01-02 15:04:05"),
					ChangeFreq: sitemap.WEEKLY,
					Priority:   0.9,
				})
			}
			Sitemap.CreateSitemapContent(su, file)
		} else {
			i = 100
		}
	}
	if len(files) > 0 {
		for _, f := range files {
			si = append(si, sitemap.SitemapIndex{
				Loc:     domain + "/" + f,
				Lastmod: time.Now().Format("2006-01-02 15:04:05"),
			})
		}

	}
	Sitemap.CreateSitemapIndex(si, "sitemap.xml")
}
