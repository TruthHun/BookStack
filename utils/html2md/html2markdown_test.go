package html2md

import (
	"io/ioutil"
	"testing"
)

func TestConvert(t *testing.T) {
	b, _ := ioutil.ReadFile("example/presto.html")
	md := Convert(string(b))
	ioutil.WriteFile("example/code.md", []byte(md), 0777)
}
