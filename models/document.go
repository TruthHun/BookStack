package models

import (
	"os"
	"strings"
	"time"

	"bytes"
	"fmt"

	"io/ioutil"

	"path/filepath"

	"crypto/tls"

	"strconv"

	"github.com/PuerkitoBio/goquery"
	"github.com/TruthHun/BookStack/conf"
	"github.com/TruthHun/BookStack/utils"
	"github.com/TruthHun/converter/converter"
	"github.com/TruthHun/gotil/cryptil"
	"github.com/TruthHun/gotil/util"
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/httplib"
	"github.com/astaxie/beego/orm"
	"github.com/kataras/iris/core/errors"
)

// Document struct.
type Document struct {
	DocumentId   int           `orm:"pk;auto;column(document_id)" json:"doc_id"`
	DocumentName string        `orm:"column(document_name);size(500)" json:"doc_name"`
	Identify     string        `orm:"column(identify);size(100);index;null;default(null)" json:"identify"` // Identify 文档唯一标识
	BookId       int           `orm:"column(book_id);type(int);index" json:"book_id"`
	ParentId     int           `orm:"column(parent_id);type(int);index;default(0)" json:"parent_id"`
	OrderSort    int           `orm:"column(order_sort);default(0);type(int);index" json:"order_sort"`
	Release      string        `orm:"column(release);type(text);null" json:"release"` // Release 发布后的Html格式内容.
	CreateTime   time.Time     `orm:"column(create_time);type(datetime);auto_now_add" json:"create_time"`
	MemberId     int           `orm:"column(member_id);type(int)" json:"member_id"`
	ModifyTime   time.Time     `orm:"column(modify_time);type(datetime);default(null);auto_now" json:"modify_time"`
	ModifyAt     int           `orm:"column(modify_at);type(int)" json:"-"`
	Version      int64         `orm:"type(bigint);column(version)" json:"version"`
	AttachList   []*Attachment `orm:"-" json:"attach"`
	Vcnt         int           `orm:"column(vcnt);default(0)" json:"vcnt"` //文档项目被浏览次数
	Markdown     string        `orm:"-" json:"markdown"`
}

// 多字段唯一键
func (m *Document) TableUnique() [][]string {
	return [][]string{
		[]string{"BookId", "Identify"},
	}
}

// TableName 获取对应数据库表名.
func (m *Document) TableName() string {
	return "documents"
}

// TableEngine 获取数据使用的引擎.
func (m *Document) TableEngine() string {
	return "INNODB"
}

func (m *Document) TableNameWithPrefix() string {
	return conf.GetDatabasePrefix() + m.TableName()
}

func NewDocument() *Document {
	return &Document{
		Version: time.Now().Unix(),
	}
}

//根据文档ID查询指定文档.
func (m *Document) Find(id int) (*Document, error) {
	if id <= 0 {
		return m, ErrInvalidParameter
	}
	o := orm.NewOrm()

	err := o.QueryTable(m.TableNameWithPrefix()).Filter("document_id", id).One(m)

	if err == orm.ErrNoRows {
		return m, ErrDataNotExist
	}
	return m, nil
}

//插入和更新文档.
//存在文档id或者文档标识，则表示更新文档内容
func (m *Document) InsertOrUpdate(cols ...string) (id int64, err error) {
	o := orm.NewOrm()
	id = int64(m.DocumentId)
	m.ModifyTime = time.Now()
	if m.DocumentId > 0 { //文档id存在，则更新
		_, err = o.Update(m, cols...)
	} else {
		var mm Document
		//直接查询一个字段，优化MySQL IO
		o.QueryTable("md_documents").Filter("identify", m.Identify).Filter("book_id", m.BookId).One(&mm, "document_id")
		if mm.DocumentId == 0 {
			m.CreateTime = time.Now()
			id, err = o.Insert(m)
			NewBook().ResetDocumentNumber(m.BookId)
		} else { //identify存在，则执行更新
			_, err = o.Update(m)
			id = int64(mm.DocumentId)
		}
	}
	return
}

