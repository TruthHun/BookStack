package models

import (
	"errors"
	"io/ioutil"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"

	"net/http"

	"encoding/json"
	"strconv"

	"fmt"

	"github.com/TruthHun/gotil/util"
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/httplib"
	"github.com/astaxie/beego/orm"
)

// 是否正在创建全量索引
var IsRebuildAllIndex = false

//全文搜索客户端
type ElasticSearchClient struct {
	Host    string        //host
	Index   string        //索引
	Type    string        //type
	On      bool          //是否启用全文搜索
	Timeout time.Duration //超时时间
}

//全文搜索
type ElasticSearchData struct {
	Id       int    `json:"id"`       //文档或书籍id
	BookId   int    `json:"book_id"`  //书籍id。这里的book_id起到的作用是IsBooK的布尔，以及搜索书籍文档时候的过滤
	Title    string `json:"title"`    //文档标题或书籍名称
	Keywords string `json:"keywords"` //文档或书籍关键字
	Content  string `json:"content"`  //文档摘要或书籍文本内容
	Vcnt     int    `json:"vcnt"`     //浏览量
	Private  int    `json:"private"`  //书籍或者文档是否是公开的
}

//统计信息结构
type ElasticSearchCount struct {
	Shards struct {
		Failed     int `json:"failed"`
		Skipped    int `json:"skipped"`
		Successful int `json:"successful"`
		Total      int `json:"total"`
	} `json:"_shards"`
	Count int `json:"count"`
}

//搜索结果结构
type ElasticSearchResult struct {
	Shards struct {
		Failed     int `json:"failed"`
		Skipped    int `json:"skipped"`
		Successful int `json:"successful"`
		Total      int `json:"total"`
	} `json:"_shards"`
	Hits struct {
		Hits []struct {
			ID     string      `json:"_id"`
			Index  string      `json:"_index"`
			Score  interface{} `json:"_score"`
			Source struct {
				Id       int    `json:"id"`
				BookId   int    `json:"book_id"`
				Title    string `json:"title"`
				Keywords string `json:"keywords"`
				Content  string `json:"content"`
				Vcnt     int    `json:"vcnt"`
				Private  int    `json:"private"`
			} `json:"_source"`
			Type string `json:"_type"`
			Sort []int  `json:"sort"`
		} `json:"hits"`
		MaxScore interface{} `json:"max_score"`
		Total    int         `json:"total"`
	} `json:"hits"`
	TimedOut bool `json:"timed_out"`
	Took     int  `json:"took"`
}

// 搜索文档展示结果
type SearchDocResult struct {
}

//创建全文搜索客户端
func NewElasticSearchClient() (client *ElasticSearchClient) {
	client = &ElasticSearchClient{
		Host:    GetOptionValue("ELASTICSEARCH_HOST", "http://localhost:9200/"),
		Index:   "bookstack",
		Type:    "fulltext",
		On:      GetOptionValue("ELASTICSEARCH_ON", "false") == "true",
		Timeout: 10 * time.Second,
	}
	client.Host = strings.TrimRight(client.Host, "/") + "/"
	return
}

// 将HTML转成符合elasticsearch搜索的文本
func (this *ElasticSearchClient) html2Text(htmlStr string) string {
	var tags = []string{
		"</p>", "</div>", "</code>", "</span>", "</pre>", "</blockquote>",
		"</h1>", "</h2>", "</h3>", "</h4>", "</h5>", "</h6>", "</td>", "</th>",
		"</i>", "</b>", "</strong>", "</a>", "</li>",
	}

	for _, tag := range tags {
		htmlStr = strings.Replace(htmlStr, tag, tag+" ", -1)
	}

	htmlStr = strings.Replace(htmlStr, "\n", " ", -1)

	gq, err := goquery.NewDocumentFromReader(strings.NewReader(htmlStr))
	if err != nil {
		return htmlStr
	}
	return gq.Text()
}

