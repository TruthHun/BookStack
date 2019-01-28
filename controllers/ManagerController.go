package controllers

import (
	"encoding/json"
	"html/template"
	"regexp"
	"strings"

	"path/filepath"
	"strconv"

	"fmt"

	"time"

	"os"

	"github.com/TruthHun/BookStack/commands"
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
		this.Abort("403")
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
	this.Data["IsDashboard"] = true
}

// 用户列表.
func (this *ManagerController) Users() {
	this.TplName = "manager/users.html"
	this.Data["IsUsers"] = true
	pageIndex, _ := this.GetInt("page", 0)
	this.GetSeoByPage("manage_users", map[string]string{
		"title":       "用户管理 - " + this.Sitename,
		"keywords":    "用户管理",
		"description": this.Sitename + "专注于文档在线写作、协作、分享、阅读与托管，让每个人更方便地发布、分享和获得知识。",
	})

	members, totalCount, err := models.NewMember().FindToPager(pageIndex, conf.PageSize)

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

//更新用户状态.
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

//变更用户权限.
func (this *ManagerController) ChangeMemberRole() {

	memberId, _ := this.GetInt("member_id", 0)
	role, _ := this.GetInt("role", 0)
	if memberId <= 0 {
		this.JsonResult(6001, "参数错误")
	}
	if role != conf.MemberAdminRole && role != conf.MemberGeneralRole {
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

//编辑用户信息.
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

//删除一个用户，并将该用户的所有信息转移到超级管理员上.
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

//项目列表.
func (this *ManagerController) Books() {

	pageIndex, _ := this.GetInt("page", 1)
	private, _ := this.GetInt("private")

	books, totalCount, err := models.NewBookResult().FindToPager(pageIndex, conf.PageSize, private)
	if err != nil {
		this.Abort("500")
	}

	if totalCount > 0 {
		this.Data["PageHtml"] = utils.NewPaginations(conf.RollPage, totalCount, conf.PageSize, pageIndex, beego.URLFor("ManagerController.Books"), fmt.Sprintf("&private=%v", private))
	} else {
		this.Data["PageHtml"] = ""
	}

	this.Data["Lists"] = books
	this.Data["IsBooks"] = true
	this.GetSeoByPage("manage_project_list", map[string]string{
		"title":       "项目管理 - " + this.Sitename,
		"keywords":    "项目管理",
		"description": this.Sitename + "专注于文档在线写作、协作、分享、阅读与托管，让每个人更方便地发布、分享和获得知识。",
	})
	this.Data["Private"] = private
	this.TplName = "manager/books.html"
}

//编辑项目.
func (this *ManagerController) EditBook() {

	identify := this.GetString(":key")
	if identify == "" {
		this.Abort("404")
	}

	book, err := models.NewBook().FindByFieldFirst("identify", identify)
	if err != nil {
		this.Abort("500")
	}

	if this.Ctx.Input.IsPost() {
		bookName := strings.TrimSpace(this.GetString("book_name"))
		description := strings.TrimSpace(this.GetString("description", ""))
		commentStatus := this.GetString("comment_status")
		tag := strings.TrimSpace(this.GetString("label"))
		orderIndex, _ := this.GetInt("order_index", 0)

		if strings.Count(description, "") > 500 {
			this.JsonResult(6004, "项目描述不能大于500字")
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

		if err := book.Update(); err != nil {
			this.JsonResult(6006, "保存失败")
		}
		this.JsonResult(0, "ok")
	}
	if book.PrivateToken != "" {
		book.PrivateToken = this.BaseUrl() + beego.URLFor("DocumentController.Index", ":key", book.Identify, "token", book.PrivateToken)
	}
	this.Data["Model"] = book

	this.GetSeoByPage("manage_project_edit", map[string]string{
		"title":       "项目设置 - " + this.Sitename,
		"keywords":    "项目设置",
		"description": this.Sitename + "专注于文档在线写作、协作、分享、阅读与托管，让每个人更方便地发布、分享和获得知识。",
	})
	this.TplName = "manager/edit_book.html"
}

// 删除项目.
func (this *ManagerController) DeleteBook() {

	bookId, _ := this.GetInt("book_id", 0)
	if bookId <= 0 {
		this.JsonResult(6001, "参数错误")
	}

	book := models.NewBook()
	err := book.ThoroughDeleteBook(bookId)

	if err == orm.ErrNoRows {
		this.JsonResult(6002, "项目不存在")
	}
	if err != nil {
		logs.Error("DeleteBook => ", err)
		this.JsonResult(6003, "删除失败")
	}

	go func() {
		client := models.NewElasticSearchClient()
		if errDel := client.DeleteIndex(bookId, true); errDel != nil && client.On {
			beego.Error(errDel.Error())
		}
	}()

	this.JsonResult(0, "ok")
}

// CreateToken 创建访问来令牌.
func (this *ManagerController) CreateToken() {

	action := this.GetString("action")
	identify := this.GetString("identify")

	book, err := models.NewBook().FindByFieldFirst("identify", identify)
	if err != nil {
		this.JsonResult(6001, "项目不存在")
	}

	if action == "create" {
		if book.PrivatelyOwned == 0 {
			this.JsonResult(6001, "公开项目不能创建阅读令牌")
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

	options, err := models.NewOption().All()
	if err != nil {
		this.Abort("500")
	}

	if this.Ctx.Input.IsPost() {
		for _, item := range options {
			item.OptionValue = this.GetString(item.OptionName)
			item.InsertOrUpdate()
		}
		if err := models.NewElasticSearchClient().Init(); err != nil {
			this.JsonResult(1, err.Error())
		}

		this.JsonResult(0, "ok")
	}

	this.Data["SITE_TITLE"] = this.Option["SITE_NAME"]
	for _, item := range options {
		this.Data[item.OptionName] = item
	}

	this.Data["IsSetting"] = true
	this.Data["SeoTitle"] = "配置管理"
	this.TplName = "manager/setting.html"
}

// Transfer 转让项目.
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
		this.Abort("403")
	}

	identify := this.GetString("identify")

	book, err := models.NewBook().FindByFieldFirst("identify", identify)
	if err != nil {
		this.JsonResult(6001, err.Error())
	}

	rel, err := models.NewRelationship().FindFounder(book.BookId)
	if err != nil {
		beego.Error("FindFounder => ", err)
		this.JsonResult(6009, "查询项目创始人失败")
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
	if !this.Member.IsAdministrator() {
		this.Abort("403")
	}
	this.TplName = "manager/comments.html"
}

//DeleteComment 标记评论为已删除
func (this *ManagerController) DeleteComment() {

	commentId, _ := this.GetInt("comment_id", 0)
	if commentId <= 0 {
		this.JsonResult(6001, "参数错误")
	}

	comment := models.NewComment()
	if _, err := comment.Find(commentId); err != nil {
		this.JsonResult(6002, "评论不存在")
	}

	comment.Approved = 3
	if err := comment.Update("approved"); err != nil {
		this.JsonResult(6003, "删除评论失败")
	}
	this.JsonResult(0, "ok", comment)
}

//设置项目私有状态.
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
		this.Abort("403")
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

//附件列表.
func (this *ManagerController) AttachList() {

	pageIndex, _ := this.GetInt("page", 1)

	attachList, totalCount, err := models.NewAttachment().FindToPager(pageIndex, conf.PageSize)
	if err != nil {
		this.Abort("500")
	}

	if totalCount > 0 {
		html := utils.GetPagerHtml(this.Ctx.Request.RequestURI, pageIndex, conf.PageSize, int(totalCount))
		this.Data["PageHtml"] = html
	} else {
		this.Data["PageHtml"] = ""
	}

	for _, item := range attachList {
		p := filepath.Join(commands.WorkingDirectory, item.FilePath)
		item.IsExist = utils.FileExists(p)
	}

	this.Data["Lists"] = attachList
	this.Data["IsAttach"] = true
	this.TplName = "manager/attach_list.html"
}

//附件详情.
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
		this.Abort("500")
	}

	attach.FilePath = filepath.Join(commands.WorkingDirectory, attach.FilePath)
	attach.HttpPath = this.BaseUrl() + attach.HttpPath
	attach.IsExist = utils.FileExists(attach.FilePath)
	this.Data["Model"] = attach
	this.TplName = "manager/attach_detailed.html"
}

//删除附件.
func (this *ManagerController) AttachDelete() {

	attachId, _ := this.GetInt("attach_id")
	if attachId <= 0 {
		this.Abort("404")
	}

	attach, err := models.NewAttachment().Find(attachId)
	if err != nil {
		beego.Error("AttachDelete => ", err)
		this.JsonResult(6001, err.Error())
	}

	if err := attach.Delete(); err != nil {
		beego.Error("AttachDelete => ", err)
		this.JsonResult(6002, err.Error())
	}
	this.JsonResult(0, "ok")
}

//SEO管理
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

//更行书籍项目的排序
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

//分类管理
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

//更新分类字段内容
func (this *ManagerController) UpdateCate() {
	field := this.GetString("field")
	val := this.GetString("value")
	id, _ := this.GetInt("id")
	if err := new(models.Category).UpdateByField(id, field, val); err != nil {
		this.JsonResult(1, "更新失败："+err.Error())
	}
	this.JsonResult(0, "更新成功")
}

//删除分类
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

//更新分类的图标
func (this *ManagerController) UpdateCateIcon() {
	var err error
	id, _ := this.GetInt("id")
	if id == 0 {
		this.JsonResult(1, "参数不正确")
	}
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
			case utils.StoreLocal:
				store.ModelStoreLocal.DelFiles(cate.Icon)
			}
			err = model.UpdateByField(cate.Id, "icon", "/"+tmpFile)
		}
	}

	if err != nil {
		this.JsonResult(1, err.Error())
	}
	this.JsonResult(0, "更新成功")
}

//友情链接
func (this *ManagerController) FriendLink() {
	this.Data["SeoTitle"] = "友链管理"
	this.Data["Links"] = new(models.FriendLink).GetList(true)
	this.Data["IsFriendlink"] = true
	this.TplName = "manager/friendlink.html"
}

//添加友链
func (this *ManagerController) AddFriendlink() {
	if err := new(models.FriendLink).Add(this.GetString("title"), this.GetString("link")); err != nil {
		this.JsonResult(1, "新增友链失败:"+err.Error())
	}
	this.JsonResult(0, "新增友链成功")
}

//更新友链
func (this *ManagerController) UpdateFriendlink() {
	id, _ := this.GetInt("id")
	if err := new(models.FriendLink).Update(id, this.GetString("field"), this.GetString("value")); err != nil {
		this.JsonResult(1, "操作失败："+err.Error())
	}
	this.JsonResult(0, "操作成功")
}

//删除友链
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
