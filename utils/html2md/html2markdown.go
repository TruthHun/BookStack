//Author:TruthHun
//Email: TruthHun@QQ.COM
//Date:  2018-02-03
package html2md

import (
	"strings"

	"fmt"

	"regexp"

	"github.com/PuerkitoBio/goquery"
	"github.com/astaxie/beego/logs"
)

var tag2tag = map[string]string{
	"b":    "strong",
	"i":    "em",
	"dfn":  "em",
	"var":  "em",
	"cite": "em",
}

var blockTag = []string{
	"address", "div", "figure", "p", "figcaption", "br",
	"article", "aside", "nav", "footer", "fieldset", "menu",
	"header", "section", "center", "frameset", "details", "summary",
}

var nextlineTag = []string{
	"pre", "blockquote", "table",
}

//convert html to markdown
//将html转成markdown
func Convert(htmlstr string) (md string) {
	var maps map[string]string
	doc, _ := goquery.NewDocumentFromReader(strings.NewReader(htmlstr))
	doc = trimAttr(doc)
	doc, maps = compress(doc)
	doc = handleNextLine(doc) //<div>...
	doc = handleBlockTag(doc) //<div>...
	doc = handleA(doc)        //<a>
	doc = handleImg(doc)      //<img>
	doc = handleHead(doc)     //h1~h6
	doc = handleTag2Tag(doc)  //<strong>、<i>、eg..
	doc = handleHr(doc)       //<hr>
	doc = handleLi(doc)       //<li>
	md, _ = doc.Find("body").Html()
	md = depress(md, maps)
	return
}

// 解压，释放code和pre
func depress(md string, maps map[string]string) string {
	// 先替换pre，再替换code，因为有的code在pre标签里面
	for key, val := range maps {
		if strings.HasPrefix(key, "{$blockquote") {
			md = strings.Replace(md, key, "\n\r"+val+"\n\r", -1)
		}
	}

	for key, val := range maps {
		if strings.HasPrefix(key, "{$pre") {
			md = strings.Replace(md, key, "\n\r"+val+"\n\r", -1)
		}
	}

	for key, val := range maps {
		if strings.HasPrefix(key, "{$code") || strings.HasPrefix(key, "{$textarea") {
			md = strings.Replace(md, key, val, -1)
		}
	}

	if doc, err := goquery.NewDocumentFromReader(strings.NewReader(md)); err == nil {
		doc = trimAttr(doc)
		backslashes := []string{"+", "-", "_", "*"}
		doc.Find("code").Each(func(i int, selection *goquery.Selection) {
			if !selection.Parent().Is("pre") {
				text := selection.Text()
				for _, item := range backslashes {
					text = strings.Replace(text, item, "\\"+item, -1)
				}
				selection.SetHtml(text)
			}
		})
		md, _ = doc.Find("body").Html()
		md = strings.Replace(md, "<span>", "", -1)
		md = strings.Replace(md, "</span>", "", -1)
	}
	return md
}

// trip attr
func trimAttr(doc *goquery.Document) *goquery.Document {
	attrs := []string{
		"border", "colspan", "rowspan", "style", "cellspacing",
		"cellpadding", "bgcolor", "width", "align", "frame", "id", "class",
	}
	elements := []string{
		"table", "thead", "tbody", "tr", "td", "th", "h1", "h2", "h3", "h4", "img",
		"h5", "h6", "i", "em", "strong", "span", "br", "hr", "ul", "li", "ol",
	}
	elements = append(elements, blockTag...)
	elements = append(elements, nextlineTag...)
	for _, tag := range elements {
		doc.Find(tag).Each(func(i int, selection *goquery.Selection) {
			for _, attr := range attrs {
				selection.RemoveAttr(attr)
			}
		})
	}
	return doc
}

