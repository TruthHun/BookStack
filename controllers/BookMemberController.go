package controllers

import (
	"errors"

	"github.com/TruthHun/BookStack/conf"
	"github.com/TruthHun/BookStack/models"
	"github.com/astaxie/beego/logs"
	"github.com/astaxie/beego/orm"
)

type BookMemberController struct {
	BaseController
}

// AddMember 参加参与用户.
func (this *BookMemberController) AddMember() {
	identify := this.GetString("identify")
	account := this.GetString("account")
	role_id, _ := this.GetInt("role_id", 3)

	if identify == "" || account == "" {
		this.JsonResult(6001, "参数错误")
	}
	book, err := this.IsPermission()

	if err != nil {
		this.JsonResult(6001, err.Error())
	}

	member := models.NewMember()

	if _, err := member.FindByAccount(account); err != nil {
		this.JsonResult(404, "用户不存在")
	}
	if member.Status == 1 {
		this.JsonResult(6003, "用户已被禁用")
	}

	if _, err := models.NewRelationship().FindForRoleId(book.BookId, member.MemberId); err == nil {
		this.JsonResult(6003, "用户已存在该项目中")
	}

	relationship := models.NewRelationship()
	relationship.BookId = book.BookId
	relationship.MemberId = member.MemberId
	relationship.RoleId = role_id

	if err := relationship.Insert(); err == nil {
		memberRelationshipResult := models.NewMemberRelationshipResult().FromMember(member)
		memberRelationshipResult.RoleId = role_id
		memberRelationshipResult.RelationshipId = relationship.RelationshipId
		memberRelationshipResult.BookId = book.BookId
		memberRelationshipResult.ResolveRoleName()

		this.JsonResult(0, "ok", memberRelationshipResult)
	}
	this.JsonResult(500, err.Error())
}

// 变更指定用户在指定项目中的权限
func (this *BookMemberController) ChangeRole() {
	identify := this.GetString("identify")
	member_id, _ := this.GetInt("member_id", 0)
	role, _ := this.GetInt("role_id", 0)

	if identify == "" || member_id <= 0 {
		this.JsonResult(6001, "参数错误")
	}
	if member_id == this.Member.MemberId {
		this.JsonResult(6006, "不能变更自己的权限")
	}
	book, err := models.NewBookResult().FindByIdentify(identify, this.Member.MemberId)

	if err != nil {
		if err == models.ErrPermissionDenied {
			this.JsonResult(403, "权限不足")
		}
		if err == orm.ErrNoRows {
			this.JsonResult(404, "项目不存在")
		}
		this.JsonResult(6002, err.Error())
	}
	if book.RoleId != 0 && book.RoleId != 1 {
		this.JsonResult(403, "权限不足")
	}

	member := models.NewMember()

	if _, err := member.Find(member_id); err != nil {
		this.JsonResult(6003, "用户不存在")
	}
	if member.Status == 1 {
		this.JsonResult(6004, "用户已被禁用")
	}

	relationship, err := models.NewRelationship().UpdateRoleId(book.BookId, member_id, role)

	if err != nil {
		logs.Error("变更用户在项目中的权限 => ", err)
		this.JsonResult(6005, err.Error())
	}

	memberRelationshipResult := models.NewMemberRelationshipResult().FromMember(member)
	memberRelationshipResult.RoleId = relationship.RoleId
	memberRelationshipResult.RelationshipId = relationship.RelationshipId
	memberRelationshipResult.BookId = book.BookId
	memberRelationshipResult.ResolveRoleName()

	this.JsonResult(0, "ok", memberRelationshipResult)
}

// 删除参与者.
func (this *BookMemberController) RemoveMember() {
	identify := this.GetString("identify")
	member_id, _ := this.GetInt("member_id", 0)

	if identify == "" || member_id <= 0 {
		this.JsonResult(6001, "参数错误")
	}
	if member_id == this.Member.MemberId {
		this.JsonResult(6006, "不能删除自己")
	}
	book, err := models.NewBookResult().FindByIdentify(identify, this.Member.MemberId)

	if err != nil {
		if err == models.ErrPermissionDenied {
			this.JsonResult(403, "权限不足")
		}
		if err == orm.ErrNoRows {
			this.JsonResult(404, "项目不存在")
		}
		this.JsonResult(6002, err.Error())
	}
	//如果不是创始人也不是管理员则不能操作
	if book.RoleId != conf.BookFounder && book.RoleId != conf.BookAdmin {
		this.JsonResult(403, "权限不足")
	}
	err = models.NewRelationship().DeleteByBookIdAndMemberId(book.BookId, member_id)

	if err != nil {
		this.JsonResult(6007, err.Error())
	}
	this.JsonResult(0, "ok")
}

func (this *BookMemberController) IsPermission() (*models.BookResult, error) {
	identify := this.GetString("identify")
	book, err := models.NewBookResult().FindByIdentify(identify, this.Member.MemberId)

	if err != nil {
		if err == models.ErrPermissionDenied {
			return book, errors.New("权限不足")
		}
		if err == orm.ErrNoRows {
			return book, errors.New("项目不存在")
		}
		return book, err
	}
	if book.RoleId != conf.BookAdmin && book.RoleId != conf.BookFounder {
		return book, errors.New("权限不足")
	}
	return book, nil
}
