package controllers

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/TruthHun/BookStack/models"

	"github.com/TruthHun/BookStack/utils"
	"github.com/astaxie/beego"
)

type StaticController struct {
	beego.Controller
	OssDomain string
}

func (this *StaticController) Prepare() {
	this.OssDomain = strings.TrimRight(beego.AppConfig.String("oss::Domain"), "/ ")
}

func (this *StaticController) APP() {
	link := strings.TrimSpace(models.GetOptionValue("APP_PAGE", ""))
	if link != "" {
		this.Redirect(link, 302)
	}
	this.Abort("404")
}

// Uploads 查询上传的静态资源。
// 如果是音频和视频文件，需要根据后台设置而判断是否加密处理
// 如果使用了OSS存储，则需要将文件处理好
func (this *StaticController) Uploads() {
	file := strings.TrimLeft(this.GetString(":splat"), "./")
	path := strings.ReplaceAll(filepath.Join("uploads", file), "\\", "/")
	fmt.Println("===========", path)
	http.ServeFile(this.Ctx.ResponseWriter, this.Ctx.Request, path)
}

//静态文件，这个加在路由的最后
func (this *StaticController) StaticFile() {
	file := this.GetString(":splat")
	if strings.HasPrefix(file, ".well-known") || file == "sitemap.xml" {
		http.ServeFile(this.Ctx.ResponseWriter, this.Ctx.Request, file)
		return
	}
	file = strings.TrimLeft(file, "./")
	path := filepath.Join(utils.VirtualRoot, file)
	http.ServeFile(this.Ctx.ResponseWriter, this.Ctx.Request, path)
}

// 书籍静态文件
func (this *StaticController) ProjectsFile() {
	object := filepath.Join("projects/", strings.TrimLeft(this.GetString(":splat"), "./"))
	if utils.StoreType == utils.StoreOss { //oss
		this.Redirect(this.OssDomain+"/"+object, 302)
	} else { //local
		this.Abort("404")
	}
}
