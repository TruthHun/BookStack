package controllers

import (
	"strconv"
	"strings"
	"time"

	"github.com/TruthHun/BookStack/models"
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/orm"
)

//只有请求头的host为localhost的才能访问。
type LocalhostController struct {
	BaseController
}

func (c *LocalhostController) Prepare() {
	c.NoNeedLoginRouter = true
	prefix := "localhost"
	port := beego.AppConfig.String("httpport")
	if port != "80" {
		prefix = prefix + ":" + port
	}
	if !strings.HasPrefix(c.Ctx.Request.Host, prefix) {
		c.Abort("404")
		return
	}
}

//渲染markdown.
//根据文档id来。
func (this *LocalhostController) RenderMarkdown() {
	id, _ := this.GetInt("id")
	if id > 0 {
		var doc models.Document
		ModelStore := new(models.DocumentStore)
		o := orm.NewOrm()
		qs := o.QueryTable("md_documents").Filter("document_id", id)
		if this.Ctx.Input.IsPost() {
			qs.One(&doc, "identify", "book_id")
			var book models.Book
			o.QueryTable("md_books").Filter("book_id", doc.BookId).One(&book, "identify")
			content := this.GetString("content")
			//doc, err := goquery.NewDocumentFromReader(strings.NewReader(content))
			//if err == nil {
			//	doc.Find("br").Each(func(i int, selection *goquery.Selection) {
			//		selection.Remove()
			//	})
			//	content, _ = doc.Find("body").Html()
			//}
			content = this.replaceLinks(book.Identify, content)
			qs.Update(orm.Params{
				"release":     content,
				"modify_time": time.Now(),
			})
			//这里要指定更新字段，否则markdown内容会被置空
			ModelStore.InsertOrUpdate(models.DocumentStore{DocumentId: id, Content: content}, "content")
			this.JsonResult(0, "成功")
		}

		this.Data["Markdown"] = ModelStore.GetFiledById(id, "markdown")
		this.TplName = "widgets/render.html"
		return
	}
}

// 渲染生成封面截图
func (this *LocalhostController) RenderCover() {
	identify := this.GetString("id")
	id, err := strconv.Atoi(identify)
	if identify == "" && err != nil {
		this.Abort("404")
		return
	}
	book := models.NewBook()
	q := orm.NewOrm().QueryTable(book)
	if id > 0 {
		err = q.Filter("book_id", id).One(book)
	} else {
		err = q.Filter("identify", identify).One(book)
	}
	if err != nil {
		beego.Error(err)
	}
	if book.BookId == 0 {
		this.Abort("404")
		return
	}
	this.Data["Book"] = book
	this.TplName = "ebook/cover.html"
}
