package controllers

import (
	"encoding/json"
	"html/template"
	"net/http"
	"regexp"
	"strings"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/araddon/dateparse"

	"path/filepath"
	"strconv"

	"fmt"

	"time"

	"os"

	"github.com/TruthHun/BookStack/conf"
	"github.com/TruthHun/BookStack/models"
	"github.com/TruthHun/BookStack/models/store"
	"github.com/TruthHun/BookStack/utils"
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/logs"
	"github.com/astaxie/beego/orm"
)

type ManagerController struct {
	BaseController
}

func (this *ManagerController) Prepare() {
	this.BaseController.Prepare()
	if !this.Member.IsAdministrator() {
		this.Abort("404")
	}
}

func (this *ManagerController) Index() {
	this.TplName = "manager/index.html"
	this.Data["Model"] = models.NewDashboard().Query()
	this.GetSeoByPage("manage_dashboard", map[string]string{
		"title":       "仪表盘 - " + this.Sitename,
		"keywords":    "仪表盘",
		"description": this.Sitename + "专注于文档在线写作、协作、分享、阅读与托管，让每个人更方便地发布、分享和获得知识。",
	})
	this.Data["Installed"] = utils.GetInstalledDependencies()
	this.Data["IsDashboard"] = true
}

func (this *ManagerController) UpdateMemberNoRank() {
	memberId, _ := this.GetInt("member_id", 0)
	noRankInt, _ := this.GetInt("no_rank", 0)

	if memberId <= 0 {
		this.JsonResult(6001, "参数错误")
	}
	noRank := false
	if noRankInt == 1 {
		noRank = true
	}
	member := models.NewMember()

	if _, err := member.Find(memberId); err != nil {
		this.JsonResult(6002, "用户不存在")
	}
	if member.MemberId == this.Member.MemberId {
		this.JsonResult(6004, "不能变更自己的状态")
	}
	if member.Role == conf.MemberSuperRole {
		this.JsonResult(6005, "不能变更超级管理员的状态")
	}
	member.NoRank = noRank

	if err := member.Update(); err != nil {
		logs.Error("", err)
		this.JsonResult(6003, "用户状态设置失败")
	}
	this.JsonResult(0, "ok", member)
}

// 用户列表.
func (this *ManagerController) Users() {
	this.TplName = "manager/users.html"
	this.Data["IsUsers"] = true
	wd := this.GetString("wd")
	role, err := this.GetInt("role")
	if err != nil {
		role = -1
	}
	pageIndex, _ := this.GetInt("page", 0)
	this.GetSeoByPage("manage_users", map[string]string{
		"title":       "用户管理 - " + this.Sitename,
		"keywords":    "用户管理",
		"description": this.Sitename + "专注于文档在线写作、协作、分享、阅读与托管，让每个人更方便地发布、分享和获得知识。",
	})

	members, totalCount, err := models.NewMember().FindToPager(pageIndex, conf.PageSize, wd, role)

	if err != nil {
		this.Data["ErrorMessage"] = err.Error()
		return
	}

	if totalCount > 0 {
		this.Data["PageHtml"] = utils.NewPaginations(conf.RollPage, int(totalCount), conf.PageSize, pageIndex, beego.URLFor("ManagerController.Users"), "")
	} else {
		this.Data["PageHtml"] = ""
	}

	b, err := json.Marshal(members)

	if err != nil {
		this.Data["Result"] = template.JS("[]")
	} else {
		this.Data["Result"] = template.JS(string(b))
	}
	this.Data["Role"] = role
	this.Data["Wd"] = wd
}

// 标签管理.
func (this *ManagerController) Tags() {
	this.TplName = "manager/tags.html"
	this.Data["IsTag"] = true
	size := 150
	wd := this.GetString("wd")
	pageIndex, _ := this.GetInt("page", 1)
	tags, totalCount, err := models.NewLabel().FindToPager(pageIndex, size, wd)
	if err != nil {
		this.Data["ErrorMessage"] = err.Error()
		return
	}
	if totalCount > 0 {
		this.Data["PageHtml"] = utils.NewPaginations(conf.RollPage, int(totalCount), size, pageIndex, beego.URLFor("ManagerController.Tags"), "")
	} else {
		this.Data["PageHtml"] = ""
	}
	this.Data["Total"] = totalCount
	this.Data["Tags"] = tags
	this.Data["Wd"] = wd
}

func (this *ManagerController) AddTags() {
	tags := this.GetString("tags")
	if tags != "" {
		tags = strings.Join(strings.Split(tags, "\n"), ",")
		models.NewLabel().InsertOrUpdateMulti(tags)
	}
	this.JsonResult(0, "新增标签成功")
}

func (this *ManagerController) DelTags() {
	id, _ := this.GetInt("id")
	if id > 0 {
		orm.NewOrm().QueryTable(models.NewLabel()).Filter("label_id", id).Delete()
	}
	this.JsonResult(0, "标签删除成功")
}

