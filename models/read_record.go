package models

import (
	"time"

	"github.com/astaxie/beego"

	"strconv"

	"fmt"

	"errors"

	"github.com/astaxie/beego/orm"
)

//增加一个重新阅读的功能，即重置阅读，清空所有阅读记录

//阅读记录.用于记录阅读的文档，以及阅读进度统计
type ReadRecord struct {
	Id       int //自增主键
	BookId   int `orm:"index"` //书籍id
	DocId    int //文档id
	Uid      int `orm:"index"` //用户id
	CreateAt int //记录创建时间，也就是内容阅读时间
}

// 阅读统计
// 用来记录一本书（假设有100个章节），用户已经阅读了多少章节，以标识用户书籍的阅读进度
// 从而不用每次从阅读记录的表 read_record 表里面进行mysql 的 count 统计
type ReadCount struct {
	Id     int // 自增主键
	BookId int // 书籍
	Uid    int // 用户id
	Cnt    int // 阅读的文档数
}

//阅读记录列表（非表）
type RecordList struct {
	DocId    int
	Title    string
	Identify string
	CreateAt int
}

//阅读进度(非表)
type ReadProgress struct {
	Cnt          int    `json:"cnt"`     //已阅读过的文档
	Total        int    `json:"total"`   //总文档
	Percent      string `json:"percent"` //占的百分比
	BookIdentify string `json:"book_identify"`
}

// 阅读计时规则
type ReadingRule struct {
	Min       int
	Max       int
	MaxReward int
	Invalid   int
}

func NewReadRecord() *ReadRecord {
	return &ReadRecord{}
}

// 多字段唯一键
func (this *ReadCount) TableUnique() [][]string {
	return [][]string{
		[]string{"BookId", "Uid"},
	}
}

// 多字段唯一键
func (this *ReadRecord) TableUnique() [][]string {
	return [][]string{
		[]string{"DocId", "Uid"},
	}
}

var (
	tableReadRecord = "md_read_record"
	tableReadCount  = "md_read_count"
	_readingRule    = &ReadingRule{}
)

//添加阅读记录
func (this *ReadRecord) Add(docId, uid int) (err error) {
	// 1、根据文档id查询书籍id
	// 2、写入或者更新阅读记录
	// 3、更新书籍被阅读的文档统计
	// 4、更新用户阅读时长
	var (
		doc      Document
		r        ReadRecord
		o        = orm.NewOrm()
		tableDoc = NewDocument()
		member   = NewMember()
		now      = time.Now()
		rt       = NewReadingTime()
	)

	err = o.QueryTable(tableDoc).Filter("document_id", docId).One(&doc, "book_id")
	if err != nil {
		beego.Error(err)
		return
	}

	if doc.BookId <= 0 {
		return
	}

	record := ReadRecord{
		BookId:   doc.BookId,
		DocId:    docId,
		Uid:      uid,
		CreateAt: int(now.Unix()),
	}

	// 更新书架中的书籍最后的阅读时间
	go new(Star).SetLastReadTime(uid, doc.BookId)

	// 计算奖励的阅读时长
	readingTime, lastReadDocId := this.calcReadingTime(uid, docId, now)
	// 如果现在阅读的文档id与上次阅读的文档id相同，则不更新阅读时长
	if lastReadDocId == docId {
		return
	}

	o.Begin()
	defer func() {
		if err != nil {
			o.Rollback()
			beego.Error(err)
		} else {
			o.Commit()
		}
	}()

	o.QueryTable(tableReadRecord).Filter("doc_id", docId).Filter("uid", uid).One(&r, "id")

	readCnt := 1
	if r.Id > 0 { // 先删再增，以便根据主键id索引的倒序查询列表
		o.QueryTable(tableReadRecord).Filter("id", r.Id).Delete()
		readCnt = 0 // 如果是更新，则阅读次数
	}

	// 更新阅读记录
	_, err = o.Insert(&record)
	if err != nil {
		return
	}

	if readCnt == 1 {
		rc := &ReadCount{}
		o.QueryTable(tableReadCount).Filter("uid", uid).Filter("book_id", doc.BookId).One(rc)
		if rc.Id > 0 { // 更新已存在的阅读进度统计记录
			rc.Cnt += 1
			_, err = o.Update(rc)
		} else { // 增加阅读进度统计记录
			rc = &ReadCount{BookId: doc.BookId, Uid: uid, Cnt: 1}
			_, err = o.Insert(rc)
		}
	}
	if err != nil {
		return
	}
	if readingTime <= 0 {
		return
	}

	o.QueryTable(member).Filter("member_id", uid).One(member, "member_id", "total_reading_time")
	if member.MemberId > 0 {
		_, err = o.QueryTable(member).Filter("member_id", uid).Update(orm.Params{"total_reading_time": member.TotalReadingTime + readingTime})
		if err != nil {
			return
		}
		o.QueryTable(rt).Filter("uid", uid).Filter("day", now.Format(signDayLayout)).One(rt)
		if rt.Id > 0 {
			rt.Duration += readingTime
			_, err = o.Update(rt)
		} else {
			rt.Day, _ = strconv.Atoi(now.Format(signDayLayout))
			rt.Uid = uid
			rt.Duration = readingTime
			_, err = o.Insert(rt)
		}
	}
	return
}