//初始化全文搜索客户端，包括检查索引是否存在，mapping设置等
func (this *ElasticSearchClient) Init() (err error) {
	if !this.On { //未开启ElasticSearch，则不初始化
		return
	}
	//检测是否能ping同
	if err = this.ping(); err == nil {
		//检测索引是否存在；索引不存在，则创建索引；如果索引存在，则直接跳过初始化
		if err = this.existIndex(); err != nil {
			//创建索引成功
			if err = this.createIndex(); err == nil {
				//创建mapping
				js := `{
	"properties": {
		"title": {
			"type": "text",
			"analyzer": "ik_max_word",
			"search_analyzer": "ik_smart"
		},
		"keywords": {
			"type": "text",
			"analyzer": "ik_max_word",
			"search_analyzer": "ik_smart"
		},
		"description": {
			"type": "text",
			"analyzer": "ik_max_word",
			"search_analyzer": "ik_smart"
		},
		"vcnt": {
			"type": "integer"
		},
		"is_book": {
			"type": "integer"
		}
	}
}`
				if orm.Debug {
					beego.Debug(" ==== ElasticSearch初始化mapping ==== ")
					beego.Info(js)
					beego.Debug(" ==== ElasticSearch初始化mapping ==== ")
				}
				api := this.Host + this.Index + "/" + this.Type + "/_mapping"
				req := this.post(api)
				if resp, errResp := req.Header("Content-Type", "application/json").Body(js).Response(); errResp != nil {
					err = errResp
				} else {
					if resp.StatusCode >= 300 || resp.StatusCode < 200 {
						err = errors.New(resp.Status)
					}
				}
			}
		}
	}
	return
}

//搜索内容
// 如果书籍id大于0，则表示搜索指定的书籍的文档。否则表示搜索书籍
// 如果不指定书籍id，则只能搜索
func (this *ElasticSearchClient) Search(wd string, p, listRows int, isSearchDoc bool, bookId ...int) (result ElasticSearchResult, err error) {
	wd = strings.Replace(wd, "\"", " ", -1)
	wd = strings.Replace(wd, "\\", " ", -1)
	bid := 0
	if len(bookId) > 0 && bookId[0] > 0 {
		bid = bookId[0]
	}

	var queryBody string
	// 请求体
	if bid > 0 { // 搜索指定书籍的文档，不限私有和公有
		queryBody = `{"query": {"bool": {"filter": [{
				"term": {
					"book_id": {$bookId}
				}
			}],
          "must":{"multi_match" : {
              "query":    "%v", 
              "fields": [ "title", "keywords","content" ] 
            }
          }
		}},"from": %v,"size": %v,"_source":["id"]}`
		queryBody = strings.Replace(queryBody, "{$bookId}", strconv.Itoa(bid), 1)
	} else {
		if isSearchDoc { //搜索公开的文档
			queryBody = `{"query": {"bool": {
			"filter": [
              {"range": {"book_id": {"gt": 0}}},
              {"term": {"private": 0}}
            ],"must":{
          	"multi_match" : {
              "query":    "%v", 
              "fields": [ "title", "keywords","content" ] 
            }}}},"from": %v,"size": %v,"_source":["id"]}`
		} else { //搜索公开的书籍
			queryBody = `{"query": {"bool": {
			"filter": [
            	{"term": {"book_id": 0}},
                {"term": {"private": 0}}
            ],"must":{
          	"multi_match" : {
              "query":    "%v", 
              "fields": [ "title", "keywords","content" ] 
            }
          }}},"from": %v, "size": %v,"_source":["id"]}`
		}
	}

	queryBody = fmt.Sprintf(queryBody, wd, (p-1)*listRows, listRows)
	api := this.Host + this.Index + "/" + this.Type + "/_search"
	if orm.Debug {
		beego.Debug(api)
		beego.Debug(queryBody)
	}
	if resp, errResp := this.post(api).Body(queryBody).Response(); errResp != nil {
		err = errResp
	} else {
		b, _ := ioutil.ReadAll(resp.Body)
		err = json.Unmarshal(b, &result)
	}
	return
}