// 添加用户.
func (this *ManagerController) CreateMember() {

	account := strings.TrimSpace(this.GetString("account"))
	nickname := strings.TrimSpace(this.GetString("nickname"))
	password1 := strings.TrimSpace(this.GetString("password1"))
	password2 := strings.TrimSpace(this.GetString("password2"))
	email := strings.TrimSpace(this.GetString("email"))
	phone := strings.TrimSpace(this.GetString("phone"))
	role, _ := this.GetInt("role", 1)
	//status, _ := this.GetInt("status", 0)

	if ok, err := regexp.MatchString(conf.RegexpAccount, account); account == "" || !ok || err != nil {
		this.JsonResult(6001, "账号只能由英文字母数字组成，且在3-50个字符")
	}
	if l := strings.Count(nickname, "") - 1; l < 2 || l > 20 {
		this.JsonResult(6001, "昵称限制在2-20个字符")
	}
	if l := strings.Count(password1, ""); password1 == "" || l > 50 || l < 6 {
		this.JsonResult(6002, "密码必须在6-50个字符之间")
	}
	if password1 != password2 {
		this.JsonResult(6003, "确认密码不正确")
	}
	if ok, err := regexp.MatchString(conf.RegexpEmail, email); !ok || err != nil || email == "" {
		this.JsonResult(6004, "邮箱格式不正确")
	}
	if role != 0 && role != 1 && role != 2 {
		role = 1
	}

	member := models.NewMember()

	if _, err := member.FindByAccount(account); err == nil && member.MemberId > 0 {
		this.JsonResult(6005, "账号已存在")
	}

	member.Account = account
	member.Password = password1
	member.Role = role
	member.Avatar = conf.GetDefaultAvatar()
	member.CreateAt = this.Member.MemberId
	member.Email = email
	member.Nickname = nickname
	if phone != "" {
		member.Phone = phone
	}

	if err := member.Add(); err != nil {
		beego.Error(err.Error())
		this.JsonResult(6006, "注册失败，可能昵称已存在")
	}

	this.JsonResult(0, "ok", member)
}

// 更新用户状态.
func (this *ManagerController) UpdateMemberStatus() {

	memberId, _ := this.GetInt("member_id", 0)
	status, _ := this.GetInt("status", 0)

	if memberId <= 0 {
		this.JsonResult(6001, "参数错误")
	}
	if status != 0 && status != 1 {
		status = 0
	}
	member := models.NewMember()

	if _, err := member.Find(memberId); err != nil {
		this.JsonResult(6002, "用户不存在")
	}
	if member.MemberId == this.Member.MemberId {
		this.JsonResult(6004, "不能变更自己的状态")
	}
	if member.Role == conf.MemberSuperRole {
		this.JsonResult(6005, "不能变更超级管理员的状态")
	}
	member.Status = status

	if err := member.Update(); err != nil {
		logs.Error("", err)
		this.JsonResult(6003, "用户状态设置失败")
	}
	this.JsonResult(0, "ok", member)
}

// 变更用户权限.
func (this *ManagerController) ChangeMemberRole() {

	memberId, _ := this.GetInt("member_id", 0)
	role, _ := this.GetInt("role", 0)
	if memberId <= 0 {
		this.JsonResult(6001, "参数错误")
	}
	if role != conf.MemberAdminRole && role != conf.MemberGeneralRole && role != conf.MemberEditorRole {
		this.JsonResult(6001, "用户权限不正确")
	}
	member := models.NewMember()

	if _, err := member.Find(memberId); err != nil {
		this.JsonResult(6002, "用户不存在")
	}
	if member.MemberId == this.Member.MemberId {
		this.JsonResult(6004, "不能变更自己的权限")
	}
	if member.Role == conf.MemberSuperRole {
		this.JsonResult(6005, "不能变更超级管理员的权限")
	}
	member.Role = role

	if err := member.Update(); err != nil {
		logs.Error("", err)
		this.JsonResult(6003, "用户权限设置失败")
	}
	member.ResolveRoleName()
	this.JsonResult(0, "ok", member)
}

// 编辑用户信息.
func (this *ManagerController) EditMember() {

	memberId, _ := this.GetInt(":id", 0)
	if memberId <= 0 {
		this.Abort("404")
	}

	member, err := models.NewMember().Find(memberId)
	if err != nil {
		beego.Error(err)
		this.Abort("404")
	}

	if this.Ctx.Input.IsPost() {
		password1 := this.GetString("password1")
		password2 := this.GetString("password2")
		email := this.GetString("email")
		phone := this.GetString("phone")
		description := this.GetString("description")
		member.Email = email
		member.Phone = phone
		member.Description = description
		if password1 != "" && password2 != password1 {
			this.JsonResult(6001, "确认密码不正确")
		}
		if password1 != "" && member.AuthMethod != conf.AuthMethodLDAP {
			member.Password = password1
		}
		if err := member.Valid(password1 == ""); err != nil {
			this.JsonResult(6002, err.Error())
		}
		if password1 != "" {
			password, err := utils.PasswordHash(password1)
			if err != nil {
				beego.Error(err)
				this.JsonResult(6003, "对用户密码加密时出错")
			}
			member.Password = password
		}
		if err := member.Update(); err != nil {
			beego.Error(err)
			this.JsonResult(6004, "保存失败")
		}
		this.JsonResult(0, "ok")
	}

	this.GetSeoByPage("manage_users_edit", map[string]string{
		"title":       "用户编辑 - " + this.Sitename,
		"keywords":    "用户标记",
		"description": this.Sitename + "专注于文档在线写作、协作、分享、阅读与托管，让每个人更方便地发布、分享和获得知识。",
	})
	this.Data["IsUsers"] = true
	this.Data["Model"] = member
	this.TplName = "manager/edit_users.html"
}

