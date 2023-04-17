package controllers

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/TruthHun/BookStack/graphics"
	"github.com/TruthHun/BookStack/models/store"

	"fmt"

	"github.com/TruthHun/BookStack/conf"
	"github.com/TruthHun/BookStack/models"
	"github.com/TruthHun/BookStack/utils"
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/logs"
	"github.com/astaxie/beego/orm"
)

type SettingController struct {
	BaseController
}

// 基本信息
func (this *SettingController) Index() {
	if this.Ctx.Input.IsPost() {
		email := strings.TrimSpace(this.GetString("email", ""))
		phone := strings.TrimSpace(this.GetString("phone"))
		wechatNO := strings.TrimSpace(this.GetString("wechat_no"))
		description := strings.TrimSpace(this.GetString("description"))
		nickname := strings.TrimSpace(this.GetString("nickname"))
		if email == "" {
			this.JsonResult(601, "邮箱不能为空")
		}

		if l := strings.Count(nickname, "") - 1; l < 2 || l > 20 {
			this.JsonResult(6004, "用户昵称限制在2-20个字符")
		}

		existMember := models.NewMember().FindByNickname(nickname, "member_id")
		if existMember.MemberId > 0 && this.Member.MemberId != existMember.MemberId {
			this.JsonResult(6004, "用户昵称已存在，请换一个")
		}

		member := this.Member
		member.Email = email
		member.Phone = phone
		member.WechatNO = wechatNO
		member.Description = description
		member.Nickname = nickname
		if err := member.Update(); err != nil {
			this.JsonResult(602, err.Error())
		}
		this.SetMember(*member)
		this.JsonResult(0, "ok")
	}
	this.Data["SeoTitle"] = "基本信息 - " + this.Sitename
	this.Data["SettingBasic"] = true
	this.TplName = "setting/index.html"
}

// 修改密码
func (this *SettingController) Password() {

	if this.Ctx.Input.IsPost() {
		if this.Member.AuthMethod == conf.AuthMethodLDAP {
			this.JsonResult(6009, "当前用户不支持修改密码")
		}
		password1 := this.GetString("password1")
		password2 := this.GetString("password2")
		password3 := this.GetString("password3")
		if password1 == "" {
			this.JsonResult(6003, "原密码不能为空")
		}

		if password2 == "" {
			this.JsonResult(6004, "新密码不能为空")
		}

		if count := strings.Count(password2, ""); count < 6 || count > 18 {
			this.JsonResult(6009, "密码必须在6-18字之间")
		}

		if password2 != password3 {
			this.JsonResult(6003, "确认密码不正确")
		}

		if ok, _ := utils.PasswordVerify(this.Member.Password, password1); !ok {
			this.JsonResult(6005, "原始密码不正确")
		}

		if password1 == password2 {
			this.JsonResult(6006, "新密码不能和原始密码相同")
		}

		pwd, err := utils.PasswordHash(password2)
		if err != nil {
			this.JsonResult(6007, "密码加密失败")
		}

		this.Member.Password = pwd
		if err := this.Member.Update(); err != nil {
			this.JsonResult(6008, err.Error())
		}

		this.JsonResult(0, "ok")
	}

	this.Data["SettingPwd"] = true
	this.Data["SeoTitle"] = "修改密码 - " + this.Sitename
	this.TplName = "setting/password.html"
}

// 收藏
func (this *SettingController) Star() {
	page, _ := this.GetInt("page")
	cid, _ := this.GetInt("cid")
	if page < 1 {
		page = 1
	}
	sort := this.GetString("sort", "read")

	cnt, books, _ := new(models.Star).List(this.Member.MemberId, page, conf.PageSize, cid, sort)
	if cnt > 1 {
		//this.Data["PageHtml"] = utils.GetPagerHtml(this.Ctx.Request.RequestURI, page, listRows, int(cnt))
		this.Data["PageHtml"] = utils.NewPaginations(conf.RollPage, int(cnt), conf.PageSize, page, beego.URLFor("SettingController.Star"), "")
	}
	this.Data["Pid"] = 0

	cates := models.NewCategory().CategoryOfUserCollection(this.Member.MemberId)
	for _, cate := range cates {
		if cate.Id == cid {
			if cate.Pid == 0 {
				this.Data["Pid"] = cate.Id
			} else {
				this.Data["Pid"] = cate.Pid
			}
		}
	}

	this.Data["Books"] = books
	this.Data["Sort"] = sort
	this.Data["SettingStar"] = true
	this.Data["SeoTitle"] = "我的收藏 - " + this.Sitename
	this.TplName = "setting/star.html"
	this.Data["Cid"] = cid
	this.Data["Cates"] = cates
}

