package models

import (
	"os"
	"strings"
	"time"

	"github.com/TruthHun/gotil/mdtil"

	"bytes"
	"fmt"

	"io/ioutil"

	"path/filepath"

	"crypto/tls"

	"errors"

	"github.com/PuerkitoBio/goquery"
	"github.com/TruthHun/BookStack/conf"
	"github.com/TruthHun/BookStack/models/store"
	"github.com/TruthHun/BookStack/utils"
	"github.com/TruthHun/converter/converter"
	"github.com/TruthHun/gotil/cryptil"
	"github.com/TruthHun/gotil/util"
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/httplib"
	"github.com/astaxie/beego/orm"
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
//func (m *Document) TableUnique() [][]string {
//	return [][]string{
//		[]string{"BookId", "Identify"},
//	}
//}

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
func (m *Document) Find(id int) (doc *Document, err error) {
	if id <= 0 {
		return m, ErrInvalidParameter
	}

	o := orm.NewOrm()

	err = o.QueryTable(m.TableNameWithPrefix()).Filter("document_id", id).One(m)
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
	m.DocumentName = strings.TrimSpace(m.DocumentName)
	if m.DocumentId > 0 { //文档id存在，则更新
		_, err = o.Update(m, cols...)
		return
	}

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
	q := orm.NewOrm().QueryTable(m.TableNameWithPrefix()).Filter("BookId", BookId)
	err := q.Filter("Identify", Identify).One(m)
	if m.DocumentId == 0 && !strings.HasSuffix(fmt.Sprint(Identify), ".md") {
		err = q.Filter("identify", fmt.Sprint(Identify)+".md").One(m)
	}
	return m, err
}

//递归删除一个文档.
func (m *Document) RecursiveDocument(docId int) error {

	o := orm.NewOrm()
	modelStore := new(DocumentStore)

	if doc, err := m.Find(docId); err == nil {
		o.Delete(doc)
		modelStore.DeleteById(docId)
		NewDocumentHistory().Clear(docId)
	}

	var docs []*Document

	_, err := o.QueryTable(m.TableNameWithPrefix()).Filter("parent_id", docId).All(&docs)
	if err != nil {
		beego.Error("RecursiveDocument => ", err)
		return err
	}

	for _, item := range docs {
		docId := item.DocumentId
		o.QueryTable(m.TableNameWithPrefix()).Filter("document_id", docId).Delete()
		//删除document_store表的文档
		modelStore.DeleteById(docId)
		m.RecursiveDocument(docId)
	}
	return nil
}

//发布文档内容为HTML
func (m *Document) ReleaseContent(bookId int, baseUrl string) {
	// 加锁
	utils.BooksRelease.Set(bookId)
	defer utils.BooksRelease.Delete(bookId)

	var (
		o           = orm.NewOrm()
		docs        []*Document
		book        Book
		tableBooks  = "md_books"
		releaseTime = time.Now() //发布的时间戳
	)

	qs := o.QueryTable(tableBooks).Filter("book_id", bookId)
	qs.One(&book)

	//全部重新发布。查询该书籍的所有文档id
	_, err := o.QueryTable(m.TableNameWithPrefix()).Filter("book_id", bookId).Limit(20000).All(&docs, "document_id", "identify")
	if err != nil {
		beego.Error("发布失败 => ", err)
		return
	}
	docMap := make(map[string]bool)
	for _, item := range docs {
		docMap[item.Identify] = true
	}

	ModelStore := new(DocumentStore)
	for _, item := range docs {
		ds, err := ModelStore.GetById(item.DocumentId)
		if err != nil {
			beego.Error(err)
			continue
		}
		if len(utils.GetTextFromHtml(ds.Content)) == 0 {
			//内容为空，渲染一下文档，然后再重新获取
			utils.RenderDocumentById(item.DocumentId)
			ds, _ = ModelStore.GetById(item.DocumentId)
		}

		item.Release = ds.Content
		attachList, err := NewAttachment().FindListByDocumentId(item.DocumentId)
		if err == nil && len(attachList) > 0 {
			content := bytes.NewBufferString("<div class=\"attach-list\"><strong>附件</strong><ul>")
			for _, attach := range attachList {
				li := fmt.Sprintf("<li><a href=\"%s\" target=\"_blank\" title=\"%s\">%s</a></li>", attach.HttpPath, attach.FileName, attach.FileName)
				content.WriteString(li)
			}
			content.WriteString("</ul></div>")
			item.Release += content.String()
		}

		// 采集图片与稳定内容连接替换
		if gq, err := goquery.NewDocumentFromReader(strings.NewReader(item.Release)); err == nil {
			images := gq.Find("img")
			if images.Length() > 0 {
				images.Each(func(i int, selection *goquery.Selection) {
					if src, ok := selection.Attr("src"); ok {
						lowerSrc := strings.ToLower(src)
						if strings.HasPrefix(lowerSrc, "https://") || strings.HasPrefix(lowerSrc, "http://") {
							tmpFile, err := utils.DownImage(src)
							if err == nil {
								defer os.Remove(tmpFile)
								var newSrc string
								switch utils.StoreType {
								case utils.StoreLocal:
									newSrc = "/uploads/projects/" + book.Identify + "/" + filepath.Base(tmpFile)
									err = store.ModelStoreLocal.MoveToStore(tmpFile, strings.TrimPrefix(newSrc, "/"))
								case utils.StoreOss:
									newSrc = "projects/" + book.Identify + "/" + filepath.Base(tmpFile)
									err = store.ModelStoreOss.MoveToOss(tmpFile, newSrc, true)
									newSrc = "/" + newSrc
								}
								if err != nil {
									beego.Error(err.Error())
								}
								selection.SetAttr("src", newSrc)
								ds.Markdown = strings.Replace(ds.Markdown, src, newSrc, -1)
							} else {
								beego.Error(err.Error())
							}
						}
					}
				})
			}

			links := gq.Find("a")
			if links.Length() > 0 {
				links.Each(func(i int, selection *goquery.Selection) {
					if href, ok := selection.Attr("href"); ok {
						lowerHref := strings.ToLower(href)
						if strings.HasPrefix(lowerHref, "https://") || strings.HasPrefix(lowerHref, "http://") {
							identify := utils.MD5Sub16(strings.Trim(href, "/")) + ".md"
							if _, ok := docMap[identify]; ok {
								// 替换markdown中的连接，markdown的链接形式：  [链接名称](xxURL)
								ds.Markdown = strings.Replace(ds.Markdown, "("+href+")", "($"+identify+")", -1)
								// 直接identify就好了，比如在 /read/BookIdentify/DocIdentify.md 文档下，xx_identify.md 浏览器会转为 /read/BookIdentify/xx_identify.md
								selection.SetAttr("href", identify)
							}
						}
					}
				})
			}
			item.Release, _ = gq.Find("body").Html()
		}
		ds.Content = item.Release
		ModelStore.InsertOrUpdate(ds, "markdown", "content")

		_, err = o.Update(item, "release")
		if err != nil {
			beego.Error(fmt.Sprintf("发布失败 => %+v", item), err)
		}
	}

	//最后再更新时间戳
	if _, err = qs.Update(orm.Params{
		"release_time": releaseTime,
	}); err != nil {
		beego.Error(err.Error())
	}
	client := NewElasticSearchClient()
	client.RebuildAllIndex(bookId)
}

//离线文档生成
func (m *Document) GenerateBook(book *Book, baseUrl string) {
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

	if beego.AppConfig.DefaultBool("exportCustomCover", false) {
		// 生成书籍封面
		if err = utils.RenderCoverByBookIdentify(book.Identify); err != nil {
			beego.Error(err)
		}
		cover := "cover.png"
		if _, err = os.Stat(folder + cover); err != nil {
			cover = ""
		}
		// 用相对路径
		ExpCfg.Cover = cover
	}

	//生成致谢内容
	statementFile := "ebook/statement.html"
	_, err = os.Stat("views/" + statementFile)
	if err != nil {
		beego.Error(err)
	} else {
		if htmlStr, err := utils.ExecuteViewPathTemplate(statementFile, map[string]interface{}{"Model": book, "Nickname": Nickname, "Date": ExpCfg.Timestamp}); err == nil {
			h1Title := "说明"
			if doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlStr)); err == nil {
				h1Title = doc.Find("h1").Text()
			}
			toc := converter.Toc{
				Id:    time.Now().Nanosecond(),
				Pid:   0,
				Title: h1Title,
				Link:  "statement.html",
			}
			htmlname := folder + toc.Link
			ioutil.WriteFile(htmlname, []byte(htmlStr), os.ModePerm)
			ExpCfg.Toc = append(ExpCfg.Toc, toc)
		}
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
						if utils.StoreType == utils.StoreOss {
							pic = strings.TrimRight(beego.AppConfig.String("oss::Domain"), "/ ") + "/" + strings.TrimLeft(src, "./")
						} else {
							pic = baseUrl + src
						}
					}
					//下载图片，放到folder目录下
					ext := ""
					if picSlice := strings.Split(pic, "?"); len(picSlice) > 0 {
						ext = filepath.Ext(picSlice[0])
					}
					filename := cryptil.Md5Crypt(pic) + ext
					localPic := folder + filename
					req := httplib.Get(pic).SetTimeout(5*time.Second, 5*time.Second)
					if strings.HasPrefix(pic, "https") {
						req.SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true})
					}
					req.Header("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/65.0.3298.4 Safari/537.36")
					if err := req.ToFile(localPic); err == nil { //成功下载图片
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
		if htmlStr, err := utils.ExecuteViewPathTemplate("document/tpl_export.html", map[string]interface{}{"Model": book, "Doc": doc, "BaseUrl": baseUrl, "Nickname": Nickname, "Date": ExpCfg.Timestamp}); err == nil {
			htmlName := folder + toc.Link
			ioutil.WriteFile(htmlName, []byte(htmlStr), os.ModePerm)
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

	cfgFile := folder + "config.json"
	ioutil.WriteFile(cfgFile, []byte(util.InterfaceToJson(ExpCfg)), os.ModePerm)
	if Convert, err := converter.NewConverter(cfgFile, debug); err == nil {
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
	newBookFmt := fmt.Sprintf("projects/%v/books/%v", book.Identify, book.GenerateTime.Unix())

	//删除旧文件
	switch utils.StoreType {
	case utils.StoreOss:
		if err := store.ModelStoreOss.DelFromOss(oldBook+".pdf", oldBook+".epub", oldBook+".mobi"); err != nil { //删除旧版
			beego.Error(err)
		}
	case utils.StoreLocal: //本地存储
		if utils.StoreType == utils.StoreLocal {
			if err = os.RemoveAll(fmt.Sprintf("uploads/projects/%v/books/", book.Identify)); err != nil {
				beego.Error(err)
			}
		}
	}

	exts := []string{".pdf", ".epub", ".mobi"}
	for _, ext := range exts {
		switch utils.StoreType {
		case utils.StoreOss:
			//不要开启gzip压缩，否则会出现文件损坏的情况
			if err := store.ModelStoreOss.MoveToOss(folder+"output/book"+ext, newBookFmt+ext, true, false); err != nil {
				beego.Error(err)
			} else { //设置下载头
				store.ModelStoreOss.SetObjectMeta(newBookFmt+ext, book.BookName+ext)
			}
		case utils.StoreLocal: //本地存储
			store.ModelStoreLocal.MoveToStore(folder+"output/book"+ext, "uploads/"+newBookFmt+ext)
		}
	}
}

//根据项目ID查询文档列表(含文档内容).
func (m *Document) FindListByBookId(bookId int, withoutContent ...bool) (docs []*Document, err error) {
	q := orm.NewOrm().QueryTable(m.TableNameWithPrefix()).Filter("book_id", bookId).OrderBy("order_sort")
	if len(withoutContent) > 0 && withoutContent[0] {
		cols := []string{"document_id", "identify", "document_name", "book_id", "vcnt", "version",
			"modify_time", "member_id", "create_time", "order_sort", "parent_id"}
		_, err = q.All(&docs, cols...)
	} else {
		_, err = q.All(&docs)
	}
	return
}

//根据项目ID查询文档一级目录.
func (m *Document) GetMenuTop(bookId int) (docs []*Document, err error) {
	var docsAll []*Document
	o := orm.NewOrm()
	cols := []string{"document_id", "document_name", "member_id", "parent_id", "book_id", "identify"}
	_, err = o.QueryTable(m.TableNameWithPrefix()).Filter("book_id", bookId).Filter("parent_id", 0).OrderBy("order_sort", "document_id").Limit(5000).All(&docsAll, cols...)
	//以"."开头的文档标识，不在阅读目录显示
	for _, doc := range docsAll {
		if !strings.HasPrefix(doc.Identify, ".") {
			docs = append(docs, doc)
		}
	}
	return
}

func (m *Document) GetParentTitle(pid int) (title string) {
	var d Document
	orm.NewOrm().QueryTable(m).Filter("document_id", pid).One(&d, "document_id", "parent_id", "document_name")
	return d.DocumentName
}

//自动生成下一级的内容
func (m *Document) BookStackAuto(bookId, docId int) (md, cont string) {
	//自动生成文档内容
	var docs []Document
	orm.NewOrm().QueryTable("md_documents").Filter("book_id", bookId).Filter("parent_id", docId).OrderBy("order_sort").All(&docs, "document_id", "document_name", "identify")
	var newCont []string //新HTML内容
	var newMd []string   //新markdown内容
	for _, doc := range docs {
		newMd = append(newMd, fmt.Sprintf(`- [%v]($%v)`, doc.DocumentName, doc.Identify))
		newCont = append(newCont, fmt.Sprintf(`<li><a href="$%v">%v</a></li>`, doc.Identify, doc.DocumentName))
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
	project := ""
	if book, err := NewBook().Find(bookId); err == nil {
		project = book.Identify
	}
	//执行采集
	if gq, err = goquery.NewDocumentFromReader(strings.NewReader(content)); err == nil {
		//采集模式mode
		CrawlByChrome := false
		if strings.ToLower(gq.Find("mode").Text()) == "chrome" {
			CrawlByChrome = true
		}
		//内容选择器selector
		selector := ""
		if selector = strings.TrimSpace(gq.Find("selector").Text()); selector == "" {
			err = errors.New("内容选择器不能为空")
			return
		}

		// 截屏选择器
		if screenshot := strings.TrimSpace(gq.Find("screenshot").Text()); screenshot != "" {
			utils.ScreenShotProjects.Store(project, screenshot)
			defer utils.DeleteScreenShot(project)
		}

		//排除的选择器
		var exclude []string
		if excludeStr := strings.TrimSpace(gq.Find("exclude").Text()); excludeStr != "" {
			slice := strings.Split(excludeStr, ",")
			for _, item := range slice {
				exclude = append(exclude, strings.TrimSpace(item))
			}
		}

		var links = make(map[string]string) //map[url]identify

		gq.Find("a").Each(func(i int, selection *goquery.Selection) {
			if href, ok := selection.Attr("href"); ok {
				if !strings.HasPrefix(href, "$") {
					hrefTrim := strings.TrimRight(href, "/")
					identify := utils.MD5Sub16(hrefTrim) + ".md"
					links[hrefTrim] = identify
					links[href] = identify
				}
			}
		})

		gq.Find("a").Each(func(i int, selection *goquery.Selection) {
			if href, ok := selection.Attr("href"); ok {
				hrefLower := strings.ToLower(href)
				//以http或者https开头
				if strings.HasPrefix(hrefLower, "http://") || strings.HasPrefix(hrefLower, "https://") {
					//采集文章内容成功，创建文档，填充内容，替换链接为标识
					if retMD, err := utils.CrawlHtml2Markdown(href, 0, CrawlByChrome, 2, selector, exclude, links, map[string]string{"project": project}); err == nil {
						var doc Document
						identify := utils.MD5Sub16(strings.TrimRight(href, "/")) + ".md"
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
							ds.Markdown = "[TOC]\n\r\n\r" + retMD
							if err := new(DocumentStore).InsertOrUpdate(ds, "markdown", "content"); err != nil {
								beego.Error(err)
							}
						}
						selection = selection.SetAttr("href", "$"+identify)
						if _, ok := links[href]; ok {
							markdown = strings.Replace(markdown, "("+href+")", "($"+identify+")", -1)
						}
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

// markdown 文档拆分
func (m *Document) SplitMarkdownAndStore(seg string, markdown string, docId int) (err error) {
	var mapReplace = map[string]string{
		"${7}$": "#######",
		"${6}$": "######",
		"${5}$": "#####",
		"${4}$": "####",
		"${3}$": "###",
		"${2}$": "##",
		"${1}$": "#",
	}

	m, err = m.Find(docId)
	if err != nil {
		return
	}

	newIdentifyFmt := "spilt.%v." + m.Identify

	seg = fmt.Sprintf("${%v}$", strings.Count(seg, "#"))
	for i := 7; i > 0; i-- {
		slice := make([]string, i+1)
		k := "\n" + strings.Join(slice, "#")
		markdown = strings.Replace(markdown, k, fmt.Sprintf("\n${%v}$", i), -1)
	}
	contSlice := strings.Split(markdown, seg)

	for idx, val := range contSlice {
		var doc = NewDocument()

		if idx != 0 {
			val = seg + val
		}
		for k, v := range mapReplace {
			val = strings.Replace(val, k, v, -1)
		}

		doc.Identify = fmt.Sprintf(newIdentifyFmt, idx)
		if idx == 0 { //不需要使用newIdentify
			doc = m
		} else {
			doc.OrderSort = idx
			doc.ParentId = m.DocumentId
		}
		doc.Release = ""
		doc.BookId = m.BookId
		doc.Markdown = val
		doc.DocumentName = utils.ParseTitleFromMdHtml(mdtil.Md2html(val))
		doc.Version = time.Now().Unix()
		doc.MemberId = m.MemberId

		if !strings.Contains(doc.Markdown, "[TOC]") {
			doc.Markdown = "[TOC]\r\n" + doc.Markdown
		}

		if docId, err := doc.InsertOrUpdate(); err != nil {
			beego.Error("InsertOrUpdate => ", err)
		} else {
			var ds = DocumentStore{
				DocumentId: int(docId),
				Markdown:   doc.Markdown,
			}
			if err := ds.InsertOrUpdate(ds, "markdown"); err != nil {
				beego.Error(err)
			}
		}

	}
	return
}
