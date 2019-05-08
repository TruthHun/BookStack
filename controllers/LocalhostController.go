package controllers

import (
	"strings"

	"github.com/PuerkitoBio/goquery"

	"time"

	"github.com/TruthHun/BookStack/models"
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/orm"
)

//只有请求头的host为localhost的才能访问。
type LocalhostController struct {
	BaseController
}

//渲染markdown.
//根据文档id来。
func (this *LocalhostController) RenderMarkdown() {
	if strings.HasPrefix(this.Ctx.Request.Host, "localhost:"+beego.AppConfig.String("httpport")) {
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
				doc, err := goquery.NewDocumentFromReader(strings.NewReader(content))
				if err == nil {
					doc.Find("blockquote br").Each(func(i int, selection *goquery.Selection) {
						selection.Remove()
					})
					content, _ = doc.Find("body").Html()
				}
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
	this.Abort("404")
}