//压缩html
func compress(doc *goquery.Document) (*goquery.Document, map[string]string) {
	//blockquote、pre、code，并替换 span 为空

	var maps = make(map[string]string)

	if ele := doc.Find("textarea"); len(ele.Nodes) > 0 {
		ele.Each(func(i int, selection *goquery.Selection) {
			key := fmt.Sprintf("{$textarea%v}", i)
			cont := "<textarea>" + getInnerHtml(selection) + "</textarea>"
			selection.BeforeHtml(key)
			selection.Remove()
			maps[key] = cont
		})
	}

	if ele := doc.Find("code"); len(ele.Nodes) > 0 {
		ele.Each(func(i int, selection *goquery.Selection) {
			key := fmt.Sprintf("{$code%v}", i)
			cont := "<code>" + getInnerHtml(selection) + "</code>"
			selection.BeforeHtml(key)
			selection.Remove()
			maps[key] = cont
		})
	}

	if ele := doc.Find("pre"); len(ele.Nodes) > 0 {
		ele.Each(func(i int, selection *goquery.Selection) {
			key := fmt.Sprintf("{$pre%v}", i)
			cont := "<pre>" + getInnerHtml(selection) + "</pre>"
			selection.BeforeHtml(key)
			selection.Remove()
			maps[key] = cont
		})
	}

	if ele := doc.Find("blockquote"); len(ele.Nodes) > 0 {
		ele.Each(func(i int, selection *goquery.Selection) {
			key := fmt.Sprintf("{$blockquote%v}", i)
			cont := "<blockquote>" + getInnerHtml(selection) + "</blockquote>"
			selection.BeforeHtml(key)
			selection.Remove()
			maps[key] = cont
		})
	}

	replaces := map[string]string{
		"\n": " ", "\r": " ", "\t": " ", "<dl": "<ul",
		"</dl": "</ul", "<dt": "<li", "</dt": "</li",
		"<dd": "<li", "</dd": "</li",
	}

	htmlstr, _ := doc.Html()
	for old, new := range replaces {
		htmlstr = strings.Replace(htmlstr, old, new, -1)
	}

	//正则匹配，把“>”和“<”直接的空格全部去掉
	//去除标签之间的空格，如果是存在代码预览的页面，不要替换空格，否则预览的代码会错乱
	r, _ := regexp.Compile(">\\s+<")
	htmlstr = r.ReplaceAllString(htmlstr, "> <")
	//多个空格替换成一个空格
	r2, _ := regexp.Compile("\\s+")
	htmlstr = r2.ReplaceAllString(htmlstr, " ")
	doc, _ = goquery.NewDocumentFromReader(strings.NewReader(htmlstr))
	return doc, maps
}

func handleBlockTag(doc *goquery.Document) *goquery.Document {
	for _, tag := range blockTag {
		hasTag := true
		for hasTag {
			if tagEle := doc.Find(tag); len(tagEle.Nodes) > 0 {
				tagEle.Each(func(i int, selection *goquery.Selection) {
					selection.BeforeHtml("\n" + getInnerHtml(selection) + "\n")
					selection.Remove()
				})
			} else {
				hasTag = false
			}
		}
	}
	return doc
}

//func handleBlockquote(doc *goquery.Document) *goquery.Document {
//	if tagEle := doc.Find("blockquote"); len(tagEle.Nodes) > 0 {
//		tagEle.Each(func(i int, selection *goquery.Selection) {
//			cont := getInnerHtml(selection)
//			cont = strings.Replace(cont, "\r", "", -1)
//			cont = strings.Replace(cont, "\n", "", -1)
//			selection.BeforeHtml("\r\n<blockquote>" + cont + "\n</blockquote>\n")
//			selection.Remove()
//		})
//	}
//
//	doc.Find("code").Each(func(i int, selection *goquery.Selection) {
//		fmt.Println(selection.Html())
//	})
//
//	return doc
//}

//[ok]handle tag <a>
func handleA(doc *goquery.Document) *goquery.Document {
	doc.Find("a").Each(func(i int, selection *goquery.Selection) {
		if href, ok := selection.Attr("href"); ok {
			if cont, err := selection.Html(); err == nil {
				md := fmt.Sprintf(`[%v](%v)`, cont, href)
				selection.BeforeHtml(md)
				selection.Remove()
			}
		}
	})
	return doc
}

