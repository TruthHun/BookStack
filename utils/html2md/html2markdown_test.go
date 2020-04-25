package html2md

import (
	"fmt"
	"io/ioutil"
	"regexp"
	"testing"
)

func TestConvert(t *testing.T) {
	b, _ := ioutil.ReadFile("example/presto.html")
	md := Convert(string(b))
	ioutil.WriteFile("example/code.md", []byte(md), 0777)
}

func TestRemoveHTMLComments(t *testing.T) {
	html := `
<span>标签内容1</span><!-- 这是一行注释 --><span>标签内容2</span>
<span>标签内容3</span>
<!-- 这是一行注释<span>标签内容4</span>
<span>标签内容5</span> -->
<div>hello world</div>
<!---->
<!--
  
-->
`
	re, _ := regexp.Compile("\\<\\!\\-\\-(.|(\\n))*?\\-\\-\\>")
	html = re.ReplaceAllString(html, "")
	fmt.Println(html)
}
