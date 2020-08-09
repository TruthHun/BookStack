package models

import (
	"errors"

	"github.com/TruthHun/BookStack/conf"
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/logs"
	"github.com/astaxie/beego/orm"
)

type Relationship struct {
	RelationshipId int `orm:"pk;auto;unique;column(relationship_id)" json:"relationship_id"`
	MemberId       int `orm:"column(member_id);type(int)" json:"member_id"`
	BookId         int `orm:"column(book_id);type(int);index" json:"book_id"`
	RoleId         int `orm:"column(role_id);type(int);index" json:"role_id"` // RoleId 角色：0 创始人(创始人不能被移除) / 1 管理员/2 编辑者/3 观察者
}

// TableName 获取对应数据库表名.
func (m *Relationship) TableName() string {
	return "relationship"
}
func (m *Relationship) TableNameWithPrefix() string {
	return conf.GetDatabasePrefix() + m.TableName()
}

// TableEngine 获取数据使用的引擎.
func (m *Relationship) TableEngine() string {
	return "INNODB"
}

// 联合唯一键
func (u *Relationship) TableUnique() [][]string {
	return [][]string{
		[]string{"MemberId", "BookId"},
	}
}

func NewRelationship() *Relationship {
	return &Relationship{}
}

func (m *Relationship) Find(id int) (*Relationship, error) {
	o := orm.NewOrm()

	err := o.QueryTable(m.TableNameWithPrefix()).Filter("relationship_id", id).One(m)
	return m, err
}

//查询指定书籍的创始人.
func (m *Relationship) FindFounder(bookId int) (*Relationship, error) {
	o := orm.NewOrm()

	err := o.QueryTable(m.TableNameWithPrefix()).Filter("book_id", bookId).Filter("role_id", 0).One(m)

	return m, err
}

func (m *Relationship) UpdateRoleId(bookId, memberId, roleId int) (*Relationship, error) {
	o := orm.NewOrm()
	book := NewBook()
	book.BookId = bookId

	if err := o.Read(book); err != nil {
		logs.Error("UpdateRoleId => ", err)
		return m, errors.New("书籍不存在")
	}
	err := o.QueryTable(m.TableNameWithPrefix()).Filter("member_id", memberId).Filter("book_id", bookId).One(m)
	if err == orm.ErrNoRows {
		m.BookId = bookId
		m.MemberId = memberId
		m.RoleId = roleId
	} else if err != nil {
		return m, err
	} else if m.RoleId == conf.BookFounder {
		return m, errors.New("不能变更创始人的权限")
	}
	m.RoleId = roleId

	if m.RelationshipId > 0 {
		_, err = o.Update(m)
	} else {
		_, err = o.Insert(m)
	}

	return m, err

}

func (m *Relationship) FindForRoleId(bookId, memberId int) (int, error) {
	o := orm.NewOrm()
	err := o.QueryTable(m.TableNameWithPrefix()).Filter("book_id", bookId).Filter("member_id", memberId).One(m, "role_id")
	if err != nil {
		return 0, err
	}
	return m.RoleId, nil
}

func (m *Relationship) FindByBookIdAndMemberId(bookId, memberId int) (*Relationship, error) {
	o := orm.NewOrm()
	err := o.QueryTable(m.TableNameWithPrefix()).Filter("book_id", bookId).Filter("member_id", memberId).One(m)
	return m, err
}

func (m *Relationship) Insert() error {
	o := orm.NewOrm()
	_, err := o.Insert(m)
	return err
}

func (m *Relationship) Update() error {
	o := orm.NewOrm()
	_, err := o.Update(m)
	return err
}

func (m *Relationship) DeleteByBookIdAndMemberId(bookId, memberId int) error {
	o := orm.NewOrm()

	err := o.QueryTable(m.TableNameWithPrefix()).Filter("book_id", bookId).Filter("member_id", memberId).One(m)

	if err == orm.ErrNoRows {
		return errors.New("用户未参与该书籍")
	}
	if m.RoleId == conf.BookFounder {
		return errors.New("不能删除创始人")
	}
	_, err = o.Delete(m)

	if err != nil {
		logs.Error("删除书籍参与者 => ", err)
		return errors.New("删除失败")
	}
	return nil

}

func (m *Relationship) Transfer(bookId, founderId, receiveId int) error {
	o := orm.NewOrm()

	founder := NewRelationship()

	err := o.QueryTable(m.TableNameWithPrefix()).Filter("book_id", bookId).Filter("member_id", founderId).One(founder)

	if err != nil {
		return err
	}
	if founder.RoleId != conf.BookFounder {
		return errors.New("转让者不是创始人")
	}
	receive := NewRelationship()

	err = o.QueryTable(m.TableNameWithPrefix()).Filter("book_id", bookId).Filter("member_id", receiveId).One(receive)

	if err != orm.ErrNoRows && err != nil {
		return err
	}
	o.Begin()

	founder.RoleId = conf.BookAdmin

	receive.MemberId = receiveId
	receive.RoleId = conf.BookFounder
	receive.BookId = bookId

	if err := founder.Update(); err != nil {
		o.Rollback()
		return err
	}
	if receive.RelationshipId > 0 {
		if _, err := o.Update(receive); err != nil {
			o.Rollback()
			return err
		}
	} else {
		if _, err := o.Insert(receive); err != nil {
			o.Rollback()
			return err
		}
	}

	return o.Commit()
}

// HasRelatedBook 查询用户是否有相关联的书籍
func (m *Relationship) HasRelatedBook(uid int) bool {
	err := orm.NewOrm().QueryTable(m).Filter("member_id", uid).One(m, "relationship_id")
	if err != nil && err != orm.ErrNoRows {
		beego.Error(err)
		return false
	}
	return m.RelationshipId > 0
}
