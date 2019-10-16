package models

import (
	"os"
	"time"

	"strings"

	"fmt"

	"strconv"

	"github.com/TruthHun/BookStack/conf"
	"github.com/TruthHun/BookStack/models/store"
	"github.com/TruthHun/BookStack/utils"
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

// Book struct .
type Book struct {
	BookId            int       `orm:"pk;auto;unique;column(book_id)" json:"book_id"`
	BookName          string    `orm:"column(book_name);size(500)" json:"book_name"`      // BookName 项目名称.
	Identify          string    `orm:"column(identify);size(100);unique" json:"identify"` // Identify 项目唯一标识.
	OrderIndex        int       `orm:"column(order_index);type(int);default(0)" json:"order_index"`
	Pin               int       `orm:"column(pin);type(int);default(0)" json:"pin"`       // pin值，用于首页固定显示
	Description       string    `orm:"column(description);size(2000)" json:"description"` // Description 项目描述.
	Label             string    `orm:"column(label);size(500)" json:"label"`
	PrivatelyOwned    int       `orm:"column(privately_owned);type(int);default(0)" json:"privately_owned"` // PrivatelyOwned 项目私有： 0 公开/ 1 私有
	PrivateToken      string    `orm:"column(private_token);size(500);null" json:"private_token"`           // 当项目是私有时的访问Token.
	Status            int       `orm:"column(status);type(int);default(0)" json:"status"`                   //状态：0 正常/1 已删除
	Editor            string    `orm:"column(editor);size(50)" json:"editor"`                               //默认的编辑器.
	DocCount          int       `orm:"column(doc_count);type(int)" json:"doc_count"`                        // DocCount 包含文档数量.
	CommentStatus     string    `orm:"column(comment_status);size(20);default(open)" json:"comment_status"` // CommentStatus 评论设置的状态:open 为允许所有人评论，closed 为不允许评论, group_only 仅允许参与者评论 ,registered_only 仅允许注册者评论.
	CommentCount      int       `orm:"column(comment_count);type(int)" json:"comment_count"`
	Cover             string    `orm:"column(cover);size(1000)" json:"cover"`                              //封面地址
	Theme             string    `orm:"column(theme);size(255);default(default)" json:"theme"`              //主题风格
	CreateTime        time.Time `orm:"type(datetime);column(create_time);auto_now_add" json:"create_time"` // CreateTime 创建时间 .
	MemberId          int       `orm:"column(member_id);size(100)" json:"member_id"`
	ModifyTime        time.Time `orm:"type(datetime);column(modify_time);auto_now" json:"modify_time"`
	ReleaseTime       time.Time `orm:"type(datetime);column(release_time);" json:"release_time"`   //项目发布时间，每次发布都更新一次，如果文档更新时间小于发布时间，则文档不再执行发布
	GenerateTime      time.Time `orm:"type(datetime);column(generate_time);" json:"generate_time"` //下载文档生成时间
	LastClickGenerate time.Time `orm:"type(datetime);column(last_click_generate)" json:"-"`        //上次点击上传文档的时间，用于显示频繁点击浪费服务器硬件资源的情况
	Version           int64     `orm:"type(bigint);column(version);default(0)" json:"version"`
	Vcnt              int       `orm:"column(vcnt);default(0)" json:"vcnt"`    // 文档项目被阅读次数
	Star              int       `orm:"column(star);default(0)" json:"star"`    // 文档项目被收藏次数
	Score             int       `orm:"column(score);default(40)" json:"score"` // 文档项目评分，默认40，即4.0星
	CntScore          int       // 评分人数
	CntComment        int       // 评论人数
	Author            string    `orm:"size(50)"`            //原作者，即来源
	AuthorURL         string    `orm:"column(author_url)"`  //原作者链接，即来源链接
	AdTitle           string    `orm:"default()"`           // 文字广告标题
	AdLink            string    `orm:"default();size(512)"` // 文字广告链接
	Lang              string    `orm:"size(10);index;default(zh)"`
}

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
		logs.Error("插入项目与用户关联 => ", err)
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

//分页查询指定用户的项目
//按照最新的进行排序
func (m *Book) FindToPager(pageIndex, pageSize, memberId int, PrivatelyOwned ...int) (books []*BookResult, totalCount int, err error) {

	relationship := NewRelationship()

	o := orm.NewOrm()
	sql1 := "SELECT COUNT(book.book_id) AS total_count FROM " + m.TableNameWithPrefix() + " AS book LEFT JOIN " +
		relationship.TableNameWithPrefix() + " AS rel ON book.book_id=rel.book_id AND rel.member_id = ? WHERE rel.relationship_id > 0 "
	if len(PrivatelyOwned) > 0 {
		sql1 = sql1 + " and book.privately_owned=" + strconv.Itoa(PrivatelyOwned[0])
	}
	err = o.Raw(sql1, memberId).QueryRow(&totalCount)
	if err != nil {
		return
	}

	offset := (pageIndex - 1) * pageSize
	sql2 := "SELECT book.*,rel.member_id,rel.role_id,m.account as create_name FROM " + m.TableNameWithPrefix() + " AS book" +
		" LEFT JOIN " + relationship.TableNameWithPrefix() + " AS rel ON book.book_id=rel.book_id AND rel.member_id = ? " +
		" LEFT JOIN " + NewMember().TableNameWithPrefix() + " AS m ON rel.member_id=m.member_id " +
		" WHERE rel.relationship_id > 0 %v ORDER BY book.book_id DESC LIMIT " + fmt.Sprintf("%d,%d", offset, pageSize)

	if len(PrivatelyOwned) > 0 {
		sql2 = fmt.Sprintf(sql2, " and book.privately_owned="+strconv.Itoa(PrivatelyOwned[0]))
	}
	_, err = o.Raw(sql2, memberId).QueryRows(&books)
	if err != nil {
		logs.Error("分页查询项目列表 => ", err)
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

// 彻底删除项目.
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
	//删除oss中项目对应的文件夹
	switch utils.StoreType {
	case utils.StoreLocal: //删除本地存储，记得加上uploads
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
		order = "book_id desc"
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
	m.Cover = book.Cover
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
				ds.Markdown = strings.Replace(ds.Markdown, src, dst, -1)
				ds.Content = strings.Replace(ds.Content, src, dst, -1)
				o.Update(ds)
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
