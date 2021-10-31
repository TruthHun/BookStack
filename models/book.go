package models

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"strings"

	"fmt"

	"strconv"

	"github.com/PuerkitoBio/goquery"
	"github.com/TruthHun/BookStack/conf"
	"github.com/TruthHun/BookStack/models/store"
	"github.com/TruthHun/BookStack/utils"
	"github.com/TruthHun/BookStack/utils/html2md"
	"github.com/TruthHun/gotil/filetil"
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/logs"
	"github.com/astaxie/beego/orm"
)

//定义书籍排序类型
type BookOrder string

const (
	OrderRecommend       BookOrder = "recommend"
	OrderPopular         BookOrder = "popular"          //热门
	OrderLatest          BookOrder = "latest"           //最新
	OrderNew             BookOrder = "new"              //最新
	OrderScore           BookOrder = "score"            //评分排序
	OrderComment         BookOrder = "comment"          //评论排序
	OrderStar            BookOrder = "star"             //收藏排序
	OrderView            BookOrder = "vcnt"             //浏览排序
	OrderLatestRecommend BookOrder = "latest-recommend" //最新推荐
)

type BookNav struct {
	Sort   int    `json:"sort"`
	Icon   string `json:"icon"`
	Color  string `json:"color"`
	Name   string `json:"name"`
	URL    string `json:"url"`
	Target string `json:"target"`
}

// Book struct .
type Book struct {
	BookId            int       `orm:"pk;auto;unique;column(book_id)" json:"book_id"`
	ParentId          int       `orm:"column(parent_id);default(0)"`                      // 书籍的父级id，一般用于拷贝书籍项目生成新项目的时候
	BookName          string    `orm:"column(book_name);size(500)" json:"book_name"`      // BookName 书籍名称.
	Identify          string    `orm:"column(identify);size(100);unique" json:"identify"` // Identify 书籍唯一标识.
	OrderIndex        int       `orm:"column(order_index);type(int);default(0);index" json:"order_index"`
	Pin               int       `orm:"column(pin);type(int);default(0)" json:"pin"`       // pin值，用于首页固定显示
	Description       string    `orm:"column(description);size(2000)" json:"description"` // Description 书籍描述.
	Label             string    `orm:"column(label);size(500)" json:"label"`
	PrivatelyOwned    int       `orm:"column(privately_owned);type(int);default(0)" json:"privately_owned"` // PrivatelyOwned 书籍私有： 0 公开/ 1 私有
	PrivateToken      string    `orm:"column(private_token);size(500);null" json:"private_token"`           // 当书籍是私有时的访问Token.
	Status            int       `orm:"column(status);type(int);default(0)" json:"status"`                   //状态：0 正常/1 已删除
	Editor            string    `orm:"column(editor);size(50)" json:"editor"`                               //默认的编辑器.
	DocCount          int       `orm:"column(doc_count);type(int)" json:"doc_count"`                        // DocCount 包含文档数量.
	CommentStatus     string    `orm:"column(comment_status);size(20);default(open)" json:"comment_status"` // CommentStatus 评论设置的状态:open 为允许所有人评论，closed 为不允许评论, group_only 仅允许参与者评论 ,registered_only 仅允许注册者评论.
	CommentCount      int       `orm:"column(comment_count);type(int)" json:"comment_count"`
	Cover             string    `orm:"column(cover);size(1000)" json:"cover"`                              //封面地址
	Theme             string    `orm:"column(theme);size(255);default(default)" json:"theme"`              //主题风格
	CreateTime        time.Time `orm:"type(datetime);column(create_time);auto_now_add" json:"create_time"` // CreateTime 创建时间 .
	MemberId          int       `orm:"column(member_id);size(100);index" json:"member_id"`
	ModifyTime        time.Time `orm:"type(datetime);column(modify_time);auto_now" json:"modify_time"`
	ReleaseTime       time.Time `orm:"type(datetime);column(release_time);" json:"release_time"`   //书籍发布时间，每次发布都更新一次，如果文档更新时间小于发布时间，则文档不再执行发布
	GenerateTime      time.Time `orm:"type(datetime);column(generate_time);" json:"generate_time"` //电子书生成时间
	LastClickGenerate time.Time `orm:"type(datetime);column(last_click_generate)" json:"-"`        //上次点击生成电子书的时间，用于显示频繁点击浪费服务器硬件资源的情况
	Version           int64     `orm:"type(bigint);column(version);default(0)" json:"version"`
	Vcnt              int       `orm:"column(vcnt);default(0)" json:"vcnt"`    // 书籍被阅读次数
	Star              int       `orm:"column(star);default(0)" json:"star"`    // 书籍被收藏次数
	Score             int       `orm:"column(score);default(40)" json:"score"` // 书籍评分，默认40，即4.0星
	CntScore          int       // 评分人数
	CntComment        int       // 评论人数
	Author            string    `orm:"size(50)"`            //原作者，即来源
	AuthorURL         string    `orm:"column(author_url)"`  //原作者链接，即来源链接
	AdTitle           string    `orm:"default()"`           // 文字广告标题
	AdLink            string    `orm:"default();size(512)"` // 文字广告链接
	Lang              string    `orm:"size(10);index;default(zh)"`
	NavJSON           string    `orm:"size(8192);default();column(nav_json)"` // 导航栏扩展
	Navs              []BookNav `orm:"-" json:"navs"`
}