// 查询用户最后的一条阅读记录
func (this *ReadRecord) LastReading(uid int, cols ...string) (r ReadRecord) {
	orm.NewOrm().QueryTable(this).Filter("uid", uid).OrderBy("-id").One(&r)
	return
}

func (this *ReadRecord) HistoryReadBook(uid, page, size int) (books []Book) {
	// 由于md_books 没有 created_at 这个字段，所以这里将这个字段映射到version里面...
	fields := "b.book_id,b.book_name,b.cover,b.vcnt,b.doc_count,b.description,b.identify,max(rr.create_at) as version"
	sql := `
select 
	%v 
from 
	md_read_record rr 
left join 
	md_books b 
on 
	rr.book_id = b.book_id 
where 
	b.privately_owned=0 and rr.uid = ? 
group by b.book_id 
order by version desc 
limit ? offset ?
`
	sql = fmt.Sprintf(sql, fields)
	orm.NewOrm().Raw(sql, uid, size, (page-1)*size).QueryRows(&books)
	for idx, book := range books {
		book.ModifyTime = time.Unix(book.Version, 0)
		book.Version = 0
		books[idx] = book
	}
	return
}

//清空阅读记录
//当删除书籍时，直接删除该书籍的所有记录
func (this *ReadRecord) Clear(uid, bookId int) (err error) {
	o := orm.NewOrm()
	if bookId > 0 && uid > 0 {
		_, err = o.QueryTable(tableReadCount).Filter("uid", uid).Filter("book_id", bookId).Delete()
		if err == nil {
			_, err = o.QueryTable(tableReadRecord).Filter("uid", uid).Filter("book_id", bookId).Delete()
		}
	} else if uid == 0 && bookId > 0 {
		_, err = o.QueryTable(tableReadCount).Filter("book_id", bookId).Delete()
		if err == nil {
			_, err = o.QueryTable(tableReadRecord).Filter("book_id", bookId).Delete()
		}
	}
	return
}

//查询阅读记录
func (this *ReadRecord) List(uid, bookId int) (lists []RecordList, cnt int64, err error) {
	if uid*bookId == 0 {
		err = errors.New("用户id和书籍id不能为空")
		return
	}
	fields := "r.doc_id,r.create_at,d.document_name title,d.identify"
	sql := "select %v from %v r left join md_documents d on r.doc_id=d.document_id where r.book_id=? and r.uid=? order by r.id desc limit 5000"
	sql = fmt.Sprintf(sql, fields, tableReadRecord)
	cnt, err = orm.NewOrm().Raw(sql, bookId, uid).QueryRows(&lists)
	return
}