// 删除一个用户，并将该用户的所有信息转移到超级管理员上.
func (this *ManagerController) DeleteMember() {
	memberId, _ := this.GetInt("id", 0)
	if memberId <= 0 {
		this.JsonResult(404, "参数错误")
	}

	member, err := models.NewMember().Find(memberId)

	if err != nil {
		beego.Error(err)
		this.JsonResult(500, "用户不存在")
	}
	if member.Role == conf.MemberSuperRole {
		this.JsonResult(500, "不能删除超级管理员")
	}
	superMember, err := models.NewMember().FindByFieldFirst("role", 0)

	if err != nil {
		beego.Error(err)
		this.JsonResult(5001, "未能找到超级管理员")
	}

	err = models.NewMember().Delete(memberId, superMember.MemberId)
	if err != nil {
		beego.Error(err)
		this.JsonResult(5002, "删除失败")
	}

	this.JsonResult(0, "ok")
}

// 书籍列表.
func (this *ManagerController) Books() {
	pageIndex, _ := this.GetInt("page", 1)
	private, _ := this.GetInt("private")
	wd := this.GetString("wd")

	size := conf.PageSize

	books, totalCount, _ := models.NewBookResult().FindToPager(pageIndex, size, private, wd)

	if totalCount > 0 {
		this.Data["PageHtml"] = utils.NewPaginations(conf.RollPage, totalCount, size, pageIndex, beego.URLFor("ManagerController.Books"), fmt.Sprintf("&private=%v&wd=%v", private, wd))
	} else {
		this.Data["PageHtml"] = ""
	}

	this.Data["Lists"] = books
	this.Data["Wd"] = wd
	this.Data["IsBooks"] = true
	this.GetSeoByPage("manage_project_list", map[string]string{
		"title":       "书籍管理 - " + this.Sitename,
		"keywords":    "书籍管理",
		"description": this.Sitename + "专注于文档在线写作、协作、分享、阅读与托管，让每个人更方便地发布、分享和获得知识。",
	})
	this.Data["Private"] = private
	this.TplName = "manager/books.html"
	this.Data["Versions"], _ = models.NewVersion().Lists(1, 10000, "")
}

// 编辑书籍.
func (this *ManagerController) EditBook() {

	identify := this.GetString(":key")
	if identify == "" {
		this.Abort("404")
	}

	book, err := models.NewBook().FindByFieldFirst("identify", identify)
	if err != nil {
		this.Abort("404")
	}

	if this.Ctx.Input.IsPost() {
		bookName := strings.TrimSpace(this.GetString("book_name"))
		description := strings.TrimSpace(this.GetString("description", ""))
		commentStatus := this.GetString("comment_status")
		tag := strings.TrimSpace(this.GetString("label"))
		orderIndex, _ := this.GetInt("order_index", 0)
		pin, _ := this.GetInt("pin", 0)

		if strings.Count(description, "") > 500 {
			this.JsonResult(6004, "书籍描述不能大于500字")
		}
		if commentStatus != "open" && commentStatus != "closed" && commentStatus != "group_only" && commentStatus != "registered_only" {
			commentStatus = "closed"
		}
		if tag != "" {
			tags := strings.Split(tag, ";")
			if len(tags) > 10 {
				this.JsonResult(6005, "最多允许添加10个标签")
			}
		}

		book.BookName = bookName
		book.Description = description
		book.CommentStatus = commentStatus
		book.Label = tag
		book.OrderIndex = orderIndex
		book.Pin = pin

		if err := book.Update(); err != nil {
			this.JsonResult(6006, "保存失败")
		}

		go func() {
			es := models.ElasticSearchData{
				Id:       book.BookId,
				BookId:   0,
				Title:    book.BookName,
				Keywords: book.Label,
				Content:  book.Description,
				Vcnt:     book.Vcnt,
				Private:  book.PrivatelyOwned,
			}
			client := models.NewElasticSearchClient()
			if errSearch := client.BuildIndex(es); errSearch != nil && client.On {
				beego.Error(errSearch.Error())
			}
		}()

		this.JsonResult(0, "ok")
	}
	if book.PrivateToken != "" {
		book.PrivateToken = this.BaseUrl() + beego.URLFor("DocumentController.Index", ":key", book.Identify, "token", book.PrivateToken)
	}
	this.Data["Model"] = book

	this.GetSeoByPage("manage_project_edit", map[string]string{
		"title":       "书籍设置 - " + this.Sitename,
		"keywords":    "书籍设置",
		"description": this.Sitename + "专注于文档在线写作、协作、分享、阅读与托管，让每个人更方便地发布、分享和获得知识。",
	})
	this.TplName = "manager/edit_book.html"
}