type BookNavs []BookNav

func (s BookNavs) Len() int { return len(s) }

func (s BookNavs) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

func (s BookNavs) Less(i, j int) bool { return s[i].Sort < s[j].Sort }

// TableName 获取对应数据库表名.
func (m *Book) TableName() string {
	return "books"
}

// TableEngine 获取数据使用的引擎.
func (m *Book) TableEngine() string {
	return "INNODB"
}
func (m *Book) TableNameWithPrefix() string {
	return conf.GetDatabasePrefix() + m.TableName()
}

func NewBook() *Book {
	return &Book{}
}

// minRole 最小的角色权限
//conf.BookFounder
//conf.BookAdmin
//conf.BookEditor
//conf.BookObserver
func (m *Book) HasProjectAccess(identify string, memberId int, minRole int) bool {
	book := NewBook()
	rel := NewRelationship()
	o := orm.NewOrm()
	o.QueryTable(book).Filter("identify", identify).One(book, "book_id")
	if book.BookId <= 0 {
		return false
	}
	o.QueryTable(rel).Filter("book_id", book.BookId).Filter("member_id", memberId).One(rel)
	if rel.RelationshipId <= 0 {
		return false
	}
	return rel.RoleId <= minRole
}

func (m *Book) Sorted(limit int, orderField string) (books []Book) {
	o := orm.NewOrm()
	fields := []string{"book_id", "book_name", "identify", "cover", "vcnt", "star", "cnt_comment"}
	o.QueryTable(m).Filter("order_index__gte", 0).Filter("privately_owned", 0).OrderBy("-"+orderField).Limit(limit).All(&books, fields...)
	return
}

func (m *Book) Insert() (err error) {
	o := orm.NewOrm()
	if _, err = o.Insert(m); err != nil {
		return
	}

	if m.Label != "" {
		NewLabel().InsertOrUpdateMulti(m.Label)
	}

	relationship := NewRelationship()
	relationship.BookId = m.BookId
	relationship.RoleId = 0
	relationship.MemberId = m.MemberId

	if err = relationship.Insert(); err != nil {
		logs.Error("插入书籍与用户关联 => ", err)
		return err
	}

	document := NewDocument()
	document.BookId = m.BookId
	document.DocumentName = "空白文档"
	document.Identify = "blank"
	document.MemberId = m.MemberId

	var id int64
	if id, err = document.InsertOrUpdate(); err == nil {
		var ds = DocumentStore{
			DocumentId: int(id),
			Markdown:   "[TOC]\n\r\n\r", //默认内容
		}
		err = new(DocumentStore).InsertOrUpdate(ds)
	}
	return err
}

func (m *Book) Find(id int, cols ...string) (book *Book, err error) {
	if id <= 0 {
		return
	}
	o := orm.NewOrm()
	err = o.QueryTable(m.TableNameWithPrefix()).Filter("book_id", id).One(m, cols...)
	return m, err
}

func (m *Book) Update(cols ...string) (err error) {
	o := orm.NewOrm()

	temp := NewBook()
	temp.BookId = m.BookId

	if err = o.Read(temp); err != nil {
		return err
	}

	if (m.Label + temp.Label) != "" {
		go NewLabel().InsertOrUpdateMulti(m.Label + "," + temp.Label)
	}

	_, err = o.Update(m, cols...)
	return err
}

//根据指定字段查询结果集.
func (m *Book) FindByField(field string, value interface{}) (books []*Book, err error) {
	o := orm.NewOrm()
	_, err = o.QueryTable(m.TableNameWithPrefix()).Filter(field, value).All(&books)
	return
}

//根据指定字段查询一个结果.
func (m *Book) FindByFieldFirst(field string, value interface{}) (book *Book, err error) {
	o := orm.NewOrm()
	err = o.QueryTable(m.TableNameWithPrefix()).Filter(field, value).One(m)
	return m, err
}

func (m *Book) FindByIdentify(identify string, cols ...string) (book *Book, err error) {
	o := orm.NewOrm()
	book = &Book{}
	err = o.QueryTable(m.TableNameWithPrefix()).Filter("identify", identify).One(book, cols...)
	return
}