//重建索引【全量】
//采用批量重建索引的方式进行
//每次操作100条数据
func (this *ElasticSearchClient) RebuildAllIndex() {

	if IsRebuildAllIndex {
		return
	}

	defer func() {
		IsRebuildAllIndex = false
	}()
	IsRebuildAllIndex = true

	pageSize := 1000
	maxPage := int(1e7)

	privateMap := make(map[int]int) //map[book_id]private

	o := orm.NewOrm()
	book := NewBook()
	// 更新书籍
	for page := 1; page < maxPage; page++ {
		var books []Book
		fields := []string{"book_id", "book_name", "label", "description", "privately_owned"}
		o.QueryTable(book).Limit(pageSize).Offset((page-1)*pageSize).All(&books, fields...)
		if len(books) > 0 {
			var data []ElasticSearchData
			for _, item := range books {
				data = append(data, ElasticSearchData{
					Id:       item.BookId,
					Title:    item.BookName,
					Keywords: item.Label,
					Content:  item.Description,
					BookId:   0, //注意：这里必须设置为0
					Private:  item.PrivatelyOwned,
					Vcnt:     item.Vcnt,
				})
				privateMap[item.BookId] = item.PrivatelyOwned
			}
			if err := this.BuildIndexByBuck(data); err != nil {
				beego.Error(err.Error())
			}
		} else {
			page = maxPage
		}
	}

	// 文档内容可能比较大，每次更新20个文档
	pageSize = 20
	doc := NewDocument()
	for page := 1; page < maxPage; page++ {
		var docs []Document
		fields := []string{"document_id", "book_id", "document_name", "release", "vcnt"}
		o.QueryTable(doc).Limit(pageSize).Offset((page-1)*pageSize).All(&docs, fields...)
		if len(docs) > 0 {
			var data []ElasticSearchData

			for _, item := range docs {
				private := 1
				if v, ok := privateMap[item.BookId]; ok {
					private = v
				}

				d := ElasticSearchData{
					Id:       item.DocumentId,
					Title:    item.DocumentName,
					Keywords: "",
					Content:  this.html2Text(item.Release),
					//Content: item.Release,
					BookId:  item.BookId,
					Private: private,
					Vcnt:    item.Vcnt,
				}
				data = append(data, d)
				//if err := this.BuildIndex(d); err != nil {
				//	beego.Error(err.Error())
				//}
			}
			if err := this.BuildIndexByBuck(data); err != nil {
				beego.Error(err.Error())
			}
		} else {
			page = maxPage
		}
	}
}

//通过bulk，批量创建/更新索引
func (this *ElasticSearchClient) BuildIndexByBuck(data []ElasticSearchData) (err error) {
	now := time.Now()
	var bodySlice []string
	if len(data) > 0 {
		var _id string
		for _, item := range data {
			if item.BookId > 0 { //书籍的id大于0，表示这个数据是文档的数据，否则是书籍的数据
				_id = fmt.Sprintf("doc_%v", item.Id)
			} else {
				_id = fmt.Sprintf("book_%v", item.Id)
			}
			action := fmt.Sprintf(`{"index":{"_index":"%v","_type":"%v","_id":"%v"}}`, this.Index, this.Type, _id)
			bodySlice = append(bodySlice, action)
			bodySlice = append(bodySlice, util.InterfaceToJson(item))
		}
		api := this.Host + "_bulk"
		body := strings.Join(bodySlice, "\n") + "\n"
		if orm.Debug {
			beego.Info("批量更新索引请求体")
			beego.Info(body)
		}
		if resp, errResp := this.post(api).Body(body).Response(); errResp != nil {
			err = errResp
		} else {
			if resp.StatusCode >= http.StatusMultipleChoices || resp.StatusCode < http.StatusOK {
				b, _ := ioutil.ReadAll(resp.Body)
				err = errors.New(resp.Status + "；" + string(b))
			}
		}
	}
	d := time.Since(now)
	if d > time.Duration(this.Timeout) {
		// 生成索引时长过长，休眠一小段时间
		time.Sleep(time.Duration(this.Timeout/2) * time.Second)
	}
	return
}