// 删除书籍.
func (this *ManagerController) DeleteBook() {
	var bookIds []int
	beego.Debug(this.Ctx.Request.Form)
	if ids, ok := this.Ctx.Request.Form["book_id"]; ok {
		for _, id := range ids {
			if v, _ := strconv.Atoi(id); v > 0 {
				bookIds = append(bookIds, v)
			}
		}
	}
	if len(bookIds) <= 0 {
		this.JsonResult(6001, "参数错误")
	}

	//用户密码
	pwd := this.GetString("password")
	if m, err := models.NewMember().Login(this.Member.Account, pwd); err != nil || m.MemberId == 0 {
		this.JsonResult(1, "书籍删除失败，您的登录密码不正确")
	}
	identify := strings.TrimSpace(this.GetString("identify"))
	book := models.NewBook()
	client := models.NewElasticSearchClient()
	for _, bookID := range bookIds {
		if identify != "" {
			if b, _ := book.FindByIdentify(identify, "book_id"); b.BookId != bookID {
				this.JsonResult(6002, "书籍标识输入不正确")
			}
		}

		err := book.ThoroughDeleteBook(bookID)
		if err == orm.ErrNoRows {
			this.JsonResult(6002, "书籍不存在")
		}
		if err != nil {
			logs.Error("DeleteBook => ", err)
			this.JsonResult(6003, "删除失败")
		}
		if errDel := client.DeleteIndex(bookID, true); errDel != nil && client.On {
			beego.Error(errDel.Error())
		}
	}
	go models.CountCategory()
	this.JsonResult(0, "书籍删除成功")
}

// CreateToken 创建访问来令牌.
func (this *ManagerController) CreateToken() {

	if this.forbidGeneralRole() {
		this.JsonResult(6001, "您的角色非作者和管理员，无法创建访问令牌")
	}

	action := this.GetString("action")
	identify := this.GetString("identify")

	book, err := models.NewBook().FindByFieldFirst("identify", identify)
	if err != nil {
		this.JsonResult(6001, "书籍不存在")
	}

	if action == "create" {
		if book.PrivatelyOwned == 0 {
			this.JsonResult(6001, "公开书籍不能创建阅读令牌")
		}

		book.PrivateToken = string(utils.Krand(conf.GetTokenSize(), utils.KC_RAND_KIND_ALL))
		if err := book.Update(); err != nil {
			logs.Error("生成阅读令牌失败 => ", err)
			this.JsonResult(6003, "生成阅读令牌失败")
		}
		this.JsonResult(0, "ok", this.BaseUrl()+beego.URLFor("DocumentController.Index", ":key", book.Identify, "token", book.PrivateToken))
	}

	book.PrivateToken = ""
	if err := book.Update(); err != nil {
		beego.Error("CreateToken => ", err)
		this.JsonResult(6004, "删除令牌失败")
	}
	this.JsonResult(0, "ok", "")
}

func (this *ManagerController) Setting() {
	tab := this.GetString("tab", "basic")
	options, err := models.NewOption().All()
	if err != nil {
		this.Abort("404")
	}

	if this.Ctx.Input.IsPost() {
		for _, item := range options {
			if _, ok := this.Ctx.Request.PostForm[item.OptionName]; ok {
				item.OptionValue = this.GetString(item.OptionName)
				item.InsertOrUpdate()
			}
		}

		if err := models.NewElasticSearchClient().Init(); err != nil {
			this.JsonResult(1, err.Error())
		}

		models.NewSign().UpdateSignRule()
		models.NewReadRecord().UpdateReadingRule()
		this.JsonResult(0, "ok")
	}

	for _, item := range options {
		if item.OptionName == "APP_PAGE" {
			this.Data["APP_PAGE"] = item.OptionValue
			this.Data["M_APP_PAGE"] = item
		} else {
			this.Data[item.OptionName] = item
		}
	}
	this.Data["SITE_TITLE"] = this.Option["SITE_NAME"]
	this.Data["Tab"] = tab
	this.Data["IsSetting"] = true
	this.Data["SeoTitle"] = "配置管理"
	this.TplName = "manager/setting.html"
}

