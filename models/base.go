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

var httpCache = 0

func Init() {
	initAPI()
	go func() {
		httpCache, _ = strconv.Atoi(GetOptionValue("HTTP_CACHE", "0"))
		time.Sleep(60 * time.Second)
	}()
}

func GetHTTPCache() int {
	return httpCache
}

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

// 统计书籍分类
var counting = false

type Count struct {
	Cnt        int
	CategoryId int
}

func CountCategory() {
	if counting {
		return
	}
	counting = true
	defer func() {
		counting = false
	}()

	var count []Count

	o := orm.NewOrm()
	sql := "select count(bc.id) cnt, bc.category_id from md_book_category bc left join md_books b on b.book_id=bc.book_id where b.privately_owned=0 group by bc.category_id"
	o.Raw(sql).QueryRows(&count)
	if len(count) == 0 {
		return
	}

	var cates []Category
	tableCate := "md_category"
	o.QueryTable(tableCate).All(&cates, "id", "pid", "cnt")
	if len(cates) == 0 {
		return
	}

	var err error

	o.Begin()
	defer func() {
		if err != nil {
			o.Rollback()
		} else {
			o.Commit()
		}
	}()

	o.QueryTable(tableCate).Update(orm.Params{"cnt": 0})
	cateChild := make(map[int]int)
	for _, item := range count {
		if item.Cnt > 0 {
			cateChild[item.CategoryId] = item.Cnt
			_, err = o.QueryTable(tableCate).Filter("id", item.CategoryId).Update(orm.Params{"cnt": item.Cnt})
			if err != nil {
				return
			}
		}
	}
}
