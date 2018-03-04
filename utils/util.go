package utils

import (
	"fmt"
	"os"

	"strconv"
	"strings"

	html1 "html/template"

	"github.com/TruthHun/BookStack/conf"
	"github.com/TruthHun/html2article"
	"github.com/alexcesaro/mail/mailer"

	"net/mail"

	"path/filepath"

	"os/exec"

	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/TruthHun/gotil/util"
	"github.com/astaxie/beego"
	"github.com/huichen/sego"

	"sync"

	"github.com/TruthHun/html2md"
)

//存储类型

//更多存储类型有待扩展
const (
	StoreLocal string = "local"
	StoreOss   string = "oss"
)

//分词器
var (
	Segmenter       sego.Segmenter
	ReleaseMaps     = make(map[int]bool) //是否正在发布内容，map[book_id]bool
	ReleaseMapsLock sync.RWMutex
	BasePath, _            = filepath.Abs(filepath.Dir(os.Args[0]))
	StoreType       string = beego.AppConfig.String("store_type") //存储类型
)

func init() {
	//加载分词字典
	go func() {
		Segmenter.LoadDictionary(BasePath + "/dictionary/dictionary.txt")
	}()
}

//分词
//@param            str         需要分词的文字
func SegWord(str interface{}) (wds string) {
	//如果已经成功加载字典
	if Segmenter.Dictionary() != nil {
		wds = sego.SegmentsToString(Segmenter.Segment([]byte(fmt.Sprintf("%v", str))), true)
		var wdslice []string
		slice := strings.Split(wds, " ")
		for _, wd := range slice {
			w := strings.Split(wd, "/")[0]
			if (strings.Count(w, "") - 1) >= 2 {
				if i, _ := strconv.Atoi(w); i == 0 { //如果为0，则表示非数字
					wdslice = append(wdslice, w)
				}
			}
		}
		wds = strings.Join(wdslice, ",")
	}
	return
}

//评分处理
func ScoreFloat(score int) string {
	return fmt.Sprintf("%1.1f", float32(score)/10.0)
}

//@param            conf            邮箱配置
//@param            subject         邮件主题
//@param            email           收件人
//@param            body            邮件内容
func SendMail(conf *conf.SmtpConf, subject, email string, body string) error {
	msg := &mail.Message{
		mail.Header{
			"From":         {conf.FormUserName},
			"To":           {email},
			"Reply-To":     {conf.ReplyUserName},
			"Subject":      {subject},
			"Content-Type": {"text/html"},
		},
		strings.NewReader(body),
	}
	port := conf.SmtpPort
	host := conf.SmtpHost
	username := conf.FormUserName
	password := conf.SmtpPassword
	m := mailer.NewMailer(host, username, password, port)
	return m.Send(msg)
}

//渲染markdown为html并录入数据库
func RenderDocumentById(id int) {
	//使用chromium-browser
	//	chromium-browser --headless --disable-gpu --screenshot --no-sandbox --window-size=320,480 http://www.bookstack.cn
	link := "http://localhost:" + beego.AppConfig.DefaultString("httpport", "8080") + "/local-render?id=" + strconv.Itoa(id)
	chrome := beego.AppConfig.DefaultString("chrome", "chromium-browser")
	args := []string{"--headless", "--disable-gpu", "--screenshot", "--no-sandbox", "--window-size=320,480", link}
	cmd := exec.Command(chrome, args...)
	if err := cmd.Run(); err != nil {
		beego.Error(err)
	}
}

//使用chrome采集网页HTML
func CrawlByChrome(urlstr string) (b []byte, err error) {
	chrome := beego.AppConfig.DefaultString("chrome", "chromium-browser")
	args := []string{"--headless", "--disable-gpu", "--dump-dom", "--no-sandbox", urlstr}
	cmd := exec.Command(chrome, args...)
	return cmd.Output()
}

