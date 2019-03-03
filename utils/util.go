package utils

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"os"

	"strconv"
	"strings"

	html1 "html/template"

	"github.com/TruthHun/BookStack/conf"
	"github.com/TruthHun/BookStack/models/store"
	"github.com/TruthHun/html2article"
	"github.com/alexcesaro/mail/mailer"

	"net/http"
	"net/mail"

	"path/filepath"

	"os/exec"

	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/TruthHun/gotil/util"
	"github.com/astaxie/beego"
	"github.com/huichen/sego"

	"github.com/TruthHun/gotil/cryptil"
	"github.com/TruthHun/html2md"
)

//存储类型

//更多存储类型有待扩展
const (
	Version           = "1.6"
	StoreLocal string = "local"
	StoreOss   string = "oss"
)

//分词器
var (
	Segmenter   sego.Segmenter
	BasePath, _        = filepath.Abs(filepath.Dir(os.Args[0]))
	StoreType   string = beego.AppConfig.String("store_type") //存储类型
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
		Header: mail.Header{
			"From":         {conf.FormUserName},
			"To":           {email},
			"Reply-To":     {conf.ReplyUserName},
			"Subject":      {subject},
			"Content-Type": {"text/html"},
		},
		Body: strings.NewReader(body),
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
	name := beego.AppConfig.DefaultString("chrome", "chromium-browser")
	args := []string{"--headless", "--disable-gpu", "--screenshot", "--no-sandbox", "--window-size=320,480", link}
	if ok, _ := beego.AppConfig.Bool("puppeteer"); ok {
		name = "node"
		args = []string{"crawl.js", "--url", link}
	}
	cmd := exec.Command(name, args...)
	beego.Info("render document by document_id:", cmd.Args)
	// 超过10秒，杀掉进程，避免长期占用
	time.AfterFunc(30*time.Second, func() {
		if cmd.Process.Pid != 0 {
			cmd.Process.Kill()
		}
	})
	if err := cmd.Run(); err != nil {
		beego.Error(err)
	}
}

//使用chrome采集网页HTML
func CrawlByChrome(urlStr string) (b []byte, err error) {
	if strings.Contains(urlStr, "bookstack") {
		return
	}
	var args []string
	name := beego.AppConfig.DefaultString("chrome", "chromium-browser")
	if ok, _ := beego.AppConfig.Bool("puppeteer"); ok {
		name = "node"
		args = []string{"crawl.js", "--url", urlStr}
	} else { // chrome
		args = []string{"--headless", "--disable-gpu", "--dump-dom", "--no-sandbox", urlStr}
	}
	cmd := exec.Command(name, args...)

	// 超过10秒，杀掉进程，避免长期占用
	time.AfterFunc(30*time.Second, func() {
		if cmd.Process.Pid != 0 {
			cmd.Process.Kill()
		}
	})

	return cmd.Output()
}