//创建索引
func (this *ElasticSearchClient) BuildIndex(es ElasticSearchData) (err error) {
	var (
		js   []byte
		resp *http.Response
	)
	if !this.On {
		return
	}
	if orm.Debug {
		beego.Info("创建索引--------start--------")
		fmt.Printf("内容：%+v\n", es)
		beego.Info("创建索引-------- end --------")
	}

	var _id string

	es.Content = this.html2Text(es.Content)

	if es.BookId > 0 {
		_id = fmt.Sprintf("doc_%v", es.Id)
	} else {
		_id = fmt.Sprintf("book_%v", es.Id)
	}
	api := this.Host + this.Index + "/" + this.Type + "/" + _id
	if js, err = json.Marshal(es); err == nil {
		if resp, err = this.post(api).Body(js).Response(); err == nil {
			if resp.StatusCode >= 300 || resp.StatusCode < 200 {
				b, _ := ioutil.ReadAll(resp.Body)
				err = errors.New("生成索引失败：" + resp.Status + "；" + string(b))
			}
		}
	}
	return
}

//查询索引量
//@return           count           统计数据
//@return           err             错误
func (this *ElasticSearchClient) Count() (count int, err error) {
	if !this.On {
		err = errors.New("未启用ElasticSearch")
		return
	}
	api := this.Host + this.Index + "/" + this.Type + "/_count"
	if resp, errResp := this.get(api).Response(); errResp != nil {
		err = errResp
	} else {
		b, _ := ioutil.ReadAll(resp.Body)
		body := string(b)
		if resp.StatusCode >= http.StatusMultipleChoices || resp.StatusCode < http.StatusOK {
			err = errors.New(resp.Status + "；" + body)
		} else {
			var cnt ElasticSearchCount
			if err = json.Unmarshal(b, &cnt); err == nil {
				count = cnt.Count
			}
		}
	}
	return
}

//删除索引
//@param            id          索引id
//@return           err         错误
func (this *ElasticSearchClient) DeleteIndex(id int) (err error) {
	api := this.Host + this.Index + "/" + this.Type + "/" + strconv.Itoa(id)
	if resp, errResp := this.delete(api).Response(); errResp != nil {
		err = errResp
	} else {
		if resp.StatusCode >= 300 || resp.StatusCode < 200 {
			b, _ := ioutil.ReadAll(resp.Body)
			err = errors.New("删除索引失败：" + resp.Status + "；" + string(b))
		}
	}
	return
}

//检验es服务能否连通
func (this *ElasticSearchClient) ping() error {
	if resp, err := this.get(this.Host).Response(); err != nil {
		return err
	} else {
		if resp.StatusCode >= 300 || resp.StatusCode < 200 {
			body, _ := ioutil.ReadAll(resp.Body)
			err = errors.New(resp.Status + "；" + string(body))
		}
	}
	return nil
}

//查询索引是否存在
//@return			err				nil表示索引存在，否则表示不存在
func (this *ElasticSearchClient) existIndex() (err error) {
	var resp *http.Response
	api := this.Host + this.Index
	if resp, err = this.get(api).Response(); err == nil {
		if resp.StatusCode >= 300 || resp.StatusCode < 200 {
			b, _ := ioutil.ReadAll(resp.Body)
			err = errors.New(resp.Status + "：" + string(b))
		}
	}
	return
}

//创建索引
//@return           err             创建索引
func (this *ElasticSearchClient) createIndex() (err error) {
	var resp *http.Response
	api := this.Host + this.Index
	if resp, err = this.put(api).Response(); err == nil {
		if resp.StatusCode >= 300 || resp.StatusCode < 200 {
			b, _ := ioutil.ReadAll(resp.Body)
			err = errors.New(resp.Status + "：" + string(b))
		}
	}
	return
}

//put请求
func (this *ElasticSearchClient) put(api string) (req *httplib.BeegoHTTPRequest) {
	return httplib.Put(api).Header("Content-Type", "application/json").SetTimeout(this.Timeout, this.Timeout)
}

//post请求
func (this *ElasticSearchClient) post(api string) (req *httplib.BeegoHTTPRequest) {
	return httplib.Post(api).Header("Content-Type", "application/json").SetTimeout(this.Timeout, this.Timeout)
}

//delete请求
func (this *ElasticSearchClient) delete(api string) (req *httplib.BeegoHTTPRequest) {
	return httplib.Delete(api).Header("Content-Type", "application/json").SetTimeout(this.Timeout, this.Timeout)
}

//get请求
func (this *ElasticSearchClient) get(api string) (req *httplib.BeegoHTTPRequest) {
	return httplib.Get(api).Header("Content-Type", "application/json").SetTimeout(this.Timeout, this.Timeout)
}