//分页查询指定用户的书籍
//按照最新的进行排序
func (m *Book) FindToPager(pageIndex, pageSize, memberId int, wd string, PrivatelyOwned ...int) (books []*BookResult, totalCount int, err error) {
	var args = []interface{}{memberId}
	relationship := NewRelationship()
	o := orm.NewOrm()
	sqlCount := "SELECT COUNT(book.book_id) AS total_count FROM " + m.TableNameWithPrefix() + " AS book LEFT JOIN " +
		relationship.TableNameWithPrefix() + " AS rel ON book.book_id=rel.book_id AND rel.member_id = ? WHERE rel.relationship_id > 0 "
	if len(PrivatelyOwned) > 0 {
		sqlCount = sqlCount + " and book.privately_owned=" + strconv.Itoa(PrivatelyOwned[0])
	}

	if wd = strings.TrimSpace(wd); wd != "" {
		wd = "%" + wd + "%"
		sqlCount = sqlCount + " and (book.book_name like ? or book.description like ?)"
		args = append(args, wd, wd)
	}

	err = o.Raw(sqlCount, args...).QueryRow(&totalCount)
	if err != nil {
		beego.Error(err)
		return
	}

	offset := (pageIndex - 1) * pageSize
	sqlQuery := "SELECT book.*,rel.member_id,rel.role_id,m.account as create_name FROM " + m.TableNameWithPrefix() + " AS book" +
		" LEFT JOIN " + relationship.TableNameWithPrefix() + " AS rel ON book.book_id=rel.book_id AND rel.member_id = ? " +
		" LEFT JOIN " + NewMember().TableNameWithPrefix() + " AS m ON rel.member_id=m.member_id " +
		" WHERE rel.relationship_id > 0 %v ORDER BY book.book_id DESC LIMIT " + fmt.Sprintf("%d,%d", offset, pageSize)

	cond := []string{}
	if wd != "" { // 不需要处理 wd 和 args，因为在上面已处理过
		cond = append(cond, " and (book.book_name like ? or book.description like ?)")
	}

	if len(PrivatelyOwned) > 0 {
		cond = append(cond, " and book.privately_owned="+strconv.Itoa(PrivatelyOwned[0]))
	}
	sqlQuery = fmt.Sprintf(sqlQuery, strings.Join(cond, " "))

	_, err = o.Raw(sqlQuery, args...).QueryRows(&books)
	if err != nil {
		beego.Error("分页查询书籍列表 => ", err, sqlQuery)
		return
	}

	if err == nil && len(books) > 0 {
		sql := "SELECT m.account,doc.modify_time FROM md_documents AS doc LEFT JOIN md_members AS m ON doc.modify_at=m.member_id WHERE book_id = ? ORDER BY doc.modify_time DESC LIMIT 1 "

		for index, book := range books {
			var text struct {
				Account    string
				ModifyTime time.Time
			}

			err = o.Raw(sql, book.BookId).QueryRow(&text)
			if err == nil {
				books[index].LastModifyText = text.Account + " 于 " + text.ModifyTime.Format("2006-01-02 15:04:05")
			}

			if book.RoleId == conf.BookFounder {
				book.RoleName = "创始人"
			} else if book.RoleId == conf.BookAdmin {
				book.RoleName = "管理员"
			} else if book.RoleId == conf.BookEditor {
				book.RoleName = "编辑者"
			} else if book.RoleId == conf.BookObserver {
				book.RoleName = "观察者"
			}
		}
	}
	return
}

// 彻底删除书籍.
func (m *Book) ThoroughDeleteBook(id int) (err error) {
	if id <= 0 {
		return ErrInvalidParameter
	}

	o := orm.NewOrm()

	m.BookId = id
	if err = o.Read(m); err != nil {
		return err
	}

	var (
		docs  []Document
		docId []string
	)

	o.QueryTable(new(Document)).Filter("book_id", id).Limit(10000).All(&docs, "document_id")
	if len(docs) > 0 {
		for _, doc := range docs {
			docId = append(docId, strconv.Itoa(doc.DocumentId))
		}
	}

	o.Begin()

	//删除md_document_store中的文档
	if len(docId) > 0 {
		sql1 := fmt.Sprintf("delete from md_document_store where document_id in(%v)", strings.Join(docId, ","))
		if _, err1 := o.Raw(sql1).Exec(); err1 != nil {
			o.Rollback()
			return err1
		}
	}

	sql2 := "DELETE FROM " + NewDocument().TableNameWithPrefix() + " WHERE book_id = ?"
	_, err = o.Raw(sql2, m.BookId).Exec()
	if err != nil {
		o.Rollback()
		return err
	}
	sql3 := "DELETE FROM " + m.TableNameWithPrefix() + " WHERE book_id = ?"

	_, err = o.Raw(sql3, m.BookId).Exec()
	if err != nil {
		o.Rollback()
		return err
	}

	sql4 := "DELETE FROM " + NewRelationship().TableNameWithPrefix() + " WHERE book_id = ?"
	_, err = o.Raw(sql4, m.BookId).Exec()

	if err != nil {
		o.Rollback()
		return err
	}

	if m.Label != "" {
		NewLabel().InsertOrUpdateMulti(m.Label)
	}

	if err = o.Commit(); err != nil {
		return err
	}
	//删除oss中书籍对应的文件夹
	switch utils.StoreType {
	case utils.StoreLocal: //删除本地存储，记得加上uploads
		m.Cover = strings.ReplaceAll(m.Cover, "\\", "/")
		if m.Cover != beego.AppConfig.DefaultString("cover", "/static/images/book.png") {
			os.Remove(strings.TrimLeft(m.Cover, "/ ")) //删除封面
		}
		go store.ModelStoreLocal.DelFromFolder("uploads/projects/" + m.Identify)
	case utils.StoreOss:
		go store.ModelStoreOss.DelOssFolder("projects/" + m.Identify)
	}

	// 删除历史记录
	go func() {
		history := NewDocumentHistory()
		for _, id := range docId {
			idInt, _ := strconv.Atoi(id)
			history.DeleteByDocumentId(idInt)
		}
	}()

	return
}

