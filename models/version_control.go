package models

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/astaxie/beego"

	"github.com/TruthHun/BookStack/models/store"
	"github.com/TruthHun/BookStack/utils"
)

// 版本控制，文件存储于获取
type versionControl struct {
	DocId    int    //文档id
	Version  int64  //版本(时间戳)
	HtmlFile string //HTML文件名
	MdFile   string //md文件名
}

func NewVersionControl(docId int, version int64) *versionControl {
	t := time.Unix(version, 0).Format("2006/01/02/%v/15/04/05")
	folder := "./version_control/" + fmt.Sprintf(t, docId)
	if utils.StoreType == utils.StoreLocal {
		os.MkdirAll(folder, os.ModePerm)
	}
	return &versionControl{
		DocId:    docId,
		Version:  version,
		HtmlFile: folder + "master.html",
		MdFile:   folder + "master.md",
	}
}

// 保存版本数据
func (v *versionControl) SaveVersion(htmlContent, mdContent string) (err error) {
	if utils.StoreType == utils.StoreLocal { //本地存储
		if err = ioutil.WriteFile(v.HtmlFile, []byte(htmlContent), os.ModePerm); err != nil {
			return err
		}
		if err = ioutil.WriteFile(v.MdFile, []byte(mdContent), os.ModePerm); err != nil {
			return err
		}
	} else { // OSS 存储
		bucket, err := store.NewOss().GetBucket()
		if err != nil {
			return err
		}
		if err = bucket.PutObject(strings.TrimLeft(v.HtmlFile, "./"), strings.NewReader(htmlContent)); err != nil {
			return err
		}
		if err = bucket.PutObject(strings.TrimLeft(v.MdFile, "./"), strings.NewReader(mdContent)); err != nil {
			return err
		}
	}
	return
}

// 获取版本数据
func (v *versionControl) GetVersionContent(isHtml bool) (content string) {
	file := v.MdFile
	if isHtml {
		file = v.HtmlFile
	}
	if utils.StoreType == utils.StoreLocal { //本地存储
		b, err := ioutil.ReadFile(file)
		if err == nil {
			content = string(b)
		}
	} else { // OSS 存储
		bucket, err := store.NewOss().GetBucket()
		if err != nil {
			beego.Error(err.Error())
			return
		}
		reader, err := bucket.GetObject(strings.TrimLeft(file, "./"))
		if err == nil {
			b, _ := ioutil.ReadAll(reader)
			content = string(b)
		}
	}
	return
}

// 删除版本文件
func (v *versionControl) DeleteVersion() (err error) {
	if utils.StoreType == utils.StoreLocal { //本地存储
		os.Remove(v.HtmlFile)
		os.Remove(v.MdFile)
		os.Remove(filepath.Dir(v.HtmlFile))
	} else { // OSS 存储
		bucket, err := store.NewOss().GetBucket()
		if err != nil {
			beego.Error(err.Error())
			return err
		}
		_, err = bucket.DeleteObjects([]string{
			strings.TrimLeft(v.MdFile, "./"),
			strings.TrimLeft(v.HtmlFile, "./"),
		})
	}
	return
}