//采集HTML并把相对链接和相对图片
//内容类型，contType:0表示markdown，1表示html，2表示文本
//force:是否是强力采集
//intelligence:是否是智能提取，智能提取，使用html2article，否则提取body
func CrawlHtml2Markdown(urlstr string, contType int, force, intelligence bool, headers ...map[string]string) (cont string, err error) {
	if force {
		var b []byte
		b, err = CrawlByChrome(urlstr)
		cont = string(b)
	} else {
		req := util.BuildRequest("get", urlstr, "", "", "", true, false, headers...)
		req.SetTimeout(10*time.Second, 30*time.Second)
		cont, err = req.String()
	}

	if err == nil {
		//http://www.bookstack.cn/login.html
		slice := strings.Split(strings.TrimRight(urlstr, "/")+"/", "/")
		if sliceLen := len(slice); sliceLen > 2 {
			var doc *goquery.Document
			if doc, err = goquery.NewDocumentFromReader(strings.NewReader(cont)); err == nil {
				//遍历a标签替换相对链接
				doc.Find("a").Each(func(i int, selection *goquery.Selection) {
					//存在href，且不以http://和https://开头
					if href, ok := selection.Attr("href"); ok && (!strings.HasPrefix(strings.ToLower(href), "http://") && !strings.HasPrefix(strings.ToLower(href), "https://") && !strings.HasPrefix(strings.ToLower(href), "#")) {
						if strings.HasPrefix(href, "/") {
							selection.SetAttr("href", strings.Join(slice[0:3], "/")+href)
						} else {
							l := strings.Count(href, "../")
							//需要多减1，因为"http://"或"https://"后面多带一个斜杠
							selection.SetAttr("href", strings.Join(slice[0:sliceLen-l-1], "/")+"/"+strings.TrimLeft(href, "./"))
						}
					}
				})

				//遍历替换图片相对链接
				doc.Find("img").Each(func(i int, selection *goquery.Selection) {
					//存在href，且不以http://和https://开头
					if src, ok := selection.Attr("src"); ok && (!strings.HasPrefix(strings.ToLower(src), "http://") && !strings.HasPrefix(strings.ToLower(src), "https://")) {
						if strings.HasPrefix(src, "/") { //以斜杠开头
							//TODO: 域名+src
							selection.SetAttr("src", strings.Join(slice[0:3], "/")+src)
						} else {
							l := strings.Count(src, "../")
							//需要多减1，因为"http://"或"https://"后面多带一个斜杠
							selection.SetAttr("src", strings.Join(slice[0:sliceLen-l-1], "/")+"/"+strings.TrimLeft(src, "./"))
						}
					}
				})

				//h1-h6标题中不要存在链接或者图片，所以提取文本
				Hs := []string{"h1", "h2", "h3", "h4", "h5", "h6"}
				for _, tag := range Hs {
					doc.Find(tag).Each(func(i int, selection *goquery.Selection) {
						//存在href，且不以http://和https://开头
						selection.SetText(selection.Text())
					})
				}

				cont, err = doc.Html()
				if intelligence {
					ext, err := html2article.NewFromHtml(cont)
					if err != nil {
						return cont, err
					}
					article, err := ext.ToArticle()
					if err != nil {
						return cont, err
					}
					switch contType {
					case 1: //=>html
						cont = article.Html + "<br/><br/><br/>原文：" + urlstr
					case 2: //=>text
						cont = article.Content + fmt.Sprintf("\n\r\n\r原文:%v", urlstr)
					default: //0 && other=>markdown
						cont = html2md.Convert(article.Html) + fmt.Sprintf("\n\r\n\r原文:[%v](%v)", urlstr, urlstr)
					}
				} else {
					//移除body中的所有js标签
					doc.Find("script").Each(func(i int, selection *goquery.Selection) {
						selection.Remove()
					})

					switch contType {
					case 1: //=>html
						htmlstr, _ := doc.Find("body").Html()
						cont = htmlstr + "<br/><br/><br/>原文：" + urlstr
					case 2: //=>text
						cont = doc.Find("body").Text() + fmt.Sprintf("\n\r\n\r原文:%v", urlstr)
					default: //0 && other=>markdown
						htmlstr, _ := doc.Find("body").Html()
						cont = html2md.Convert(htmlstr) + fmt.Sprintf("\n\r\n\r原文:[%v](%v)", urlstr, urlstr)
					}
				}

			} else {
				beego.Error(err)
			}
		}
	}

	return
}