//首页数据
//完善根据分类查询数据
//orderType:排序条件，可选值：recommend(推荐)、latest（）
func (m *Book) HomeData(pageIndex, pageSize int, orderType BookOrder, lang string, cid int, fields ...string) (books []Book, totalCount int, err error) {
	if cid > 0 { //针对cid>0
		return m.homeData(pageIndex, pageSize, orderType, lang, cid, fields...)
	}
	o := orm.NewOrm()
	order := "pin desc" //排序
	condStr := ""       //查询条件
	cond := []string{"privately_owned=0", "order_index>=0"}
	if len(fields) == 0 {
		fields = append(fields, "book_id", "book_name", "identify", "cover", "order_index", "pin")
	} else {
		fields = append(fields, "pin")
	}
	switch orderType {
	case OrderRecommend: //推荐
		cond = append(cond, "order_index>0")
		order = "pin desc,order_index desc"
	case OrderLatestRecommend: //最新推荐
		cond = append(cond, "order_index>0")
		order = "release_time desc"
	case OrderPopular: //受欢迎
		order = "pin desc,star desc,vcnt desc"
	case OrderLatest, OrderNew: //最新发布
		order = "pin desc,release_time desc"
	case OrderScore: //评分
		order = "pin desc,score desc"
	case OrderComment: //评论
		order = "pin desc,cnt_comment desc"
	case OrderStar: //收藏
		order = "pin desc,star desc"
	case OrderView: //收藏
		order = "pin desc,vcnt desc"
	}
	if len(cond) > 0 {
		condStr = " where " + strings.Join(cond, " and ")
	}

	lang = strings.ToLower(lang)
	switch lang {
	case "zh", "en", "other":
	default:
		lang = ""
	}
	if strings.TrimSpace(lang) != "" {
		condStr = condStr + " and `lang` = '" + lang + "'"
	}
	sqlFmt := "select %v from md_books " + condStr
	fieldStr := strings.Join(fields, ",")
	sql := fmt.Sprintf(sqlFmt, fieldStr) + " order by " + order + fmt.Sprintf(" limit %v offset %v", pageSize, (pageIndex-1)*pageSize)
	sqlCount := fmt.Sprintf(sqlFmt, "count(book_id) cnt")
	var params []orm.Params
	if _, err := o.Raw(sqlCount).Values(&params); err == nil {
		if len(params) > 0 {
			totalCount, _ = strconv.Atoi(params[0]["cnt"].(string))
		}
	}
	if totalCount > 0 {
		_, err = o.Raw(sql).QueryRows(&books)
	}
	return
}

//针对cid大于0
func (m *Book) homeData(pageIndex, pageSize int, orderType BookOrder, lang string, cid int, fields ...string) (books []Book, totalCount int, err error) {
	o := orm.NewOrm()
	order := ""   //排序
	condStr := "" //查询条件
	cond := []string{"b.privately_owned=0", "b.order_index>=0"}
	if len(fields) == 0 {
		fields = append(fields, "book_id", "book_name", "identify", "cover", "order_index")
	}
	switch orderType {
	case OrderRecommend: //推荐
		cond = append(cond, "b.order_index>0")
		order = "b.order_index desc"
	case OrderPopular: //受欢迎
		order = "b.star desc,b.vcnt desc"
	case OrderLatest, OrderNew: //最新发布
		order = "b.release_time desc"
	case OrderScore: //评分
		order = "b.score desc"
	case OrderComment: //评论
		order = "b.cnt_comment desc"
	case OrderStar: //收藏
		order = "b.star desc"
	case OrderView: //收藏
		order = "b.vcnt desc"
	}
	if cid > 0 {
		cond = append(cond, "c.category_id="+strconv.Itoa(cid))
	}
	if len(cond) > 0 {
		condStr = " where " + strings.Join(cond, " and ")
	}
	lang = strings.ToLower(lang)
	switch lang {
	case "zh", "en", "other":
	default:
		lang = ""
	}
	if strings.TrimSpace(lang) != "" {
		condStr = condStr + " and `lang` = '" + lang + "'"
	}
	sqlFmt := "select %v from md_books b left join md_book_category c on b.book_id=c.book_id" + condStr
	fieldStr := "b." + strings.Join(fields, ",b.")
	sql := fmt.Sprintf(sqlFmt, fieldStr) + " order by " + order + fmt.Sprintf(" limit %v offset %v", pageSize, (pageIndex-1)*pageSize)
	sqlCount := fmt.Sprintf(sqlFmt, "count(*) cnt")
	var params []orm.Params
	if _, err := o.Raw(sqlCount).Values(&params); err == nil {
		if len(params) > 0 {
			totalCount, _ = strconv.Atoi(params[0]["cnt"].(string))
		}
	}
	_, err = o.Raw(sql).QueryRows(&books)
	return
}

