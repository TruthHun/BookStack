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

	"github.com/PuerkitoBio/goquery"
	"github.com/TruthHun/BookStack/conf"
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
	DocumentId   int    `orm:"pk;auto;column(document_id)" json:"doc_id"`
	DocumentName string `orm:"column(document_name);size(500)" json:"doc_name"`
	Identify     string `orm:"column(identify);size(100);index;null;default(null)" json:"identify"` // Identify 文档唯一标识
	BookId       int    `orm:"column(book_id);type(int);index" json:"book_id"`
	ParentId     int    `orm:"column(parent_id);type(int);index;default(0)" json:"parent_id"`
	OrderSort    int    `orm:"column(order_sort);default(0);type(int);index" json:"order_sort"`
	//Markdown     string        `orm:"column(markdown);type(text);null" json:"markdown"` // Markdown markdown格式文档.
	Release string `orm:"column(release);type(text);null" json:"release"` // Release 发布后的Html格式内容.
	//Content      string        `orm:"column(content);type(text);null" json:"content"`   // Content 未发布的 Html 格式内容.
	CreateTime time.Time     `orm:"column(create_time);type(datetime);auto_now_add" json:"create_time"`
	MemberId   int           `orm:"column(member_id);type(int)" json:"member_id"`
	ModifyTime time.Time     `orm:"column(modify_time);type(datetime);default(null);auto_now" json:"modify_time"`
	ModifyAt   int           `orm:"column(modify_at);type(int)" json:"-"`
	Version    int64         `orm:"type(bigint);column(version)" json:"version"`
	AttachList []*Attachment `orm:"-" json:"attach"`
	Vcnt       int           `orm:"column(vcnt);default(0)" json:"vcnt"` //文档项目被浏览次数
	Markdown   string        `orm:"-" json:"markdown"`
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
func (m *Document) InsertOrUpdate(cols ...string) (id int64, err error) {
	o := orm.NewOrm()
	id = int64(m.DocumentId)
	if m.DocumentId > 0 {
		_, err = o.Update(m, cols...)
	} else {
		var mm Document
		//直接查询一个字段，优化MySQL IO
		o.QueryTable("md_documents").Filter("identify", m.Identify).Filter("book_id", m.BookId).One(&mm, "document_id")
		if mm.DocumentId == 0 {
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
	_, err := o.QueryTable(m.TableNameWithPrefix()).Filter("book_id", book_id).Filter("modify_time__gt", book.ReleaseTime).All(&docs, "document_id")
	//_, err := o.QueryTable(m.TableNameWithPrefix()).Filter("book_id", book_id).All(&docs, "document_id", "content")
	if err != nil {
		beego.Error("发布失败 => ", err)
		return
	}
	idx := 1
	ModelStore := new(DocumentStore)
	for _, item := range docs {
		content := strings.TrimSpace(ModelStore.GetFiledById(item.DocumentId, "content"))
		if len(content) == 0 {
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
	//发布的时间戳
	releaseTime := time.Now()

	//最后再更新时间戳
	if _, err = qs.Update(orm.Params{
		"release_time": releaseTime,
	}); err != nil {
		beego.Error(err.Error())
	}
	utils.ReleaseMapsLock.Lock()
	delete(utils.ReleaseMaps, book_id)
	utils.ReleaseMapsLock.Unlock()
}

func (m *Document) GenerateBook(book *Book, base_url string) {
	if book.ReleaseTime == book.GenerateTime && book.GenerateTime.Unix() > 0 { //如果文档没有更新，则直接返回，不再生成文档
		beego.Error("下载文档生成时间跟文档发布时间一致，无需再重新生成下载文档", book)
		return
	}
	qs := orm.NewOrm().QueryTable("md_books").Filter("book_id", book.BookId)
	//更新上一次下载文档生成时间，以起到加锁的作用
	qs.Update(orm.Params{
		"last_click_generate": time.Now(),
	})
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
	folder := fmt.Sprintf("cache/%v/", book.Identify)
	os.MkdirAll(folder, os.ModePerm)
	if !debug {
		defer os.RemoveAll(folder)
	}

	//生成致谢信内容
	if htmlstr, err := utils.ExecuteViewPathTemplate("document/tpl_statement.html", map[string]interface{}{"Model": book, "Nickname": Nickname, "Date": ExpCfg.Timestamp}); err == nil {
		toc := converter.Toc{
			Id:    time.Now().Nanosecond(),
			Pid:   0,
			Title: "致谢",
			Link:  "statement.html",
		}
		htmlname := folder + toc.Link
		ioutil.WriteFile(htmlname, []byte(htmlstr), os.ModePerm)
		ExpCfg.Toc = append(ExpCfg.Toc, toc)
	}
	ModelStore := new(DocumentStore)
	for _, doc := range docs {
		content := strings.TrimSpace(ModelStore.GetFiledById(doc.DocumentId, "content"))
		if content == "" { //内容为空，渲染文档内容，并再重新获取文档内容
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
					if strings.HasPrefix(src, "http") {
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
	newBook := fmt.Sprintf("projects/%v/books/%v", book.Identify, book.ReleaseTime.Unix())
	oldBook := fmt.Sprintf("projects/%v/books/%v", book.Identify, book.GenerateTime.Unix())
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
			ModelStoreLocal.MoveToStore(folder+"output/book"+ext, "uploads/"+newBook+ext, true)
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

	//最后再更新文档生成时间
	if _, err = qs.Update(orm.Params{"generate_time": book.ReleaseTime}); err != nil {
		beego.Error(err.Error())
	}

}

//根据项目ID查询文档列表.
func (m *Document) FindListByBookId(book_id int) (docs []*Document, err error) {
	o := orm.NewOrm()
	_, err = o.QueryTable(m.TableNameWithPrefix()).Filter("book_id", book_id).OrderBy("order_sort").All(&docs)
	return
}

//根据项目ID查询文档一级目录.
func (m *Document) GetMenuTop(book_id int) (docs []*Document, err error) {
	o := orm.NewOrm()
	cols := []string{"document_id", "document_name", "member_id", "parent_id", "book_id", "identify"}
	_, err = o.QueryTable(m.TableNameWithPrefix()).Filter("book_id", book_id).Filter("parent_id", 0).OrderBy("order_sort").Limit(5000).All(&docs, cols...)
	return
}
