package controllers

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"fmt"

	"github.com/TruthHun/BookStack/conf"
	"github.com/TruthHun/BookStack/models"
	"github.com/TruthHun/BookStack/models/store"
	"github.com/TruthHun/BookStack/utils"
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/orm"
	"github.com/casdoor/casdoor-go-sdk/auth"
)

type SettingController struct {
	BaseController
}

//基本信息
func (this *SettingController) Index() {
	this.SetSession("isUpdateUser", 1)
	account := this.GetSessionClaims()
	if account == nil {
		this.Redirect(beego.URLFor("AccountController.Login"), 302)
	} else {
		myProfileUrl := auth.GetMyProfileUrl(account.AccessToken)
		this.Redirect(myProfileUrl, 302)
	}
}

//收藏
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

//二维码
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