//分页查找系统首页数据.
func (m *Book) FindForHomeToPager(pageIndex, pageSize, member_id int, orderType string) (books []*BookResult, totalCount int, err error) {
	o := orm.NewOrm()

	offset := (pageIndex - 1) * pageSize
	//如果是登录用户
	if member_id > 0 {
		sql1 := "SELECT COUNT(*) FROM md_books AS book LEFT JOIN md_relationship AS rel ON rel.book_id = book.book_id AND rel.member_id = ? WHERE relationship_id > 0 OR book.privately_owned = 0"
		err = o.Raw(sql1, member_id).QueryRow(&totalCount)
		if err != nil {
			return
		}

		sql2 := `SELECT book.*,rel1.*,member.account AS create_name FROM md_books AS book
			LEFT JOIN md_relationship AS rel ON rel.book_id = book.book_id AND rel.member_id = ?
			LEFT JOIN md_relationship AS rel1 ON rel1.book_id = book.book_id AND rel1.role_id = 0
			LEFT JOIN md_members AS member ON rel1.member_id = member.member_id
			WHERE rel.relationship_id > 0 OR book.privately_owned = 0 ORDER BY order_index DESC ,book.book_id DESC LIMIT ?,?`

		_, err = o.Raw(sql2, member_id, offset, pageSize).QueryRows(&books)
		return
	}

	count, errCount := o.QueryTable(m.TableNameWithPrefix()).Filter("privately_owned", 0).Count()
	if errCount != nil {
		err = errCount
		return
	}
	totalCount = int(count)

	sql := `SELECT book.*,rel.*,member.account AS create_name FROM md_books AS book
			LEFT JOIN md_relationship AS rel ON rel.book_id = book.book_id AND rel.role_id = 0
			LEFT JOIN md_members AS member ON rel.member_id = member.member_id
			WHERE book.privately_owned = 0 ORDER BY order_index DESC ,book.book_id DESC LIMIT ?,?`

	_, err = o.Raw(sql, offset, pageSize).QueryRows(&books)
	return
}

//分页全局搜索.
func (m *Book) FindForLabelToPager(keyword string, pageIndex, pageSize, memberId int) (books []*BookResult, totalCount int, err error) {
	o := orm.NewOrm()

	keyword = "%" + keyword + "%"
	offset := (pageIndex - 1) * pageSize
	//如果是登录用户
	if memberId > 0 {
		sql1 := "SELECT COUNT(*) FROM md_books AS book LEFT JOIN md_relationship AS rel ON rel.book_id = book.book_id AND rel.member_id = ? WHERE (relationship_id > 0 OR book.privately_owned = 0) AND (book.label LIKE ? or book.book_name like ?) limit 1"
		if err = o.Raw(sql1, memberId, keyword, keyword).QueryRow(&totalCount); err != nil {
			return
		}

		sql2 := `SELECT book.*,rel1.*,member.account AS create_name FROM md_books AS book
			LEFT JOIN md_relationship AS rel ON rel.book_id = book.book_id AND rel.member_id = ?
			LEFT JOIN md_relationship AS rel1 ON rel1.book_id = book.book_id AND rel1.role_id = 0
			LEFT JOIN md_members AS member ON rel1.member_id = member.member_id
			WHERE (rel.relationship_id > 0 OR book.privately_owned = 0) AND  (book.label LIKE ? or book.book_name like ?) ORDER BY order_index DESC ,book.book_id DESC LIMIT ?,?`

		_, err = o.Raw(sql2, memberId, keyword, keyword, offset, pageSize).QueryRows(&books)
		return
	}

	sql1 := "select COUNT(*) from md_books where privately_owned=0 and (label LIKE ? or book_name like ?) limit 1"
	if err = o.Raw(sql1, keyword, keyword).QueryRow(&totalCount); err != nil {
		return
	}

	sql := `SELECT book.*,rel.*,member.account AS create_name FROM md_books AS book
			LEFT JOIN md_relationship AS rel ON rel.book_id = book.book_id AND rel.role_id = 0
			LEFT JOIN md_members AS member ON rel.member_id = member.member_id
			WHERE book.privately_owned = 0 AND (book.label LIKE ? or book.book_name LIKE ?) ORDER BY order_index DESC ,book.book_id DESC LIMIT ?,?`

	_, err = o.Raw(sql, keyword, keyword, offset, pageSize).QueryRows(&books)
	return
}

func (book *Book) ToBookResult() (m *BookResult) {
	m = &BookResult{}
	m.BookId = book.BookId
	m.BookName = book.BookName
	m.Identify = book.Identify
	m.OrderIndex = book.OrderIndex
	m.Description = strings.Replace(book.Description, "\r\n", "<br/>", -1)
	m.PrivatelyOwned = book.PrivatelyOwned
	m.PrivateToken = book.PrivateToken
	m.DocCount = book.DocCount
	m.CommentStatus = book.CommentStatus
	m.CommentCount = book.CommentCount
	m.CreateTime = book.CreateTime
	m.ModifyTime = book.ModifyTime
	m.Cover = strings.ReplaceAll(book.Cover, "\\", "/")
	m.MemberId = book.MemberId
	m.Label = book.Label
	m.Status = book.Status
	m.Editor = book.Editor
	m.Theme = book.Theme
	m.Vcnt = book.Vcnt
	m.Star = book.Star
	m.Score = book.Score
	m.ScoreFloat = utils.ScoreFloat(book.Score)
	m.CntScore = book.CntScore
	m.CntComment = book.CntComment
	m.Author = book.Author
	m.AuthorURL = book.AuthorURL
	m.AdTitle = book.AdTitle
	m.AdLink = book.AdLink
	m.Lang = book.Lang

	json.Unmarshal([]byte(book.NavJSON), &m.Navs)

	if book.Theme == "" {
		m.Theme = "default"
	}

	if book.Editor == "" {
		m.Editor = "markdown"
	}
	return m
}