//根据指定字段查询一条文档.
func (m *Document) FindByFieldFirst(field string, v interface{}) (*Document, error) {
	o := orm.NewOrm()

	err := o.QueryTable(m.TableNameWithPrefix()).Filter(field, v).One(m)

	return m, err
}

//根据指定字段查询一条文档.
func (m *Document) FindByBookIdAndDocIdentify(BookId, Identify interface{}) (*Document, error) {
	err := orm.NewOrm().QueryTable(m.TableNameWithPrefix()).Filter("BookId", BookId).Filter("Identify", Identify).One(m)
	return m, err
}

//递归删除一个文档.
func (m *Document) RecursiveDocument(doc_id int) error {

	o := orm.NewOrm()
	modelStore := new(DocumentStore)

	if doc, err := m.Find(doc_id); err == nil {
		o.Delete(doc)
		modelStore.DeleteById(doc_id)
		NewDocumentHistory().Clear(doc_id)
	}

	var docs []*Document

	_, err := o.QueryTable(m.TableNameWithPrefix()).Filter("parent_id", doc_id).All(&docs)

	if err != nil {
		beego.Error("RecursiveDocument => ", err)
		return err
	}

	for _, item := range docs {
		doc_id := item.DocumentId
		o.QueryTable(m.TableNameWithPrefix()).Filter("document_id", doc_id).Delete()
		//删除document_store表的文档
		modelStore.DeleteById(doc_id)
		m.RecursiveDocument(doc_id)
	}

	return nil
}

//发布文档
func (m *Document) ReleaseContent(book_id int, base_url string) {
	utils.BooksRelease.Set(book_id)
	defer utils.BooksRelease.Delete(book_id)

	//发布的时间戳
	releaseTime := time.Now()

	o := orm.NewOrm()
	var (
		docs       []*Document
		book       Book
		tableBooks = "md_books"
		releaseNum = 0 //发布的文档数量，用于生成PDF、epub、mobi文档
	)
	qs := o.QueryTable(tableBooks).Filter("book_id", book_id)
	qs.One(&book)
	//查询更新时间大于项目发布时间的文档
	//_, err := o.QueryTable(m.TableNameWithPrefix()).Filter("book_id", book_id).Filter("modify_time__gt", book.ReleaseTime).All(&docs, "document_id")
	//全部重新发布
	_, err := o.QueryTable(m.TableNameWithPrefix()).Filter("book_id", book_id).All(&docs, "document_id")
	if err != nil {
		beego.Error("发布失败 => ", err)
		return
	}
	idx := 1
	ModelStore := new(DocumentStore)
	for _, item := range docs {
		content := strings.TrimSpace(ModelStore.GetFiledById(item.DocumentId, "content"))
		if len(utils.GetTextFromHtml(content)) == 0 {
			//达到5个协程，休息3秒
			//if idx%5 == 0 {
			//	time.Sleep(3 * time.Second)
			//}
			//采用单线程去发布，避免用户多操作，避免Chrome启动过多导致内存、CPU等资源耗费致使服务器宕机
			utils.RenderDocumentById(item.DocumentId)
			idx++
		} else {
			item.Release = content
			attach_list, err := NewAttachment().FindListByDocumentId(item.DocumentId)
			if err == nil && len(attach_list) > 0 {
				content := bytes.NewBufferString("<div class=\"attach-list\"><strong>附件</strong><ul>")
				for _, attach := range attach_list {
					li := fmt.Sprintf("<li><a href=\"%s\" target=\"_blank\" title=\"%s\">%s</a></li>", attach.HttpPath, attach.FileName, attach.FileName)
					content.WriteString(li)
				}
				content.WriteString("</ul></div>")
				item.Release += content.String()
			}
			_, err = o.Update(item, "release")
			if err != nil {
				beego.Error(fmt.Sprintf("发布失败 => %+v", item), err)
			} else {
				releaseNum++
			}
		}

	}

	//最后再更新时间戳
	if _, err = qs.Update(orm.Params{
		"release_time": releaseTime,
	}); err != nil {
		beego.Error(err.Error())
	}
}

