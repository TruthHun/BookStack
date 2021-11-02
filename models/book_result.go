package models

import (
	"encoding/json"
	"strings"
	"time"

	"strconv"

	"fmt"

	"github.com/TruthHun/BookStack/conf"
	"github.com/astaxie/beego/logs"
	"github.com/astaxie/beego/orm"
)

type BookResult struct {
	BookId           int       `json:"book_id"`
	BookName         string    `json:"book_name"`
	Identify         string    `json:"identify"`
	OrderIndex       int       `json:"order_index"`
	Description      string    `json:"description"`
	PrivatelyOwned   int       `json:"privately_owned"`
	PrivateToken     string    `json:"private_token"`
	DocCount         int       `json:"doc_count"`
	CommentStatus    string    `json:"comment_status"`
	CommentCount     int       `json:"comment_count"`
	CreateTime       time.Time `json:"create_time"`
	CreateName       string    `json:"create_name"`
	ModifyTime       time.Time `json:"modify_time"`
	Cover            string    `json:"cover"`
	Theme            string    `json:"theme"`
	Label            string    `json:"label"`
	MemberId         int       `json:"member_id"`
	Username         int       `json:"user_name"`
	Editor           string    `json:"editor"`
	RelationshipId   int       `json:"relationship_id"`
	RoleId           int       `json:"role_id"`
	RoleName         string    `json:"role_name"`
	Status           int
	Vcnt             int       `json:"vcnt"`
	Star             int       `json:"star"`
	Score            int       `json:"score"`
	CntComment       int       `json:"cnt_comment"`
	CntScore         int       `json:"cnt_score"`
	ScoreFloat       string    `json:"score_float"`
	LastModifyText   string    `json:"last_modify_text"`
	IsDisplayComment bool      `json:"is_display_comment"`
	Author           string    `json:"author"`
	AuthorURL        string    `json:"author_url"`
	AdTitle          string    `json:"ad_title"`
	AdLink           string    `json:"ad_link"`
	Lang             string    `json:"lang"`
	Navs             []BookNav `json:"navs"`
}

func NewBookResult() *BookResult {
	return &BookResult{}
}

// 根据书籍标识查询书籍以及指定用户权限的信息.
func (m *BookResult) FindByIdentify(identify string, memberId int) (result *BookResult, err error) {
	if identify == "" || memberId <= 0 {
		return result, ErrInvalidParameter
	}
	o := orm.NewOrm()

	book := NewBook()

	err = o.QueryTable(book.TableNameWithPrefix()).Filter("identify", identify).One(book)
	if err != nil {
		return
	}

	json.Unmarshal([]byte(book.NavJSON), &book.Navs)

	relationship := NewRelationship()

	err = o.QueryTable(relationship.TableNameWithPrefix()).Filter("book_id", book.BookId).Filter("member_id", memberId).One(relationship)
	if err != nil {
		return
	}

	var relationship2 Relationship

	err = o.QueryTable(relationship.TableNameWithPrefix()).Filter("book_id", book.BookId).Filter("role_id", conf.BookFounder).One(&relationship2)
	if err != nil {
		logs.Error("根据书籍标识查询书籍以及指定用户权限的信息 => ", err)
		return result, ErrPermissionDenied
	}

	member, err := NewMember().Find(relationship2.MemberId)
	if err != nil {
		return result, err
	}

	result = book.ToBookResult()
	result.CreateName = member.Account
	result.MemberId = relationship.MemberId
	result.RoleId = relationship.RoleId
	result.RelationshipId = relationship.RelationshipId
	result.Navs = book.Navs

	switch result.RoleId {
	case conf.BookFounder:
		result.RoleName = "创始人"
	case conf.BookAdmin:
		result.RoleName = "管理员"
	case conf.BookEditor:
		result.RoleName = "编辑者"
	case conf.BookObserver:
		result.RoleName = "观察者"
	}

	doc := NewDocument()

	err = o.QueryTable(doc.TableNameWithPrefix()).Filter("book_id", book.BookId).OrderBy("modify_time").One(doc)
	if err == nil {
		member2 := NewMember()
		member2.Find(doc.ModifyAt)
		result.LastModifyText = member2.Account + " 于 " + doc.ModifyTime.Format("2006-01-02 15:04:05")
	}
	return
}

func (m *BookResult) FindToPager(pageIndex, pageSize int, private int, wd ...string) (books []*BookResult, totalCount int, err error) {
	var args []interface{}
	word := ""
	if len(wd) > 0 {
		if w := strings.TrimSpace(wd[0]); w != "" {
			word = w
		}
	}
	o := orm.NewOrm()
	q := o.QueryTable(NewBook().TableNameWithPrefix())
	if word != "" {
		cond := orm.NewCondition().Or("book_name__icontains", word).Or("description__icontains", word)
		q = q.SetCond(orm.NewCondition().AndCond(cond))
	}
	q = q.Filter("privately_owned", private)

	count, err := q.Count()
	if err != nil {
		return
	}

	totalCount = int(count)

	sql := `SELECT
			book.*,rel.relationship_id,rel.role_id,m.account AS create_name
		FROM md_books AS book
			LEFT JOIN md_relationship AS rel ON rel.book_id = book.book_id AND rel.role_id = 0
			LEFT JOIN md_members AS m ON rel.member_id = m.member_id %v
		ORDER BY book.order_index DESC ,book.book_id DESC  LIMIT ?,?`
	condition := ""
	condition = "where book.privately_owned=" + strconv.Itoa(private)
	if word != "" {
		condition = condition + " and (book.book_name like ? or book.description like ?)"
		args = append(args, "%"+word+"%", "%"+word+"%")
	}

	sql = fmt.Sprintf(sql, condition)
	offset := (pageIndex - 1) * pageSize
	args = append(args, offset, pageSize)
	_, err = o.Raw(sql, args...).QueryRows(&books)
	return
}