//重置文档数量
func (m *Book) ResetDocumentNumber(bookId int) {
	o := orm.NewOrm()
	totalCount, err := o.QueryTable(NewDocument().TableNameWithPrefix()).Filter("book_id", bookId).Count()
	if err == nil {
		o.Raw("UPDATE md_books SET doc_count = ? WHERE book_id = ?", int(totalCount), bookId).Exec()
	} else {
		beego.Error(err)
	}
}

// 内容替换
func (m *Book) Replace(bookId int, src, dst string) {
	var docs []Document
	o := orm.NewOrm()
	o.QueryTable(NewDocument()).Filter("book_id", bookId).Limit(10000).All(&docs, "document_id")
	if len(docs) > 0 {
		for _, doc := range docs {
			ds := new(DocumentStore)
			o.QueryTable(ds).Filter("document_id", doc.DocumentId).One(ds)
			if ds.DocumentId > 0 {
				if count := strings.Count(ds.Markdown, src); count > 0 { // 如果没找到内容，则不更新这篇文章
					ds.Markdown = strings.Replace(ds.Markdown, src, dst, count)
					ds.Content = ""
					ds.UpdatedAt = time.Now()
					o.Update(ds)
				}
			}
		}
	}
}

// 根据书籍id获取(公开的)书籍
func (m *Book) GetBooksById(id []int, fields ...string) (books []Book, err error) {

	var bs []Book
	var rows int64
	var idArr []interface{}

	if len(id) == 0 {
		return
	}

	for _, i := range id {
		idArr = append(idArr, i)
	}

	rows, err = orm.NewOrm().QueryTable(m).Filter("book_id__in", idArr...).Filter("privately_owned", 0).All(&bs, fields...)
	if rows > 0 {
		bookMap := make(map[interface{}]Book)
		for _, book := range bs {
			bookMap[book.BookId] = book
		}
		for _, i := range id {
			if book, ok := bookMap[i]; ok {
				books = append(books, book)
			}
		}
	}

	return
}

// 搜索书籍，这里只返回book_id
func (n *Book) SearchBook(wd string, page, size int) (books []Book, cnt int, err error) {
	sqlFmt := "select %v from md_books where privately_owned=0 and (book_name like ? or label like ? or description like ?) order by star desc"
	sqlCount := fmt.Sprintf(sqlFmt, "count(book_id) cnt")
	sql := fmt.Sprintf(sqlFmt, "book_id")

	var count struct{ Cnt int }
	wd = "%" + wd + "%"

	o := orm.NewOrm()

	err = o.Raw(sqlCount, wd, wd, wd).QueryRow(&count)
	if count.Cnt > 0 {
		cnt = count.Cnt
		_, err = o.Raw(sql+" limit ? offset ?", wd, wd, wd, size, (page-1)*size).QueryRows(&books)
	}

	return
}

// search books with labels
func (b *Book) SearchBookByLabel(labels []string, limit int, excludeIds []int) (bookIds []int, err error) {
	bookIds = []int{}
	if len(labels) == 0 {
		return
	}

	rawRegex := strings.Join(labels, "|")

	excludeClause := ""
	if len(excludeIds) == 1 {
		excludeClause = fmt.Sprintf("book_id != %d AND", excludeIds[0])
	} else if len(excludeIds) > 1 {
		excludeVal := strings.Replace(strings.Trim(fmt.Sprint(excludeIds), "[]"), " ", ",", -1)
		excludeClause = fmt.Sprintf("book_id NOT IN (%s) AND", excludeVal)
	}

	sql := fmt.Sprintf("SELECT book_id FROM md_books WHERE %v label REGEXP ? ORDER BY star DESC LIMIT ?", excludeClause)
	o := orm.NewOrm()
	_, err = o.Raw(sql, rawRegex, limit).QueryRows(&bookIds)
	if err != nil {
		logs.Error("failed to execute sql: %s, err: %s", sql, err.Error())
	}
	return
}