//离线文档生成
func (m *Document) GenerateBook(book *Book, base_url string) {
	//将书籍id加入进去，表示正在生成离线文档
	utils.BooksGenerate.Set(book.BookId)
	defer utils.BooksGenerate.Delete(book.BookId) //最后移除

	//公开文档，才生成文档文件
	debug := true
	if beego.AppConfig.String("runmode") == "prod" {
		debug = false
	}
	Nickname := new(Member).GetNicknameByUid(book.MemberId)

	docs, err := NewDocument().FindListByBookId(book.BookId)

	if err != nil {
		beego.Error(err)
		return
	}
	var ExpCfg = converter.Config{
		Contributor: beego.AppConfig.String("exportCreator"),
		Cover:       "",
		Creator:     beego.AppConfig.String("exportCreator"),
		Timestamp:   book.ReleaseTime.Format("2006-01-02"),
		Description: book.Description,
		Header:      beego.AppConfig.String("exportHeader"),
		Footer:      beego.AppConfig.String("exportFooter"),
		Identifier:  "",
		Language:    "zh-CN",
		Publisher:   beego.AppConfig.String("exportCreator"),
		Title:       book.BookName,
		Format:      []string{"epub", "mobi", "pdf"},
		FontSize:    beego.AppConfig.String("exportFontSize"),
		PaperSize:   beego.AppConfig.String("exportPaperSize"),
		More: []string{
			"--pdf-page-margin-bottom", beego.AppConfig.DefaultString("exportMarginBottom", "72"),
			"--pdf-page-margin-left", beego.AppConfig.DefaultString("exportMarginLeft", "72"),
			"--pdf-page-margin-right", beego.AppConfig.DefaultString("exportMarginRight", "72"),
			"--pdf-page-margin-top", beego.AppConfig.DefaultString("exportMarginTop", "72"),
		},
	}
	folder := fmt.Sprintf("cache/books/%v/", book.Identify)
	os.MkdirAll(folder, os.ModePerm)
	if !debug {
		defer os.RemoveAll(folder)
	}

	//生成致谢信内容
	if htmlstr, err := utils.ExecuteViewPathTemplate("document/tpl_statement.html", map[string]interface{}{"Model": book, "Nickname": Nickname, "Date": ExpCfg.Timestamp}); err == nil {
		h1Title := "说明"
		if doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlstr)); err == nil {
			h1Title = doc.Find("h1").Text()
		}
		toc := converter.Toc{
			Id:    time.Now().Nanosecond(),
			Pid:   0,
			Title: h1Title,
			Link:  "statement.html",
		}
		htmlname := folder + toc.Link
		ioutil.WriteFile(htmlname, []byte(htmlstr), os.ModePerm)
		ExpCfg.Toc = append(ExpCfg.Toc, toc)
	}
	ModelStore := new(DocumentStore)
	for _, doc := range docs {
		content := strings.TrimSpace(ModelStore.GetFiledById(doc.DocumentId, "content"))
		if utils.GetTextFromHtml(content) == "" { //内容为空，渲染文档内容，并再重新获取文档内容
			utils.RenderDocumentById(doc.DocumentId)
			orm.NewOrm().Read(doc, "document_id")
		}

		//将图片链接更换成绝对链接
		toc := converter.Toc{
			Id:    doc.DocumentId,
			Pid:   doc.ParentId,
			Title: doc.DocumentName,
			Link:  fmt.Sprintf("%v.html", doc.DocumentId),
		}
		ExpCfg.Toc = append(ExpCfg.Toc, toc)
		//图片处理，如果图片路径不是http开头，则表示是相对路径的图片，加上BaseUrl.如果图片是以http开头的，下载下来
		if gq, err := goquery.NewDocumentFromReader(strings.NewReader(doc.Release)); err == nil {
			gq.Find("img").Each(func(i int, s *goquery.Selection) {
				pic := ""
				if src, ok := s.Attr("src"); ok {
					if srcLower := strings.ToLower(src); strings.HasPrefix(srcLower, "http://") || strings.HasPrefix(srcLower, "https://") {
						pic = src
					} else {
						pic = base_url + src
					}
					//下载图片，放到folder目录下
					ext := ""
					if picslice := strings.Split(pic, "?"); len(picslice) > 0 {
						ext = filepath.Ext(picslice[0])
					}
					filename := cryptil.Md5Crypt(pic) + ext
					localpic := folder + filename
					req := httplib.Get(pic).SetTimeout(5*time.Second, 5*time.Second)
					if strings.HasPrefix(pic, "https") {
						req.SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true})
					}
					req.Header("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/65.0.3298.4 Safari/537.36")
					if err := req.ToFile(localpic); err == nil { //成功下载图片
						s.SetAttr("src", filename)
					} else {
						beego.Error("错误:", err, filename, pic)
						s.SetAttr("src", pic)
					}

				}
			})
			gq.Find(".markdown-toc").Remove()
			doc.Release, _ = gq.Find("body").Html()
		}

		//生成html
		if htmlstr, err := utils.ExecuteViewPathTemplate("document/tpl_export.html", map[string]interface{}{"Model": book, "Doc": doc, "BaseUrl": base_url, "Nickname": Nickname, "Date": ExpCfg.Timestamp}); err == nil {
			htmlname := folder + toc.Link
			ioutil.WriteFile(htmlname, []byte(htmlstr), os.ModePerm)
		} else {
			beego.Error(err.Error())
		}

	}

	//复制css文件到目录
	if b, err := ioutil.ReadFile("static/editor.md/css/export-editormd.css"); err == nil {
		ioutil.WriteFile(folder+"editormd.css", b, os.ModePerm)
	} else {
		beego.Error("css样式不存在", err)
	}
	cfgfile := folder + "config.json"
	ioutil.WriteFile(cfgfile, []byte(util.InterfaceToJson(ExpCfg)), os.ModePerm)
	if Convert, err := converter.NewConverter(cfgfile, debug); err == nil {
		if err := Convert.Convert(); err != nil {
			beego.Error(err.Error())
		}
	} else {
		beego.Error(err.Error())
	}

	//将文档移动到oss
	//将PDF文档移动到oss
	oldBook := fmt.Sprintf("projects/%v/books/%v", book.Identify, book.GenerateTime.Unix()) //旧书籍的生成时间
	//最后再更新文档生成时间
	book.GenerateTime = time.Now()
	if _, err = orm.NewOrm().Update(book, "generate_time"); err != nil {
		beego.Error(err.Error())
	}
	orm.NewOrm().Read(book)
	newBook := fmt.Sprintf("projects/%v/books/%v", book.Identify, book.GenerateTime.Unix())

	exts := []string{".pdf", ".epub", ".mobi"}
	for _, ext := range exts {
		switch utils.StoreType {
		case utils.StoreOss:
			//不要开启gzip压缩，否则会出现文件损坏的情况
			if err := ModelStoreOss.MoveToOss(folder+"output/book"+ext, newBook+ext, true, false); err != nil {
				beego.Error(err)
			} else { //设置下载头
				ModelStoreOss.SetObjectMeta(newBook+ext, book.BookName+ext)
			}
		case utils.StoreLocal: //本地存储
			ModelStoreLocal.MoveToStore(folder+"output/book"+ext, "uploads/"+newBook+ext)
		}

	}

	//删除旧文件
	switch utils.StoreType {
	case utils.StoreOss:
		if err := ModelStoreOss.DelFromOss(oldBook+".pdf", oldBook+".epub", oldBook+".mobi"); err != nil { //删除旧版
			beego.Error(err)
		}
	case utils.StoreLocal: //本地存储
		if err := ModelStoreLocal.DelFiles(oldBook+".pdf", oldBook+".epub", oldBook+".mobi"); err != nil { //删除旧版
			beego.Error(err)
		}
	}

	//if _, err = orm.NewOrm().QueryTable("md_books").Filter("book_id", book.BookId).Update(orm.Params{"generate_time": NewGenerateTime}); err != nil {
	//	beego.Error(err.Error())
	//}
}