// 二维码
func (this *SettingController) Qrcode() {

	if this.Ctx.Input.IsPost() {
		file, moreFile, err := this.GetFile("qrcode")
		alipay := true
		if this.GetString("paytype") == "wxpay" {
			alipay = false
		}
		if err != nil {
			beego.Error(err.Error())
			this.JsonResult(500, "读取文件异常")
		}
		defer file.Close()
		ext := filepath.Ext(moreFile.Filename)

		if !strings.EqualFold(ext, ".png") && !strings.EqualFold(ext, ".jpg") && !strings.EqualFold(ext, ".gif") && !strings.EqualFold(ext, ".jpeg") {
			this.JsonResult(500, "不支持的图片格式")
		}

		savePath := fmt.Sprintf("uploads/qrcode/%v/%v%v", this.Member.MemberId, time.Now().Unix(), ext)
		os.MkdirAll(filepath.Dir(savePath), 0777)
		if err = this.SaveToFile("qrcode", savePath); err != nil {
			this.JsonResult(1, "二维码保存失败", savePath)
		}
		url := ""
		switch utils.StoreType {
		case utils.StoreOss:
			if err := store.ModelStoreOss.MoveToOss(savePath, savePath, true, false); err != nil {
				beego.Error(err.Error())
			} else {
				url = strings.TrimRight(beego.AppConfig.String("oss::Domain"), "/ ") + "/" + savePath
			}
		case utils.StoreLocal:
			if err := store.ModelStoreLocal.MoveToStore(savePath, savePath); err != nil {
				beego.Error(err.Error())
			} else {
				url = "/" + savePath
			}
		}

		var member models.Member
		o := orm.NewOrm()
		o.QueryTable("md_members").Filter("member_id", this.Member.MemberId).Filter("member_id", this.Member.MemberId).One(&member, "member_id", "wxpay", "alipay")
		if member.MemberId > 0 {
			dels := []string{}

			if alipay {
				dels = append(dels, member.Alipay)
				member.Alipay = savePath
			} else {
				dels = append(dels, member.Wxpay)
				member.Wxpay = savePath
			}
			if _, err := o.Update(&member, "wxpay", "alipay"); err == nil {
				switch utils.StoreType {
				case utils.StoreOss:
					go store.ModelStoreOss.DelFromOss(dels...)
				case utils.StoreLocal:
					go store.ModelStoreLocal.DelFiles(dels...)
				}
			}
		}
		//删除旧的二维码，并更新新的二维码
		data := map[string]interface{}{
			"url":    url,
			"alipay": alipay,
		}
		this.JsonResult(0, "二维码上传成功", data)
	}

	this.TplName = "setting/qrcode.html"
	this.Data["SeoTitle"] = "二维码管理 - " + this.Sitename
	this.Data["Qrcode"] = new(models.Member).GetQrcodeByUid(this.Member.MemberId)
	this.Data["SettingQrcode"] = true
}

// Upload 上传图片
func (this *SettingController) Upload() {

	file, moreFile, err := this.GetFile("image-file")
	if err != nil {
		logs.Error("", err.Error())
		this.JsonResult(500, "读取文件异常")
	}
	defer file.Close()

	ext := filepath.Ext(moreFile.Filename)
	if !strings.EqualFold(ext, ".png") && !strings.EqualFold(ext, ".jpg") && !strings.EqualFold(ext, ".gif") && !strings.EqualFold(ext, ".jpeg") {
		this.JsonResult(500, "不支持的图片格式")
	}

	x1, _ := strconv.ParseFloat(this.GetString("x"), 10)
	y1, _ := strconv.ParseFloat(this.GetString("y"), 10)
	w1, _ := strconv.ParseFloat(this.GetString("width"), 10)
	h1, _ := strconv.ParseFloat(this.GetString("height"), 10)

	x := int(x1)
	y := int(y1)
	width := int(w1)
	height := int(h1)

	// fmt.Println(x, x1, y, y1)

	fileName := strconv.FormatInt(time.Now().UnixNano(), 16)

	filePath := filepath.Join("uploads", time.Now().Format("2006/01"), fileName+ext)

	path := filepath.Dir(filePath)

	os.MkdirAll(path, os.ModePerm)

	err = this.SaveToFile("image-file", filePath)

	if err != nil {
		logs.Error("", err)
		this.JsonResult(500, "图片保存失败")
	}

	//剪切图片
	subImg, err := graphics.ImageCopyFromFile(filePath, x, y, width, height)

	if err != nil {
		logs.Error("ImageCopyFromFile => ", err)
		this.JsonResult(6001, "头像剪切失败")
	}
	os.Remove(filePath)

	filePath = filepath.Join("uploads", time.Now().Format("200601"), fileName+ext)

	err = graphics.ImageResizeSaveFile(subImg, 120, 120, filePath)
	err = graphics.SaveImage(filePath, subImg)

	if err != nil {
		logs.Error("保存文件失败 => ", err.Error())
		this.JsonResult(500, "保存文件失败")
	}

	url := "/" + strings.Replace(filePath, "\\", "/", -1)
	if strings.HasPrefix(url, "//") {
		url = string(url[1:])
	}

	if member, err := models.NewMember().Find(this.Member.MemberId); err == nil {
		avatar := member.Avatar
		member.Avatar = url
		err = member.Update()
		if err != nil {
			this.JsonResult(60001, "保存头像失败")
		}
		avatar = strings.TrimLeft(avatar, "./")
		if strings.HasPrefix(avatar, "uploads/") {
			os.Remove(avatar)
		}
		this.SetMember(*member)
	}
	switch utils.StoreType {
	case utils.StoreOss: //oss存储
		if err := store.ModelStoreOss.MoveToOss("."+url, strings.TrimLeft(url, "./"), true, false); err != nil {
			beego.Error(err.Error())
		} else {
			url = strings.TrimRight(beego.AppConfig.String("oss::Domain"), "/ ") + url + "/avatar"
		}
	case utils.StoreLocal: //本地存储
		if err := store.ModelStoreLocal.MoveToStore("."+url, strings.TrimLeft(url, "./")); err != nil {
			beego.Error(err.Error())
		} else {
			url = "/" + strings.TrimLeft(url, "./")
		}
	}

	this.JsonResult(0, "ok", url)
}
