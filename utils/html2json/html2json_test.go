package html2json

import "testing"

func TestParse(t *testing.T) {
	str := `<p>abc <code>def</code> ghi</p>`
	t.Log(Parse(str))
}