//根据项目ID查询文档列表.
func (m *Document) FindListByBookId(book_id int) (docs []*Document, err error) {
	o := orm.NewOrm()
	_, err = o.QueryTable(m.TableNameWithPrefix()).Filter("book_id", book_id).OrderBy("order_sort").All(&docs)
	return
}

//根据项目ID查询文档一级目录.
func (m *Document) GetMenuTop(book_id int) (docs []*Document, err error) {
	var docsAll []*Document
	o := orm.NewOrm()
	cols := []string{"document_id", "document_name", "member_id", "parent_id", "book_id", "identify"}
	_, err = o.QueryTable(m.TableNameWithPrefix()).Filter("book_id", book_id).Filter("parent_id", 0).OrderBy("order_sort").Limit(5000).All(&docsAll, cols...)
	//以"."开头的文档标识，不在阅读目录显示
	for _, doc := range docsAll {
		if !strings.HasPrefix(doc.Identify, ".") {
			docs = append(docs, doc)
		}
	}
	return
}

//自动生成下一级的内容
func (m *Document) BookStackAuto(bookId, docId int) (md, cont string) {
	//自动生成文档内容
	var docs []Document
	orm.NewOrm().QueryTable("md_documents").Filter("book_id", bookId).Filter("parent_id", docId).OrderBy("order_sort").All(&docs, "document_id", "document_name", "identify")
	var newCont []string //新HTML内容
	var newMd []string   //新markdown内容
	for _, idoc := range docs {
		newMd = append(newMd, fmt.Sprintf(`- [%v]($%v)`, idoc.DocumentName, idoc.Identify))
		newCont = append(newCont, fmt.Sprintf(`<li><a href="$%v">%v</a></li>`, idoc.Identify, idoc.DocumentName))
	}
	md = strings.Join(newMd, "\n")
	cont = "<ul>" + strings.Join(newCont, "") + "</ul>"
	return
}