// Transfer 转让书籍.
func (this *ManagerController) Transfer() {
	account := this.GetString("account")
	if account == "" {
		this.JsonResult(6004, "接受者账号不能为空")
	}

	member, err := models.NewMember().FindByAccount(account)
	if err != nil {
		beego.Error("FindByAccount => ", err)
		this.JsonResult(6005, "接受用户不存在")
	}

	if member.Status != 0 {
		this.JsonResult(6006, "接受用户已被禁用")
	}

	if !this.Member.IsAdministrator() {
		this.Abort("404")
	}

	identify := this.GetString("identify")

	book, err := models.NewBook().FindByFieldFirst("identify", identify)
	if err != nil {
		this.JsonResult(6001, err.Error())
	}

	rel, err := models.NewRelationship().FindFounder(book.BookId)
	if err != nil {
		beego.Error("FindFounder => ", err)
		this.JsonResult(6009, "查询书籍创始人失败")
	}

	if member.MemberId == rel.MemberId {
		this.JsonResult(6007, "不能转让给自己")
	}

	err = models.NewRelationship().Transfer(book.BookId, rel.MemberId, member.MemberId)
	if err != nil {
		beego.Error("Transfer => ", err)
		this.JsonResult(6008, err.Error())
	}
	this.JsonResult(0, "ok")
}

func (this *ManagerController) Comments() {
	status := this.GetString("status", "0")
	statusNum, _ := strconv.Atoi(status)
	p, _ := this.GetInt("page", 1)
	size, _ := this.GetInt("size", 10)
	m := models.NewComments()
	opt := models.CommentOpt{}
	if status == "" {
		this.Data["Comments"], _ = m.Comments(p, size, opt)
	} else {
		opt.Status = []int{statusNum}
		this.Data["Comments"], _ = m.Comments(p, size, opt)
	}
	this.Data["IsComments"] = true
	this.Data["Status"] = status
	count, _ := m.Count(0, statusNum)
	this.Data["Count"] = count
	if count > 0 {
		html := utils.GetPagerHtml(this.Ctx.Request.RequestURI, p, size, int(count))
		this.Data["PageHtml"] = html
	}
	this.TplName = "manager/comments.html"
}

func (this *ManagerController) ClearComments() {
	uid, _ := this.GetInt("uid")
	if uid > 0 {
		models.NewComments().ClearComments(uid)
	}
	this.JsonResult(0, "清除成功")
}

func (this *ManagerController) DeleteComment() {
	id, _ := this.GetInt("id")
	if id > 0 {
		models.NewComments().DeleteComment(id)
	}
	this.JsonResult(0, "删除成功")
}

func (this *ManagerController) SetCommentStatus() {
	id, _ := this.GetInt("id")
	status, _ := this.GetInt("value")
	field := this.GetString("field")
	if id > 0 && field == "status" {
		if err := models.NewComments().SetCommentStatus(id, status); err != nil {
			this.JsonResult(1, err.Error())
		}
	}
	this.JsonResult(0, "设置成功")
}

// 设置书籍私有状态.
func (this *ManagerController) PrivatelyOwned() {
	status := this.GetString("status")
	identify := this.GetString("identify")

	if status != "open" && status != "close" {
		this.JsonResult(6003, "参数错误")
	}

	state := 0
	if status == "open" {
		state = 0
	} else {
		state = 1
	}

	if !this.Member.IsAdministrator() {
		this.Abort("404")
	}

	book, err := models.NewBook().FindByFieldFirst("identify", identify)
	if err != nil {
		this.JsonResult(6001, err.Error())
	}

	book.PrivatelyOwned = state
	beego.Info("", state, status)

	err = book.Update()
	if err != nil {
		beego.Error("PrivatelyOwned => ", err)
		this.JsonResult(6004, "保存失败")
	}

	go func() {
		models.CountCategory()
		public := true
		if state == 1 {
			public = false
		}
		client := models.NewElasticSearchClient()
		if errSet := client.SetBookPublic(book.BookId, public); errSet != nil && client.On {
			beego.Error(errSet.Error())
		}
	}()

	this.JsonResult(0, "ok")
}

// 附件列表.
func (this *ManagerController) AttachList() {

	pageIndex, _ := this.GetInt("page", 1)

	attachList, totalCount, err := models.NewAttachment().FindToPager(pageIndex, conf.PageSize)
	if err != nil {
		this.Abort("404")
	}

	if totalCount > 0 {
		html := utils.GetPagerHtml(this.Ctx.Request.RequestURI, pageIndex, conf.PageSize, int(totalCount))
		this.Data["PageHtml"] = html
	} else {
		this.Data["PageHtml"] = ""
	}

	this.Data["Lists"] = attachList
	this.Data["IsAttach"] = true
	this.TplName = "manager/attach_list.html"
}

// 附件详情.
func (this *ManagerController) AttachDetailed() {

	attachId, _ := strconv.Atoi(this.Ctx.Input.Param(":id"))
	if attachId <= 0 {
		this.Abort("404")
	}

	attach, err := models.NewAttachmentResult().Find(attachId)
	if err != nil {
		beego.Error("AttachDetailed => ", err)
		if err == orm.ErrNoRows {
			this.Abort("404")
		}
		this.Abort("404")
	}

	attach.HttpPath = this.BaseUrl() + attach.HttpPath
	attach.IsExist = utils.FileExists(attach.FilePath)
	this.Data["Model"] = attach
	this.TplName = "manager/attach_detailed.html"
}

