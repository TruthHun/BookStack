package store

import (
	"io"
	"strings"

	"os"

	"fmt"

	"bytes"
	"io/ioutil"

	"compress/gzip"

	"github.com/PuerkitoBio/goquery"
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/astaxie/beego"
)

var ModelStoreOss = NewOss()
var ModelStoreLocal = new(Local)

//OSS配置【这个不再作为数据库表，直接在oss.conf文件中进行配置】
type Oss struct {
	EndpointInternal string //内网的endpoint
	EndpointOuter    string //外网的endpoint
	AccessKeyId      string //key
	AccessKeySecret  string //secret
	Bucket           string //供文档预览的bucket
	IsInternal       bool   //是否内网，内网则启用内网endpoint，否则启用外网endpoint
	Domain           string
}

func NewOss() (oss *Oss) {
	oss = &Oss{
		IsInternal:       beego.AppConfig.DefaultBool("oss::IsInternal", false),
		EndpointInternal: beego.AppConfig.String("oss::EndpointInternal"),
		EndpointOuter:    beego.AppConfig.String("oss::EndpointOuter"),
		AccessKeyId:      beego.AppConfig.String("oss::AccessKeyId"),
		AccessKeySecret:  beego.AppConfig.String("oss::AccessKeySecret"),
		Bucket:           beego.AppConfig.String("oss::Bucket"),
		Domain:           strings.TrimRight(beego.AppConfig.String("oss::Domain"), "/ "),
	}
	return oss
}

// 获取bucket
func (o *Oss) GetBucket() (bucket *oss.Bucket, err error) {
	var client *oss.Client
	if o.IsInternal {
		client, err = oss.New(o.EndpointInternal, o.AccessKeyId, o.AccessKeySecret)
	} else {
		client, err = oss.New(o.EndpointOuter, o.AccessKeyId, o.AccessKeySecret)
	}
	if err != nil {
		return
	}
	return client.Bucket(o.Bucket)
}

//判断文件对象是否存在
//@param                object              文件对象
//@param                isBucketPreview     是否是供预览的bucket，true表示预览bucket，false表示存储bucket
//@return               err                 错误，nil表示文件存在，否则表示文件不存在，并告知错误信息
func (o *Oss) IsObjectExist(object string) (err error) {
	bucket, err := o.GetBucket()
	if err != nil {
		return
	}
	_, err = bucket.GetObjectMeta(object)
	return
}

//文件移动到OSS进行存储[如果是图片文件，不要使用gzip压缩，否则在使用阿里云OSS自带的图片处理功能无法处理图片]
//@param            local            本地文件
//@param            save             存储到OSS的文件
//@param            IsPreview        是否是预览，是预览的话，则上传到预览的OSS，否则上传到存储的OSS。存储的OSS，只作为文档的存储，以供下载，但不提供预览等访问，为私有
//@param            IsDel            文件上传后，是否删除本地文件
//@param            IsGzip           是否做gzip压缩，做gzip压缩的话，需要修改oss中对象的响应头，设置gzip响应
func (o *Oss) MoveToOss(local, save string, IsDel bool, IsGzip ...bool) error {
	isgzip := false
	//如果是开启了gzip，则需要设置文件对象的响应头
	if len(IsGzip) > 0 && IsGzip[0] == true {
		isgzip = true
	}

	bucket, err := o.GetBucket()
	if err != nil {
		beego.Error("OSS Bucket初始化错误：%v", err.Error())
		return err
	}
	//在移动文件到OSS之前，先压缩文件
	if isgzip {
		if bs, err := ioutil.ReadFile(local); err != nil {
			beego.Error(err.Error())
			isgzip = false //设置为false
		} else {
			var by bytes.Buffer
			w := gzip.NewWriter(&by)
			defer w.Close()
			w.Write(bs)
			w.Flush()
			ioutil.WriteFile(local, by.Bytes(), 0777)
		}
	}
	err = bucket.PutObjectFromFile(save, local)
	if err != nil {
		beego.Error("文件移动到OSS失败：", err.Error())
		return err
	}
	//如果是开启了gzip，则需要设置文件对象的响应头
	if isgzip {
		bucket.SetObjectMeta(save, oss.ContentEncoding("gzip")) //设置gzip响应头
	}

	if err == nil && IsDel {
		err = os.Remove(local)
	}

	return err
}