// Copy 拷贝书籍项目
// 1. 创建新的书籍，设置书籍的父级id为被拷贝的项目，并同步数据信息
// 2. 迁移章节内容，包括 md_documents 和 md_document_store
// 3. 替换内容中的书籍标识
// 4. 迁移书籍相关的图片等资源文件
func (m *Book) Copy(sourceBookIdentify string) (err error) {
	var (
		sourceBook      Book
		sourceDocs      []Document
		sourceDocStores []DocumentStore
		existBook       Book
		sourceDocId     []interface{}
		docMap          = make(map[int]int) // map[old_doc_id]new_doc_id
	)
	o := orm.NewOrm()
	o.Begin()
	defer func() {
		if err == nil {
			o.Commit()
			m.ResetDocumentNumber(m.BookId)
		} else {
			o.Rollback()
		}
	}()

	o.QueryTable(m).Filter("identify", sourceBookIdentify).One(&sourceBook, "book_id")
	if sourceBook.BookId <= 0 {
		return errors.New("拷贝的书籍不存在")
	}

	o.QueryTable(m).Filter("identify", m.Identify).One(&existBook, "book_id")
	if existBook.BookId > 0 {
		return errors.New("已存在相同标识的书籍，请更换书籍标识")
	}

	if m.Cover != "" {
		newCover := strings.ReplaceAll(filepath.Join(filepath.Dir(m.Cover), fmt.Sprintf("cover-%v%v", m.Identify, filepath.Ext(m.Cover))), "\\", "/")
		if utils.StoreType == utils.StoreOss {
			store.ModelStoreOss.CopyFile(m.Cover, newCover)
		} else {
			utils.CopyFile(newCover, m.Cover)
		}
		m.Cover = newCover
	}

	m.ParentId = sourceBook.BookId
	if _, err = o.Insert(m); err != nil {
		beego.Error(err)
		return errors.New("新建书籍失败：" + err.Error())
	}

	relationship := NewRelationship()
	relationship.BookId = m.BookId
	relationship.RoleId = 0
	relationship.MemberId = m.MemberId

	if _, err = o.Insert(relationship); err != nil {
		beego.Error("插入项目与用户关联 => ", err)
		return
	}

	document := NewDocument()

	o.QueryTable(document).Filter("book_id", sourceBook.BookId).Limit(10000).All(&sourceDocs)
	if len(sourceDocs) == 0 {
		return errors.New("克隆的书籍项目章节不存在")
	}

	replacer := strings.NewReplacer(
		fmt.Sprintf("/read/%s/", sourceBookIdentify), fmt.Sprintf("/read/%s/", m.Identify), // 内容中的阅读链接
		fmt.Sprintf("/projects/%s/", sourceBookIdentify), fmt.Sprintf("/projects/%s/", m.Identify), // 内容中的相关附件
	)

	for _, doc := range sourceDocs {
		oldDocId := doc.DocumentId
		doc.DocumentId = 0
		doc.BookId = m.BookId
		doc.MemberId = m.MemberId
		doc.Vcnt = 0

		// 替换相关链接等
		doc.Release = ""
		if _, err = o.Insert(&doc); err != nil {
			return errors.New("新建章节失败：" + err.Error())
		}

		sourceDocId = append(sourceDocId, oldDocId)
		docMap[oldDocId] = doc.DocumentId
	}

	o.QueryTable(NewDocumentStore()).Filter("document_id__in", sourceDocId...).Limit(10000).All(&sourceDocStores)
	for _, ds := range sourceDocStores {
		if newId, ok := docMap[ds.DocumentId]; ok {
			ds.DocumentId = newId
			ds.Markdown = replacer.Replace(ds.Markdown)
			ds.Content = ""
			o.Insert(&ds)
		}
	}

	// 更新章节所属父级ID
	sql := "update md_documents set parent_id = ? where parent_id = ? and book_id = ?"
	for oldId, newId := range docMap {
		if _, err = o.Raw(sql, newId, oldId, m.BookId).Exec(); err != nil {
			return
		}
	}

	sourceDir := "projects/" + sourceBookIdentify
	targetDir := "projects/" + m.Identify

	if utils.StoreType == utils.StoreOss {
		err = store.ModelStoreOss.CopyDir(sourceDir, targetDir)
	} else {
		sourceDir = "uploads/" + sourceDir
		targetDir = "uploads/" + targetDir
		if _, e := os.Stat(sourceDir); e == nil { // 存在文件夹
			err = store.ModelStoreLocal.CopyDir(sourceDir, targetDir)
		}
	}
	return
}

