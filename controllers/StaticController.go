package controllers

import (
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/TruthHun/BookStack/models"

	"github.com/TruthHun/BookStack/models/store"
	"github.com/TruthHun/BookStack/utils"
	"github.com/astaxie/beego"
)

type StaticController struct {
	beego.Controller
}

func (this *StaticController) APP() {
	link := strings.TrimSpace(models.GetOptionValue("APP_PAGE", ""))
	if link != "" {
		this.Redirect(link, 302)
	}
	this.Abort("404")
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

// 项目静态文件
func (this *StaticController) ProjectsFile() {
	prefix := "projects/"
	object := prefix + strings.TrimLeft(this.GetString(":splat"), "./")

	//这里的时间只是起到缓存的作用
	t, _ := time.Parse("2006-01-02 15:04:05", "2006-01-02 15:04:05")
	date := t.Format(http.TimeFormat)
	since := this.Ctx.Request.Header.Get("If-Modified-Since")
	if since == date {
		this.Ctx.ResponseWriter.WriteHeader(http.StatusNotModified)
		return
	}

	if utils.StoreType == utils.StoreOss { //oss
		reader, err := store.NewOss().GetFileReader(object)
		if err != nil {
			beego.Error(err.Error())
			this.Abort("404")
		}
		defer reader.Close()

		b, err := ioutil.ReadAll(reader)
		if err != nil {
			beego.Error(err.Error())
			this.Abort("404")
		}
		this.Ctx.ResponseWriter.Header().Set("Last-Modified", date)
		if strings.HasSuffix(object, ".svg") {
			this.Ctx.ResponseWriter.Header().Set("Content-Type", "image/svg+xml")
		}
		this.Ctx.ResponseWriter.Write(b)
	} else { //local
		this.Abort("404")
	}
}