//从OSS中删除文件
//@param           object                     文件对象
//@param           IsPreview                  是否是预览的OSS
func (o *Oss) DelFromOss(object ...string) (err error) {

	bucket, err := o.GetBucket()
	if err != nil {
		return err
	}
	_, err = bucket.DeleteObjects(object)
	return err
}

//设置文件的下载名
//@param            obj             文档对象
//@param            filename        文件名
func (o *Oss) SetObjectMeta(obj, filename string) {
	bucket, _ := o.GetBucket()
	bucket.SetObjectMeta(obj, oss.ContentDisposition(fmt.Sprintf("attachment; filename=%v", filename)))
}

//处理html中的OSS数据：如果是用于预览的内容，则把img等的链接的相对路径转成绝对路径，否则反之
//@param            htmlstr             html字符串
//@param            forPreview          是否是供浏览的页面需求
//@return           str                 处理后返回的字符串
func (o *Oss) HandleContent(htmlStr string, forPreview bool) (str string) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlStr))
	if err != nil {
		beego.Error(err.Error())
		return htmlStr
	}
	doc.Find("img").Each(func(i int, s *goquery.Selection) {
		// For each item found, get the band and title
		if src, exist := s.Attr("src"); exist {
			//预览
			if forPreview {
				//不存在http开头的图片链接，则更新为绝对链接
				if !(strings.HasPrefix(src, "http://") || strings.HasPrefix(src, "https://")) {
					s.SetAttr("src", o.Domain+"/"+strings.TrimLeft(src, "./"))
				}
			} else {
				s.SetAttr("src", strings.TrimPrefix(src, o.Domain))
			}
		}
	})
	str, _ = doc.Find("body").Html()
	return
}

//从HTML中提取图片文件，并删除
func (o *Oss) DelByHtmlPics(htmlStr string) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlStr))
	if err != nil {
		beego.Error(err.Error())
		return
	}
	doc.Find("img").Each(func(i int, s *goquery.Selection) {
		// For each item found, get the band and title
		if src, exist := s.Attr("src"); exist {
			//不存在http开头的图片链接，则更新为绝对链接
			if !(strings.HasPrefix(src, "http://") || strings.HasPrefix(src, "https://")) {
				o.DelFromOss(strings.TrimLeft(src, "./")) //删除
			} else if strings.HasPrefix(src, o.Domain) {
				o.DelFromOss(strings.TrimPrefix(src, o.Domain)) //删除
			}
		}
	})
	return
}

//根据oss文件夹
func (o *Oss) DelOssFolder(folder string) (err error) {
	bucket, err := o.GetBucket()
	folder = strings.Trim(folder, "/") + "/"
	if lists, err := bucket.ListObjects(oss.Prefix(folder)); err != nil {
		return err
	} else {
		var objs []string
		folderLower := strings.ToLower(folder)
		for _, list := range lists.Objects {
			if strings.HasPrefix(strings.ToLower(list.Key), folderLower) {
				objs = append(objs, list.Key)
			}
		}
		if len(objs) > 0 {
			o.DelFromOss(objs...)
		}
		o.DelFromOss(folder)
	}
	return
}

func (o *Oss) GetFileReader(objKey string) (reader io.ReadCloser, err error) {
	var bucket *oss.Bucket
	bucket, err = o.GetBucket()
	if err != nil {
		return
	}
	reader, err = bucket.GetObject(objKey)
	return
}