// 删除附件.
func (this *ManagerController) AttachDelete() {

	attachId, _ := this.GetInt("attach_id")
	if attachId <= 0 {
		this.Abort("404")
	}

	attach, _ := models.NewAttachment().Find(attachId)
	if attach.AttachmentId == 0 {
		this.JsonResult(0, "ok")
	}

	obj := strings.TrimLeft(attach.FilePath, "./")
	switch utils.StoreType {
	case utils.StoreOss:
		store.ModelStoreOss.DelFromOss(obj)
		if bucket, err := store.ModelStoreOss.GetBucket(); err == nil {
			bucket.SetObjectACL(obj, oss.ACLPrivate)
		}
	case utils.StoreLocal:
		os.Remove(obj)
	}
	attach.Delete()
	this.JsonResult(0, "ok")
}

// SEO管理
func (this *ManagerController) Seo() {
	o := orm.NewOrm()
	if this.Ctx.Input.IsPost() { //SEO更新
		rows, err := o.QueryTable(models.TableSeo).Filter("id", this.GetString("id")).Update(map[string]interface{}{
			this.GetString("field"): this.GetString("value"),
		})
		if err != nil {
			beego.Error(err.Error())
			this.JsonResult(1, "更新失败，请求错误")
		}
		if rows > 0 {
			this.JsonResult(0, "更新成功")
		}
		this.JsonResult(1, "更新失败，您未对内容做更改")
	}

	//SEO展示
	var seos []models.Seo
	o.QueryTable(models.TableSeo).All(&seos)
	this.Data["Lists"] = seos
	this.Data["IsManagerSeo"] = true
	this.TplName = "manager/seo.html"
}

func (this *ManagerController) UpdateAds() {
	id, _ := this.GetInt("id")
	field := this.GetString("field")
	value := this.GetString("value")
	if field == "" {
		this.JsonResult(1, "字段不能为空")
	}
	_, err := orm.NewOrm().QueryTable(models.NewAdsCont()).Filter("id", id).Update(orm.Params{field: value})
	if err != nil {
		this.JsonResult(1, err.Error())
	}
	go models.UpdateAdsCache()
	this.JsonResult(0, "操作成功")
}

func (this *ManagerController) DelAds() {
	id, _ := this.GetInt("id")
	_, err := orm.NewOrm().QueryTable(models.NewAdsCont()).Filter("id", id).Delete()
	if err != nil {
		this.JsonResult(1, err.Error())
	}
	go models.UpdateAdsCache()
	this.JsonResult(0, "删除成功")
}

// 广告管理
func (this *ManagerController) Ads() {
	if this.Ctx.Request.Method == http.MethodPost {
		pid, _ := this.GetInt("pid")
		if pid <= 0 {
			this.JsonResult(1, "请选择广告位")
		}
		ads := &models.AdsCont{
			Title:  this.GetString("title"),
			Code:   this.GetString("code"),
			Status: true,
			Pid:    pid,
		}
		start, err := dateparse.ParseAny(this.GetString("start"))
		if err != nil {
			start = time.Now()
		}
		end, err := dateparse.ParseAny(this.GetString("end"))
		if err != nil {
			end = time.Now().Add(24 * time.Hour * 730)
		}
		ads.Start = int(start.Unix())
		ads.End = int(end.Unix())
		_, err = orm.NewOrm().Insert(ads)
		if err != nil {
			this.JsonResult(1, err.Error())
		}
		go models.UpdateAdsCache()
		this.JsonResult(0, "新增广告成功")
	} else {
		layout := "2006-01-02"
		this.Data["Mobile"] = this.GetString("mobile", "0")
		this.Data["Positions"] = models.NewAdsCont().GetPositions()
		this.Data["Lists"] = models.NewAdsCont().Lists(this.GetString("mobile", "0") == "1")
		this.Data["IsAds"] = true
		this.Data["Now"] = time.Now().Format(layout)
		this.Data["Next"] = time.Now().Add(time.Hour * 24 * 730).Format(layout)
		this.TplName = "manager/ads.html"
	}
}

// 更行书籍书籍的排序
func (this *ManagerController) UpdateBookSort() {
	bookId, _ := this.GetInt("book_id")
	orderIndex, _ := this.GetInt("value")
	if bookId > 0 {
		if _, err := orm.NewOrm().QueryTable("md_books").Filter("book_id", bookId).Update(orm.Params{
			"order_index": orderIndex,
		}); err != nil {
			this.JsonResult(1, err.Error())
		}
	}
	this.JsonResult(0, "排序更新成功")
}

func (this *ManagerController) Sitemap() {
	baseUrl := this.Ctx.Input.Scheme() + "://" + this.Ctx.Request.Host
	if host := beego.AppConfig.String("sitemap_host"); len(host) > 0 {
		baseUrl = this.Ctx.Input.Scheme() + "://" + host
	}
	go models.SitemapUpdate(baseUrl)
	this.JsonResult(0, "站点地图更新提交成功，已交由后台执行更新，请耐心等待。")
}