// Export2Markdown 将书籍导出markdown
func (m *Book) Export2Markdown(identify string) (path string, err error) {
	var (
		book               *Book
		exportDir          = fmt.Sprintf("uploads/export/%v", identify)
		attachPrefix       string
		exportAttachPrefix = "../attachments/"
		docs               []Document
		ds                 []DocumentStore
		docIds             []interface{}
		docMap             = make(map[int]Document)
		o                  = orm.NewOrm()
		isOSSProject       = utils.StoreType == utils.StoreOss
		cover              = ""
		replaces           []string
	)

	path = fmt.Sprintf("uploads/export/%v.zip", identify)
	if book, err = m.FindByIdentify(identify); err != nil {
		beego.Error(err)
		return
	}

	os.MkdirAll(filepath.Join(exportDir, "docs"), os.ModePerm)
	// 最后删除导出目录
	defer func() {
		os.RemoveAll(exportDir)
	}()

	attachPrefix = fmt.Sprintf("/uploads/projects/%s/", identify)
	if isOSSProject {
		attachPrefix = fmt.Sprintf("/projects/%s/", identify)
	}

	o.QueryTable(NewDocument()).Filter("book_id", book.BookId).Limit(100000).All(&docs, "document_id", "document_name", "identify")
	if len(docs) == 0 {
		err = errors.New("找不到书籍章节")
		return
	}

	ext := ".md"
	replaces = append(replaces, attachPrefix, exportAttachPrefix)
	for _, doc := range docs {
		docIds = append(docIds, doc.DocumentId)
		identify := doc.Identify
		id := strconv.Itoa(doc.DocumentId)

		if docExt := strings.ToLower(strings.TrimSpace(filepath.Ext(doc.Identify))); docExt != ".md" {
			doc.Identify = doc.Identify + ext
		}
		docMap[doc.DocumentId] = doc
		replaces = append(replaces,
			"]($"+identify, "]("+doc.Identify,
			"href=\"$"+identify, "href=\""+doc.Identify,
			"]($"+id, "]("+doc.Identify,
			"href=\"$"+id, "href=\""+doc.Identify,
		)
	}

	replaces = append(replaces, "]($", "](", "href=\"$", "href=\"")

	// 图片等文件附件链接、URL链接等替换
	replacer := strings.NewReplacer(replaces...)

	o.QueryTable(NewDocumentStore()).Filter("document_id__in", docIds...).Limit(100000).All(&ds)
	for _, item := range ds {
		doc, ok := docMap[item.DocumentId]
		if !ok {
			continue
		}

		// 基本的链接替换
		md := replacer.Replace(item.Markdown)
		file := filepath.Join(exportDir, "docs", doc.Identify)
		ioutil.WriteFile(file, []byte(md), os.ModePerm)
	}

	// SUMMARY 文档内容
	docModel := NewDocument()
	cont, _ := docModel.CreateDocumentTreeForHtml(book.BookId, 0)
	// 把最后没有 .md 结尾的链接替换为 .md 结尾
	if gq, _ := goquery.NewDocumentFromReader(strings.NewReader(cont)); gq != nil {
		gq.Find("a").Each(func(i int, sel *goquery.Selection) {
			if href, ok := sel.Attr("href"); ok {
				if strings.ToLower(filepath.Ext(href)) != ".md" {
					href = href + ".md"
					sel.SetAttr("href", href)
				}
			}
		})
		cont, _ = gq.Html()
	}
	md := html2md.Convert(cont)
	md = strings.ReplaceAll(md, fmt.Sprintf("](/read/%s/", identify), "](docs/")
	md = fmt.Sprintf("- [%v](README.md)\n", book.BookName) + md
	summaryFile := filepath.Join(exportDir, "SUMMARY.md")
	ioutil.WriteFile(summaryFile, []byte(md), os.ModePerm)

	// 书籍封面处理
	if book.Cover != "" {
		cover = fmt.Sprintf("![封面](attachments/%s)", strings.TrimPrefix(strings.TrimLeft(strings.ReplaceAll(book.Cover, "\\", "/"), "./"), strings.TrimLeft(attachPrefix, "./")))
	}

	// README
	md = fmt.Sprintf("# %s\n\n%s\n\n%s\n\n\n\n## 目录\n\n%s", book.BookName, cover, book.Description, md)
	readmeFile := filepath.Join(exportDir, "README.md")
	ioutil.WriteFile(readmeFile, []byte(md), os.ModePerm)

	dst := filepath.Join(exportDir, "attachments")
	src := fmt.Sprintf("projects/%s", identify)
	if !isOSSProject {
		src = "uploads/" + src
	}

	if errDown := m.down2local(src, dst); errDown != nil {
		beego.Error("down2local error:", errDown.Error())
	}

	err = m.zip(exportDir, path)
	if err != nil {
		beego.Error("压缩失败：", err.Error())
	}
	return
}

// 把书籍相关附件下载到本地
func (m *Book) down2local(srcDir, desDir string) (err error) {
	if utils.StoreType == utils.StoreLocal {
		return store.ModelStoreLocal.CopyDir(srcDir, desDir)
	}
	return store.ModelStoreOss.Down2local(srcDir, desDir)
}

func (m *Book) zip(dir, zipFile string) (err error) {
	// zip 压缩

	var (
		d     *os.File
		fw    io.Writer
		fcont []byte
		fl    []filetil.FileList
	)

	if fl, err = filetil.ScanFiles(dir); err != nil {
		return
	}

	os.Remove(zipFile)
	d, err = os.Create(zipFile)
	if err != nil {
		beego.Error(err)
		return
	}
	defer d.Close()
	zipWriter := zip.NewWriter(d)
	defer zipWriter.Close()

	for _, file := range fl {
		if file.IsDir {
			continue
		}

		info, errInfo := os.Stat(file.Path)
		if errInfo != nil {
			beego.Error(errInfo)
			continue
		}

		header, errHeader := zip.FileInfoHeader(info)
		if errHeader != nil {
			beego.Error(errHeader)
			continue
		}

		header.Method = zip.Deflate
		header.Name = strings.TrimLeft(strings.TrimPrefix(strings.ReplaceAll(file.Path, "\\", "/"), dir), "./")

		fw, err = zipWriter.CreateHeader(header)
		if err != nil {
			beego.Error(err)
			return
		}

		fcont, err = ioutil.ReadFile(file.Path)
		if err != nil {
			return
		}

		if _, err = fw.Write(fcont); err != nil {
			return
		}
	}
	return
}
