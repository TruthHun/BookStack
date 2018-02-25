package utils

import (
	"fmt"
	"os"

	"strconv"
	"strings"

	"github.com/TruthHun/BookStack/conf"
	"github.com/TruthHun/html2article"
	"github.com/alexcesaro/mail/mailer"
	//"github.com/lunny/html2md"

	"net/mail"

	"io/ioutil"
	"path/filepath"

	"os/exec"

	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/TruthHun/gotil/filetil"
	"github.com/TruthHun/gotil/util"
	"github.com/astaxie/beego"
	"github.com/huichen/sego"

	"github.com/TruthHun/html2md"
	"github.com/russross/blackfriday"
	"sync"
)

//分词器
var (
	Segmenter sego.Segmenter
	ReleaseMaps =make(map[int]bool)//是否正在发布内容，map[book_id]bool
	ReleaseMapsLock sync.RWMutex
	BasePath,_=filepath.Abs(filepath.Dir(os.Args[0]))
)

func init() {
	//加载分词字典
	go func() {
		Segmenter.LoadDictionary(BasePath+"/dictionary/dictionary.txt")
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

func SortBySummary(htmlstr string, pid int) {
	//在markdown头部加上<bookstack></bookstack>或者<bookstack/>，即解析markdown中的ul>li>a链接作为目录
	doc, _ := goquery.NewDocumentFromReader(strings.NewReader(htmlstr))
	doc.Find("body>ul>li>a").Each(func(i int, selection *goquery.Selection) {
		nid, _ := selection.Attr("id")
		fmt.Println("====", pid, selection.Text(), nid)
		if str, err := selection.Siblings().Html(); err == nil {
			id, _ := selection.Attr("id")
			idNum, _ := strconv.Atoi(id)
			SortBySummary("<ul>"+str+"</ul>", idNum)
		}
	})
}

//查找替换md(这个重要)
func ReplaceToAbs(folder string) {
	files, _ := filetil.ScanFiles(folder)
	for _, file := range files {
		if strings.HasSuffix(file.Path, ".md") {
			mdb, _ := ioutil.ReadFile(file.Path)
			mdCont := string(mdb)
			basePath := filepath.Dir(file.Path)
			basePath = strings.Trim(strings.Replace(basePath, "\\", "/", -1), "/")
			basePathSlice := strings.Split(basePath, "/")
			l := len(basePathSlice)
			b, _ := ioutil.ReadFile(file.Path)
			output := blackfriday.MarkdownCommon(b)
			doc, _ := goquery.NewDocumentFromReader(strings.NewReader(string(output)))
			doc.Find("img").Each(func(i int, selection *goquery.Selection) {
				if src, ok := selection.Attr("src"); ok && !strings.HasPrefix(strings.ToLower(src), "http") && strings.HasPrefix(src, "./") {
					if cnt := strings.Count(src, "../"); cnt < l {
						newSrc := strings.Join(basePathSlice[0:l-cnt], "/") + "/" + strings.TrimLeft(src, "./")
						mdCont = strings.Replace(mdCont, src, newSrc, -1)
						fmt.Println(file.Path, src, newSrc)
					}
				}
			})
			//要注意判断有锚点的情况
			doc.Find("a").Each(func(i int, selection *goquery.Selection) {
				if href, ok := selection.Attr("href"); ok && !strings.HasPrefix(strings.ToLower(href), "http") && strings.HasPrefix(href, "./") {
					if cnt := strings.Count(href, "../"); cnt < l {
						newHref := ""
						if cnt == 0 && strings.HasPrefix(href, "#") { //以#开头，链接则是锚点链接，
							newHref = file.Path + href
						} else {
							newHref = strings.Join(basePathSlice[0:l-cnt], "/") + "/" + strings.TrimLeft(href, "./")
						}
						fmt.Println(file.Path, href, newHref)
						mdCont = strings.Replace(mdCont, href, newHref, -1)
					}
				}
			})
			ioutil.WriteFile(file.Path, []byte(mdCont), os.ModePerm)
		}
	}

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
					if href, ok := selection.Attr("href"); ok && (!strings.HasPrefix(strings.ToLower(href), "http://") && !strings.HasPrefix(strings.ToLower(href), "https://")&& !strings.HasPrefix(strings.ToLower(href), "#")) {
						if strings.HasPrefix(href,"/"){
							selection.SetAttr("href", strings.Join(slice[0:3], "/")+href)
						}else{
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
						if strings.HasPrefix(src,"/"){//以斜杠开头
						//TODO: 域名+src
							selection.SetAttr("src", strings.Join(slice[0:3], "/")+src)
						}else{
							l := strings.Count(src, "../")
							//需要多减1，因为"http://"或"https://"后面多带一个斜杠
							selection.SetAttr("src", strings.Join(slice[0:sliceLen-l-1], "/")+"/"+strings.TrimLeft(src, "./"))
						}
					}
				})

				//h1-h6标题中不要存在链接或者图片，所以提取文本
				Hs:=[]string{"h1","h2","h3","h4","h5","h6"}
				for _,tag:=range Hs{
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