// 分类管理
func (this *ManagerController) Category() {
	Model := new(models.Category)
	if strings.ToLower(this.Ctx.Request.Method) == "post" {
		//新增分类
		pid, _ := this.GetInt("pid")
		if err := Model.AddCates(pid, this.GetString("cates")); err != nil {
			this.JsonResult(1, "新增失败："+err.Error())
		}
		this.JsonResult(0, "新增成功")
	}

	//查询所有分类
	cates, err := Model.GetCates(-1, -1)
	if err != nil {
		beego.Error(err)
	}

	var parents []models.Category
	for idx, item := range cates {
		if strings.TrimSpace(item.Icon) == "" { //赋值为默认图片
			item.Icon = "/static/images/icon.png"
		} else {
			item.Icon = utils.ShowImg(item.Icon)
		}
		if item.Pid == 0 {
			parents = append(parents, item)
		}
		cates[idx] = item
	}

	this.Data["Parents"] = parents
	this.Data["Cates"] = cates
	this.Data["IsCategory"] = true
	this.TplName = "manager/category.html"
}

// 更新分类字段内容
func (this *ManagerController) UpdateCate() {
	field := this.GetString("field")
	val := this.GetString("value")
	id, _ := this.GetInt("id")
	if err := new(models.Category).UpdateByField(id, field, val); err != nil {
		this.JsonResult(1, "更新失败："+err.Error())
	}
	this.JsonResult(0, "更新成功")
}

// 删除分类
func (this *ManagerController) DelCate() {
	var err error
	if id, _ := this.GetInt("id"); id > 0 {
		err = new(models.Category).Del(id)
	}
	if err != nil {
		this.JsonResult(1, err.Error())
	}
	this.JsonResult(0, "删除成功")
}

// 更新分类的图标
func (this *ManagerController) UpdateCateIcon() {
	var err error
	id, _ := this.GetInt("id")
	if id == 0 {
		this.JsonResult(1, "参数不正确")
	}

	data := make(map[string]interface{})
	model := new(models.Category)
	if cate := model.Find(id); cate.Id > 0 {
		cate.Icon = strings.TrimLeft(cate.Icon, "/")
		f, h, err1 := this.GetFile("icon")
		if err1 != nil {
			err = err1
		}
		defer f.Close()

		tmpFile := fmt.Sprintf("uploads/icons/%v%v"+filepath.Ext(h.Filename), id, time.Now().Unix())
		os.MkdirAll(filepath.Dir(tmpFile), os.ModePerm)
		if err = this.SaveToFile("icon", tmpFile); err == nil {
			switch utils.StoreType {
			case utils.StoreOss:
				store.ModelStoreOss.MoveToOss(tmpFile, tmpFile, true, false)
				store.ModelStoreOss.DelFromOss(cate.Icon)
				data["icon"] = utils.ShowImg(tmpFile)
			case utils.StoreLocal:
				store.ModelStoreLocal.DelFiles(cate.Icon)
				data["icon"] = "/" + tmpFile
			}
			err = model.UpdateByField(cate.Id, "icon", "/"+tmpFile)
		}
	}

	if err != nil {
		this.JsonResult(1, err.Error())
	}
	this.JsonResult(0, "更新成功", data)
}

// 友情链接
func (this *ManagerController) FriendLink() {
	this.Data["SeoTitle"] = "友链管理"
	this.Data["Links"] = new(models.FriendLink).GetList(true)
	this.Data["IsFriendlink"] = true
	this.TplName = "manager/friendlink.html"
}

// 添加友链
func (this *ManagerController) AddFriendlink() {
	if err := new(models.FriendLink).Add(this.GetString("title"), this.GetString("link")); err != nil {
		this.JsonResult(1, "新增友链失败:"+err.Error())
	}
	this.JsonResult(0, "新增友链成功")
}

// 更新友链
func (this *ManagerController) UpdateFriendlink() {
	id, _ := this.GetInt("id")
	if err := new(models.FriendLink).Update(id, this.GetString("field"), this.GetString("value")); err != nil {
		this.JsonResult(1, "操作失败："+err.Error())
	}
	this.JsonResult(0, "操作成功")
}

// 删除友链
func (this *ManagerController) DelFriendlink() {
	id, _ := this.GetInt("id")
	if err := new(models.FriendLink).Del(id); err != nil {
		this.JsonResult(1, "删除失败："+err.Error())
	}
	this.JsonResult(0, "删除成功")
}

// 重建全量索引
func (this *ManagerController) RebuildAllIndex() {
	go models.NewElasticSearchClient().RebuildAllIndex()
	this.JsonResult(0, "提交成功，请耐心等待")
}

func (this *ManagerController) Banners() {
	this.Data["SeoTitle"] = "横幅管理"
	this.TplName = "manager/banners.html"
	this.Data["Banners"], _ = models.NewBanner().All()
	this.Data["IsBanner"] = true
}

func (this *ManagerController) DeleteBanner() {
	id, _ := this.GetInt("id")
	if id > 0 {
		err := models.NewBanner().Delete(id)
		if err != nil {
			this.JsonResult(1, err.Error())
		}
	}
	this.JsonResult(0, "删除成功")
}