//爬虫批量采集
//@param		html				html
//@param		md					markdown内容
//@return		content,markdown	把链接替换为标识后的内容
func (m *Document) BookStackCrawl(html, md string, bookId, uid int) (content, markdown string, err error) {
	var gq *goquery.Document
	content = html
	markdown = md
	//执行采集
	if gq, err = goquery.NewDocumentFromReader(strings.NewReader(content)); err == nil {
		//采集模式mode
		CrawlByChrome := false
		if strings.ToLower(gq.Find("mode").Text()) == "chrome" {
			CrawlByChrome = true
		}
		beego.Error("chome", CrawlByChrome)
		//内容选择器selector
		selector := ""
		if selector = strings.ToLower(gq.Find("selector").Text()); selector == "" {
			err = errors.New("内容选择器不能为空")
			return
		}

		gq.Find("a").Each(func(i int, selection *goquery.Selection) {
			if href, ok := selection.Attr("href"); ok {
				hrefLower := strings.ToLower(href)
				//以http或者https开头
				if strings.HasPrefix(hrefLower, "http://") || strings.HasPrefix(hrefLower, "https://") {
					//采集文章内容成功，创建文档，填充内容，替换链接为标识
					if retmd, err := utils.CrawlHtml2Markdown(href, 0, CrawlByChrome, 2, selector); err == nil {
						var doc Document
						identify := strconv.Itoa(i) + ".md"
						doc.Identify = identify
						doc.BookId = bookId
						doc.Version = time.Now().Unix()
						doc.ModifyAt = int(time.Now().Unix())
						doc.DocumentName = selection.Text()
						doc.MemberId = uid

						if docId, err := doc.InsertOrUpdate(); err != nil {
							beego.Error("InsertOrUpdate => ", err)
						} else {
							var ds DocumentStore
							ds.DocumentId = int(docId)
							ds.Markdown = retmd
							if err := new(DocumentStore).InsertOrUpdate(ds, "markdown", "content"); err != nil {
								beego.Error(err)
							}
						}
						selection = selection.SetAttr("href", "$"+identify)
						markdown = strings.Replace(markdown, href, "$"+identify, -1)
					} else {
						beego.Error(err.Error())
					}
				}
			}
		})
		content, _ = gq.Find("body").Html()
	}
	return
}