//操作图片显示
//如果用的是oss存储，这style是avatar、cover可选项
func ShowImg(img string, style ...string) (url string) {
	if strings.HasPrefix(img, "https://") || strings.HasPrefix(img, "http://") {
		return img
	}
	img = "/" + strings.TrimLeft(img, "./")
	switch StoreType {
	case StoreOss:
		s := ""
		if len(style) > 0 && strings.TrimSpace(style[0]) != "" {
			s = "/" + style[0]
		}
		url = strings.TrimRight(beego.AppConfig.String("oss::Domain"), "/ ") + img + s
	case StoreLocal:
		url = img
	}
	return
}

//分页函数（这个分页函数不具有通用性）
//rollPage:展示分页的个数
//totalRows：总记录
//currentPage:每页显示记录数
//urlPrefix:url链接前缀
//urlParams:url键值对参数
func NewPaginations(rollPage, totalRows, listRows, currentPage int, urlPrefix string, urlSuffix string, urlParams ...interface{}) html1.HTML {
	var (
		htmlPage, path string
		pages          []int
		params         []string
	)
	//总页数
	totalPage := totalRows / listRows
	if totalRows%listRows > 0 {
		totalPage += 1
	}
	//只有1页的时候，不分页
	if totalPage < 2 {
		return ""
	}
	params_len := len(urlParams)
	if params_len > 0 {
		if params_len%2 > 0 {
			params_len = params_len - 1
		}
		for i := 0; i < params_len; {
			key := strings.TrimSpace(fmt.Sprintf("%v", urlParams[i]))
			val := strings.TrimSpace(fmt.Sprintf("%v", urlParams[i+1]))
			//键存在，同时值不为0也不为空
			if len(key) > 0 && len(val) > 0 && val != "0" {
				params = append(params, key, val)
			}
			i = i + 2
		}
	}

	path = strings.Trim(urlPrefix, "/")
	if len(params) > 0 {
		path = path + "/" + strings.Trim(strings.Join(params, "/"), "/")
	}
	//最后再处理一次“/”，是为了防止urlPrifix参数为空时，出现多余的“/”
	path = "/" + strings.Trim(path, "/")

	if currentPage > totalPage {
		currentPage = totalPage
	}
	if currentPage < 1 {
		currentPage = 1
	}
	index := 0
	rp := rollPage * 2
	for i := rp; i > 0; i-- {
		p := currentPage + rollPage - i
		if p > 0 && p <= totalPage {

			pages = append(pages, p)
		}
	}
	for k, v := range pages {
		if v == currentPage {
			index = k
		}
	}
	pages_len := len(pages)
	if currentPage > 1 {
		htmlPage += fmt.Sprintf(`<li><a class="num" href="`+path+`?page=1">1..</a></li><li><a class="num" href="`+path+`?page=%d%v">«</a></li>`, currentPage-1, urlSuffix)
	}
	if pages_len <= rollPage {
		for _, v := range pages {
			if v == currentPage {
				htmlPage += fmt.Sprintf(`<li class="active"><a href="javascript:void(0);">%d</a></li>`, v)
			} else {
				htmlPage += fmt.Sprintf(`<li><a class="num" href="`+path+`?page=%d%v">%d</a></li>`, v, urlSuffix, v)
			}
		}

	} else {
		index_min := index - rollPage/2
		index_max := index + rollPage/2
		page_slice := make([]int, 0)
		if index_min > 0 && index_max < pages_len { //切片索引未越界
			page_slice = pages[index_min:index_max]
		} else {
			if index_min < 0 {
				page_slice = pages[0:rollPage]
			} else if index_max > pages_len {
				page_slice = pages[(pages_len - rollPage):pages_len]
			} else {
				page_slice = pages[index_min:index_max]
			}

		}

		for _, v := range page_slice {
			if v == currentPage {
				htmlPage += fmt.Sprintf(`<li class="active"><a href="javascript:void(0);">%d</a></li>`, v)
			} else {
				htmlPage += fmt.Sprintf(`<li><a class="num" href="`+path+`?page=%d%v">%d</a></li>`, v, urlSuffix, v)
			}
		}

	}
	if currentPage < totalPage {
		htmlPage += fmt.Sprintf(`<li><a class="num" href="`+path+`?page=%v%v">»</a></li><li><a class="num" href="`+path+`?page=%v%v">..%d</a></li>`, currentPage+1, urlSuffix, totalPage, urlSuffix, totalPage)
	}

	return html1.HTML(`<ul class="pagination">` + htmlPage + `</ul>`)
}