func (this *ManagerController) UpdateBanner() {
	id, _ := this.GetInt("id")
	field := this.GetString("field")
	value := this.GetString("value")
	if id > 0 {
		err := models.NewBanner().Update(id, field, value)
		if err != nil {
			this.JsonResult(1, err.Error())
		}
	}
	this.JsonResult(0, "更新成功")
}

func (this *ManagerController) UploadBanner() {
	f, h, err := this.GetFile("image")
	if err != nil {
		this.JsonResult(1, err.Error())
	}
	ext := strings.ToLower(filepath.Ext(strings.ToLower(h.Filename)))
	tmpFile := fmt.Sprintf("uploads/tmp/banner-%v-%v%v", this.Member.MemberId, time.Now().Unix(), ext)
	destFile := fmt.Sprintf("uploads/banners/%v.%v%v", this.Member.MemberId, time.Now().Unix(), ext)
	defer func() {
		f.Close()
		os.Remove(tmpFile)
		if err != nil {
			utils.DeleteFile(destFile)
		}
	}()

	os.MkdirAll(filepath.Dir(tmpFile), os.ModePerm)
	err = this.SaveToFile("image", tmpFile)
	if err != nil {
		this.JsonResult(1, err.Error())
	}
	err = utils.UploadFile(tmpFile, destFile)
	if err != nil {
		this.JsonResult(1, err.Error())
	}
	banner := &models.Banner{
		Image:     "/" + destFile,
		Type:      this.GetString("type"),
		Title:     this.GetString("title"),
		Link:      this.GetString("link"),
		Status:    true,
		CreatedAt: time.Now(),
	}
	banner.Sort, _ = this.GetInt("sort")
	_, err = orm.NewOrm().Insert(banner)
	if err != nil {
		this.JsonResult(1, err.Error())
	}
	this.JsonResult(0, "横幅上传成功")
}

func (this *ManagerController) SubmitBook() {
	this.TplName = "manager/submit_book.html"
	m := models.NewSubmitBooks()
	page, _ := this.GetInt("page", 1)
	size, _ := this.GetInt("size", 100)
	books, total, _ := m.Lists(page, size)
	if total > 0 {
		this.Data["PageHtml"] = utils.NewPaginations(conf.RollPage, int(total), size, page, beego.URLFor("ManagerController.SubmitBook"), "")
	} else {
		this.Data["PageHtml"] = ""
	}
	this.Data["Books"] = books
	this.Data["IsSubmitBook"] = true
}

func (this *ManagerController) DeleteSubmitBook() {
	id, _ := this.GetInt("id")
	orm.NewOrm().QueryTable(models.NewSubmitBooks()).Filter("id", id).Delete()
	this.JsonResult(0, "删除成功")
}

func (this *ManagerController) UpdateSubmitBook() {
	field := this.GetString("field")
	value := this.GetString("value")
	id, _ := this.GetInt("id")
	if id > 0 {
		_, err := orm.NewOrm().QueryTable(models.NewSubmitBooks()).Filter("id", id).Update(orm.Params{field: value})
		if err != nil {
			this.JsonResult(1, err.Error())
		}
	}
	this.JsonResult(0, "更新成功")
}

// Versions 版本管理
func (this *ManagerController) Version() {
	m := models.NewVersion()
	wd := this.GetString("wd")
	page, _ := this.GetInt("page", 1)
	size := 10
	versions, total := m.Lists(page, size, wd)
	if total > 0 {
		this.Data["PageHtml"] = utils.NewPaginations(conf.RollPage, int(total), size, page, beego.URLFor("ManagerController.Version"), "&wd="+wd)
	} else {
		this.Data["PageHtml"] = ""
	}
	this.Data["Versions"] = versions
	this.Data["Wd"] = wd
	this.Data["IsManagerVersion"] = true
	this.TplName = "manager/version.html"
}

// Versions 版本管理
func (this *ManagerController) DeleteVersion() {
	id, _ := this.GetInt("id")
	if id > 0 {
		err := models.NewVersion().Delete(id)
		if err != nil {
			this.JsonResult(1, err.Error())
		}
	}
	this.JsonResult(0, "删除成功")
}

// Versions 版本管理
func (this *ManagerController) UpdateVersion() {
	id, _ := this.GetInt("id")
	field := this.GetString("field")
	value := this.GetString("value")
	if id > 0 {
		_, err := orm.NewOrm().QueryTable(models.NewVersion()).Filter("id", id).Update(orm.Params{field: value})
		if err != nil {
			this.JsonResult(1, err.Error())
		}
	}
	this.JsonResult(0, "更新成功")
}

// AddVersions 添加版本
func (this *ManagerController) AddVersions() {
	for _, title := range strings.Split(this.GetString("versions"), "\n") {
		ver := models.Version{
			Title: strings.TrimSpace(title),
		}
		ver.InsertOrUpdate()
	}
	this.JsonResult(0, "添加成功")
}