//查询阅读进度
func (this *ReadRecord) Progress(uid, bookId int) (rp ReadProgress, err error) {
	if uid*bookId == 0 {
		err = errors.New("用户id和书籍id均不能为空")
		return
	}
	var (
		rc   ReadCount
		book = new(Book)
	)
	o := orm.NewOrm()
	if err = o.QueryTable(tableReadCount).Filter("uid", uid).Filter("book_id", bookId).One(&rc, "cnt"); err == nil {
		if err = o.QueryTable(book).Filter("book_id", bookId).One(book, "doc_count", "identify"); err == nil {
			rp.Total = book.DocCount
		}
	}
	rp.Cnt = rc.Cnt
	rp.BookIdentify = book.Identify
	if rp.Total == 0 {
		rp.Percent = "0.00%"
	} else {
		if rp.Cnt > rp.Total {
			rp.Cnt = rp.Total
		}
		f := float32(rp.Cnt) / float32(rp.Total)
		rp.Percent = fmt.Sprintf("%.2f", f*100) + "%"
	}
	return
}

// 查询阅读进度
func (this *ReadRecord) BooksProgress(uid int, bookId ...int) (read map[int]int) {
	read = make(map[int]int)
	var count []ReadCount
	orm.NewOrm().QueryTable(new(ReadCount)).Filter("uid", uid).Filter("book_id__in", bookId).All(&count)
	for _, item := range count {
		read[item.BookId] = item.Cnt
	}

	for _, id := range bookId {
		if _, ok := read[id]; !ok {
			read[id] = 0
		}
	}
	return
}

//删除单条阅读记录
func (this *ReadRecord) Delete(uid, docId int) (err error) {
	if uid*docId == 0 {
		err = errors.New("用户id和文档id不能为空")
		return
	}

	var record ReadRecord

	o := orm.NewOrm()
	o.QueryTable(tableReadRecord).Filter("uid", uid).Filter("doc_id", docId).One(&record, "book_id", "id")
	if record.BookId > 0 { //存在，则删除该阅读记录
		if _, err = o.QueryTable(tableReadRecord).Filter("id", record.Id).Delete(); err == nil {
			err = SetIncreAndDecre(tableReadCount, "cnt", "book_id="+strconv.Itoa(record.BookId)+" and uid="+strconv.Itoa(uid), false, 1)
		}
	}
	return
}

// 更新签到奖励规则
func (*ReadRecord) UpdateReadingRule() {
	ops := []string{"READING_MIN_INTERVAL", "READING_MAX_INTERVAL", "READING_INTERVAL_MAX_REWARD", "READING_INVALID_INTERVAL"}
	for _, op := range ops {
		num, _ := strconv.Atoi(GetOptionValue(op, ""))
		switch op {
		case "READING_MIN_INTERVAL":
			_readingRule.Min = num
		case "READING_MAX_INTERVAL":
			_readingRule.Max = num
		case "READING_INTERVAL_MAX_REWARD":
			_readingRule.MaxReward = num
		case "READING_INVALID_INTERVAL":
			_readingRule.Invalid = num
		}
	}
}

// 获取阅读计时规则
func (*ReadRecord) GetReadingRule() (r *ReadingRule) {
	return _readingRule
}

// 在 5 - 600 秒之间的阅读计时，正常计时
// 在 600 - 1800 秒(半个小时)之间的计时，按最大计时来计算时长
// 超过半个小时之后才有阅读记录，则在此期间的阅读时长为0
func (*ReadRecord) calcReadingTime(uid, docId int, t time.Time) (duration int, lastReadDocId int) {
	r := NewReadRecord()
	rr := r.LastReading(uid, "uid", "doc_id", "created_at")
	if rr.DocId == docId {
		return 0, docId
	}

	rule := r.GetReadingRule()
	diff := int(t.Unix()) - rr.CreateAt
	if diff <= 0 || diff < rule.Min || diff >= rule.Invalid {
		return 0, rr.DocId
	}

	if diff > rule.MaxReward {
		return rule.Max, rr.DocId
	}
	return diff, rr.DocId
}