//[ok]handle tag ul、ol、li
//处理步骤：
//1、先给每个li标签里面的内容加上"- "或者"\t- "
//2、提取li内容
func handleLi(doc *goquery.Document) *goquery.Document {
	var tags = []string{"ol", "ul", "li"}
	doc.Find("li").Each(func(i int, selection *goquery.Selection) {
		l := len(selection.ParentsFiltered("li").Nodes)
		tab := strings.Join(make([]string, l+2), "{$@$space}")
		selection.PrependHtml("\r$@$" + tab)
	})
	for _, tag := range tags {
		doc.Find(tag).Each(func(i int, selection *goquery.Selection) {
			if tag == "ul" || tag == "ol" {
				text := "\n" + selection.Text() + "\n"
				if !(selection.Parent().Is("ul") || selection.Parent().Is("ol")) {
					text = "\n" + text + "\n"
				}
				selection.BeforeHtml(text)
			} else {
				selection.BeforeHtml(selection.Text())
			}
			selection.Remove()
		})
	}
	htmlstr, _ := doc.Find("body").Html()
	for i := 10; i > 0; i-- {
		oldTab := "$@$" + strings.Join(make([]string, i), "{$@$space}")
		newTab := strings.Join(make([]string, i-1), "  ") + "- "
		htmlstr = strings.Replace(htmlstr, oldTab, newTab, -1)
	}
	doc, _ = goquery.NewDocumentFromReader(strings.NewReader(htmlstr))
	return doc
}

//[ok]handle tag <hr/>
func handleHr(doc *goquery.Document) *goquery.Document {
	doc.Find("hr").Each(func(i int, selection *goquery.Selection) {
		selection.BeforeHtml("\n- - -\n")
		selection.Remove()
	})
	return doc
}

//[ok]handle tag <img/>
func handleImg(doc *goquery.Document) *goquery.Document {
	doc.Find("img").Each(func(i int, selection *goquery.Selection) {
		if src, ok := selection.Attr("src"); ok {
			alt := ""
			if val, ok := selection.Attr("alt"); ok {
				alt = val
			}
			md := fmt.Sprintf(`![%v](%v)`, alt, src)
			selection.BeforeHtml(md)
			selection.Remove()
		}
	})
	return doc
}

//[ok]handle tag h1~h6
func handleHead(doc *goquery.Document) *goquery.Document {
	heads := map[string]string{
		"title": "# ",
		"h1":    "# ",
		"h2":    "## ",
		"h3":    "### ",
		"h4":    "#### ",
		"h5":    "##### ",
		"h6":    "###### ",
	}
	for tag, replace := range heads {
		doc.Find(tag).Each(func(i int, selection *goquery.Selection) {
			text, _ := selection.Html()
			selection.BeforeHtml("\n\r" + replace + text + "\n\r")
			selection.Remove()
		})
	}
	return doc
}

func handleTag2Tag(doc *goquery.Document) *goquery.Document {
	for tag, toTag := range tag2tag {
		doc.Find(tag).Each(func(i int, selection *goquery.Selection) {
			if text, _ := selection.Html(); strings.TrimSpace(text) != "" {
				selection.BeforeHtml(fmt.Sprintf("<%v>%v</%v>", toTag, text, toTag))
			}
			selection.Remove()
		})
	}
	return doc
}

func handleNextLine(doc *goquery.Document) *goquery.Document {
	for _, tag := range nextlineTag {
		doc.Find(tag).Each(func(i int, selection *goquery.Selection) {
			selection.BeforeHtml("\n\n")
			selection.AfterHtml("\n\n")
		})
	}
	return doc
}

func getInnerHtml(selection *goquery.Selection) (html string) {
	var err error
	html, _ = selection.Html()
	if err != nil {
		logs.Error(err)
	}
	return
}
