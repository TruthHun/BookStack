package html2json

import (
	"encoding/json"
	"strings"

	"golang.org/x/net/html"

	"github.com/PuerkitoBio/goquery"
)

type h2j struct {
	Name     string            `json:"name,omitempty"` // 对应 HTML 标签
	Type     string            `json:"type,omitempty"` // element 或者 text
	Text     string            `json:"text,omitempty"`
	Attrs    map[string]string `json:"attrs,omitempty"`
	Children []h2j             `json:"children,omitempty"`
}

func Parse(htmlStr string) (js string, err error) {

	var (
		doc *goquery.Document
		m   []h2j
		b   []byte
	)

	doc, err = goquery.NewDocumentFromReader(strings.NewReader(htmlStr))
	if err != nil {
		return
	}
	doc.Find("body").Each(func(i int, selection *goquery.Selection) {
		m = parse(selection)
	})

	b, err = json.Marshal(m)
	if err == nil {
		js = string(b)
	}
	return
}

func ParseByDom(doc *goquery.Document) (js string, err error) {

	var (
		m []h2j
		b []byte
	)

	doc.Each(func(i int, selection *goquery.Selection) {
		m = parse(selection)
	})
	if len(m) == 0 {
		return
	}
	b, err = json.Marshal(m)
	if err == nil {
		js = string(b)
	}

	return
}

func parse(sel *goquery.Selection) (data []h2j) {
	nodes := sel.Children().Nodes
	if len(nodes) == 0 {
		if txt := sel.Text(); txt != "" {
			data = []h2j{{Text: txt, Type: "text"}}
		}
		return
	}
	sel.Contents().FilterFunction(func(i int, s *goquery.Selection) bool {
		ns := s.Nodes
		for _, item := range ns {
			var h h2j
			if item.Type != html.TextNode {
				h.Name = item.Data
				attr := make(map[string]string)
				for _, a := range item.Attr {
					attr[a.Key] = a.Val
				}
				if h.Name == "pre" {
					h.Name = "div"
					if class, ok := attr["class"]; ok {
						attr["class"] = "tag-pre " + class
					} else {
						attr["class"] = "tag-pre"
					}
				}
				h.Attrs = attr
				h.Children = parse(goquery.NewDocumentFromNode(item).Selection)
			} else {
				h.Type = "text"
				h.Text = goquery.NewDocumentFromNode(item).Selection.Text()
			}
			data = append(data, h)
		}
		return true
	})
	return
}