//采集HTML并把相对链接和相对图片
//内容类型，contType:0表示markdown，1表示html，2表示文本
//force:是否是强力采集
//intelligence:是否是智能提取，智能提取，使用html2article，否则提取body
//diySelecter:自定义选择器
//注意：由于参数问题，采集并下载图片的话，在headers中加上key为"project"的字段，值为文档项目的标识
func CrawlHtml2Markdown(urlstr string, contType int, force bool, intelligence int, diySelector string, excludeSelector []string, headers ...map[string]string) (cont string, err error) {

	//记录已经存在了的图片，避免重复图片出现重复采集的情况
	var existImage bool
	imageMap := make(map[string]string)

	if strings.Contains(urlstr, "bookstack.cn") {
		return
	}

	if force { //强力模式
		var b []byte
		b, err = CrawlByChrome(urlstr)
		cont = string(b)
	} else {
		req := util.BuildRequest("get", urlstr, "", "", "", true, false, headers...)
		req.SetTimeout(10*time.Second, 10*time.Second)
		cont, err = req.String()
	}

	if err != nil {
		return
	}

	//http://www.bookstack.cn/login.html
	slice := strings.Split(strings.TrimSpace(urlstr), "/")
	if sliceLen := len(slice); sliceLen > 2 {
		var doc *goquery.Document
		doc, err = goquery.NewDocumentFromReader(strings.NewReader(cont))
		if err != nil {
			return
		}

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
			//存在src，且不以http://和https://开头
			if src, ok := selection.Attr("src"); ok {
				//链接补全
				srcLower := strings.ToLower(src)
				if !strings.HasPrefix(srcLower, "http://") &&
					!strings.HasPrefix(srcLower, "https://") &&
					!strings.HasPrefix(srcLower, "data:image/") { //非base64的任意
					if strings.HasPrefix(src, "/") { //以斜杠开头
						src = strings.Join(slice[0:3], "/") + src
					} else {
						l := strings.Count(src, "../")
						//需要多减1，因为"http://"或"https://"后面多带一个斜杠
						src = strings.Join(slice[0:sliceLen-l-1], "/") + "/" + strings.TrimLeft(src, "./")
					}
				}

				// 默认记录到数据库中的图片路径
				//save := src
				project := ""
				for _, header := range headers {
					if val, ok := header["project"]; ok {
						project = val
					}
				}
				if project != "" {
					var exist string
					if exist, existImage = imageMap[srcLower]; !existImage {
						tmpFile, err := DownImage(src, headers...)
						if err == nil {
							defer os.Remove(tmpFile) //删除文件
							switch StoreType {
							case StoreLocal:
								src = "/uploads/projects/" + project + "/" + filepath.Base(tmpFile)
								store.ModelStoreLocal.MoveToStore(tmpFile, strings.TrimPrefix(src, "/"))
							case StoreOss:
								src = "projects/" + project + "/" + filepath.Base(tmpFile)
								store.ModelStoreOss.MoveToOss(tmpFile, src, true)
								src = "/" + src
							}
							imageMap[srcLower] = src
						} else {
							beego.Error(err.Error())
						}
					} else {
						src = exist
					}
				}
				selection.SetAttr("src", src)
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

		//排除标签
		excludeSelector = append(excludeSelector, "script", "style")
		for _, sel := range excludeSelector {
			doc.Find(sel).Remove()
		}

		diySelector = strings.TrimSpace(diySelector)

		cont, err = doc.Html()

		if intelligence == 1 { //智能提取
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
				//cont = article.Html + "\n原文：\n> " + urlstr
				cont = article.Html
			case 2: //=>text
				//cont = article.Content + fmt.Sprintf("\n原文：\n> %v", urlstr)
				cont = article.Content
			default: //0 && other=>markdown
				//cont = html2md.Convert(article.Html) + fmt.Sprintf("\n\r\n\r原文:\n> %v", urlstr)
				cont = html2md.Convert(article.Html)
			}
		} else if intelligence == 2 && diySelector != "" { //自定义提取
			if htmlstr, err := doc.Find(diySelector).Html(); err != nil {
				return "", err
			} else {
				switch contType {
				case 1: //=>html
					//cont = htmlstr + "\n\r\n\r> 原文： " + urlstr
					cont = htmlstr
				case 2: //=>text
					//cont = doc.Find(diySelector).Text() + fmt.Sprintf("\n\r\n\r> 原文: %v", urlstr)
					cont = doc.Find(diySelector).Text()
				default: //0 && other=>markdown
					//cont = html2md.Convert(htmlstr) + fmt.Sprintf("\n\r\n\r> 原文: %v", urlstr)
					cont = html2md.Convert(htmlstr)
				}
			}
		} else { //全文

			switch contType {
			case 1: //=>html
				htmlstr, _ := doc.Find("body").Html()
				//cont = htmlstr + "\n\n\n> 原文：" + urlstr
				cont = htmlstr
			case 2: //=>text
				//cont = doc.Find("body").Text() + fmt.Sprintf("\n\r\n\r> 原文: %v", urlstr)
				cont = doc.Find("body").Text()
			default: //0 && other=>markdown
				htmlstr, _ := doc.Find("body").Html()
				//cont = html2md.Convert(htmlstr) + fmt.Sprintf("\n\r\n\r> 原文: %v", urlstr)
				cont = html2md.Convert(htmlstr)
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

// Substr returns the substr from start to length.
func Substr(s string, length int) string {
	bt := []rune(s)
	start := 0
	dot := false

	if start > len(bt) {
		start = start % len(bt)
	}
	var end int
	if (start + length) > (len(bt) - 1) {
		end = len(bt)
	} else {
		end = start + length
		dot = true
	}

	str := string(bt[start:end])
	if dot {
		str = str + "..."
	}
	return str
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
	paramsLen := len(urlParams)
	if paramsLen > 0 {
		if paramsLen%2 > 0 {
			paramsLen = paramsLen - 1
		}
		for i := 0; i < paramsLen; {
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
		htmlPage += fmt.Sprintf(`<li><a class="num" href="`+path+`?page=1%v">1..</a></li><li><a class="num" href="`+path+`?page=%d%v">«</a></li>`, urlSuffix, currentPage-1, urlSuffix)
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
		var pageSlice []int
		indexMin := index - rollPage/2
		indexMax := index + rollPage/2
		if indexMin > 0 && indexMax < pages_len { //切片索引未越界
			pageSlice = pages[indexMin:indexMax]
		} else {
			if indexMin < 0 {
				pageSlice = pages[0:rollPage]
			} else if indexMax > pages_len {
				pageSlice = pages[(pages_len - rollPage):pages_len]
			} else {
				pageSlice = pages[indexMin:indexMax]
			}

		}

		for _, v := range pageSlice {
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

//判断数据是否在map中
func InMap(maps map[int]bool, key int) (ret bool) {
	if _, ok := maps[key]; ok {
		return true
	}
	return
}

// 从HTML中获取text
func GetTextFromHtml(htmlStr string) (txt string) {
	if doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlStr)); err == nil {
		txt = strings.TrimSpace(doc.Find("body").Text())
	}
	return
}

// Git Clone
func GitClone(url, folder string) error {
	os.RemoveAll(folder)
	args := []string{"clone", url, folder}
	return exec.Command("git", args...).Run()
}

type SplitMD struct {
	Identify string
	Cont     string
}

// 处理http响应成败
func HandleResponse(resp *http.Response, err error) error {
	if err == nil {
		defer resp.Body.Close()
		if resp.StatusCode >= 300 || resp.StatusCode < 200 {
			b, _ := ioutil.ReadAll(resp.Body)
			err = errors.New(resp.Status + "；" + string(b))
		}
	}
	return err
}

// 下载图片
func DownImage(src string, headers ...map[string]string) (filename string, err error) {
	var resp *http.Response
	var b []byte
	ext := ".png"
	file := cryptil.Md5Crypt(src)
	filename = "cache/" + file
	srcLower := strings.ToLower(src)
	if strings.HasPrefix(srcLower, "data:image/") && strings.Contains(srcLower, ";base64,") { //base64的图片
		slice := strings.Split(src, ";base64,")
		if len(slice) >= 2 {
			ext := "." + strings.TrimPrefix(slice[0], "data:image/")
			filename = filename + strings.ToLower(ext)
			b, err = base64.StdEncoding.DecodeString(strings.Join(slice[1:], ";base64,"))
			if err != nil {
				return
			}
			err = ioutil.WriteFile(filename, b, os.ModePerm)
		}
	} else { //url链接图片
		resp, err = util.BuildRequest("get", src, src, "", "", true, false, headers...).Response()
		if err != nil {
			return
		}
		defer resp.Body.Close()
		if tmp := strings.TrimPrefix(strings.ToLower(resp.Header.Get("Content-Type")), "image/"); tmp != "" {
			ext = "." + tmp
		} else {
			ext = strings.ToLower(filepath.Ext(srcLower))
		}
		filename = filename + ext
		b, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			return
		}
		err = ioutil.WriteFile(filename, b, os.ModePerm)
	}
	return
}
