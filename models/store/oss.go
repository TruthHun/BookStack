package store

import (
	"errors"

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

//获取oss的配置[如果使用到多个Bucket，则这里定义一个new方法，生成不同oss配置的OSS对象]
//@return               oss             Oss配置信息
func (this *Oss) Config() (oss Oss) {
	oss = Oss{
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

//判断文件对象是否存在
//@param                object              文件对象
//@param                isBucketPreview     是否是供预览的bucket，true表示预览bucket，false表示存储bucket
//@return               err                 错误，nil表示文件存在，否则表示文件不存在，并告知错误信息
func (this *Oss) IsObjectExist(object string) (err error) {
	var (
		b      bool
		Client *oss.Client
		Bucket *oss.Bucket
		config = this.Config()
		bucket = config.Bucket
	)
	if len(object) == 0 {
		return errors.New("文件参数为空")
	}
	if config.IsInternal {
		Client, err = oss.New(config.EndpointInternal, config.AccessKeyId, config.AccessKeySecret)
	} else {
		Client, err = oss.New(config.EndpointOuter, config.AccessKeyId, config.AccessKeySecret)
	}
	if err == nil {
		if Bucket, err = Client.Bucket(bucket); err == nil {
			if b, err = Bucket.IsObjectExist(object); b == true {
				return nil
			}
			if err == nil {
				err = errors.New("文件不存在")
			}
		}
	}
	return err
}

//文件移动到OSS进行存储[如果是图片文件，不要使用gzip压缩，否则在使用阿里云OSS自带的图片处理功能无法处理图片]
//@param            local            本地文件
//@param            save             存储到OSS的文件
//@param            IsPreview        是否是预览，是预览的话，则上传到预览的OSS，否则上传到存储的OSS。存储的OSS，只作为文档的存储，以供下载，但不提供预览等访问，为私有
//@param            IsDel            文件上传后，是否删除本地文件
//@param            IsGzip           是否做gzip压缩，做gzip压缩的话，需要修改oss中对象的响应头，设置gzip响应
func (this *Oss) MoveToOss(local, save string, IsDel bool, IsGzip ...bool) error {
	config := this.Config()
	isgzip := false
	//如果是开启了gzip，则需要设置文件对象的响应头
	if len(IsGzip) > 0 && IsGzip[0] == true {
		isgzip = true
	}

	endpoint := config.EndpointOuter
	//如果是内网，则使用内网endpoint
	if config.IsInternal {
		endpoint = config.EndpointInternal
	}

	client, err := oss.New(endpoint, config.AccessKeyId, config.AccessKeySecret)
	if err != nil {
		beego.Error("OSS Client初始化错误：%v", err.Error())
		return err
	}
	Bucket, err := client.Bucket(config.Bucket)
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
	err = Bucket.PutObjectFromFile(save, local)
	if err != nil {
		beego.Error("文件移动到OSS失败：", err.Error())
		return err
	}
	//如果是开启了gzip，则需要设置文件对象的响应头
	if isgzip {
		Bucket.SetObjectMeta(save, oss.ContentEncoding("gzip")) //设置gzip响应头
	}

	if err == nil && IsDel {
		err = os.Remove(local)
	}

	return err
}

//从OSS中删除文件
//@param           object                     文件对象
//@param           IsPreview                  是否是预览的OSS
func (this *Oss) DelFromOss(object ...string) error {
	config := this.Config()
	bucket := config.Bucket

	endpoint := config.EndpointOuter
	//如果是内网，则使用内网endpoint
	if config.IsInternal {
		endpoint = config.EndpointInternal
	}
	client, err := oss.New(endpoint, config.AccessKeyId, config.AccessKeySecret)
	if err != nil {
		return err
	}
	Bucket, err := client.Bucket(bucket)
	if err != nil {
		return err
	}
	_, err = Bucket.DeleteObjects(object)
	return err
}

//设置文件的下载名
//@param            obj             文档对象
//@param            filename        文件名
func (this *Oss) SetObjectMeta(obj, filename string) {
	config := this.Config()
	if client, err := oss.New(config.EndpointOuter, config.AccessKeyId, config.AccessKeySecret); err == nil {
		if Bucket, err := client.Bucket(config.Bucket); err == nil {
			Bucket.SetObjectMeta(obj, oss.ContentDisposition(fmt.Sprintf("attachment; filename=%v", filename)))
			//Bucket.SetObjectMeta(obj, oss.Meta("ContentDisposition", fmt.Sprintf("attachment; filename=%v", filename)))

			//Bucket.SetObjectMeta(obj, )
		}
	}
}

//处理html中的OSS数据：如果是用于预览的内容，则把img等的链接的相对路径转成绝对路径，否则反之
//@param            htmlstr             html字符串
//@param            forPreview          是否是供浏览的页面需求
//@return           str                 处理后返回的字符串
func (this *Oss) HandleContent(htmlstr string, forPreview bool) (str string) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlstr))
	config := this.Config()
	if err == nil {
		doc.Find("img").Each(func(i int, s *goquery.Selection) {
			// For each item found, get the band and title
			if src, exist := s.Attr("src"); exist {
				//预览
				if forPreview {
					//不存在http开头的图片链接，则更新为绝对链接
					if !(strings.HasPrefix(src, "http://") || strings.HasPrefix(src, "https://")) {
						s.SetAttr("src", config.Domain+"/"+strings.TrimLeft(src, "./"))
					}
				} else {
					s.SetAttr("src", strings.TrimPrefix(src, config.Domain))
				}
			}

		})
		str, _ = doc.Find("body").Html()
	}
	return
}

//从HTML中提取图片文件，并删除
func (this *Oss) DelByHtmlPics(htmlstr string) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlstr))
	config := this.Config()
	if err == nil {
		doc.Find("img").Each(func(i int, s *goquery.Selection) {
			// For each item found, get the band and title
			if src, exist := s.Attr("src"); exist {
				//不存在http开头的图片链接，则更新为绝对链接
				if !(strings.HasPrefix(src, "http://") || strings.HasPrefix(src, "https://")) {
					this.DelFromOss(strings.TrimLeft(src, "./")) //删除
				} else if strings.HasPrefix(src, config.Domain) {
					this.DelFromOss(strings.TrimPrefix(src, config.Domain)) //删除
				}
			}
		})
	}
	return
}

//根据oss文件夹
func (this *Oss) DelOssFolder(folder string) (err error) {
	config := this.Config()
	if client, err := oss.New(config.EndpointOuter, config.AccessKeyId, config.AccessKeySecret); err == nil {
		if Bucket, err := client.Bucket(config.Bucket); err == nil {
			folder = strings.Trim(folder, "/")
			if lists, err := Bucket.ListObjects(oss.Prefix(folder)); err != nil {
				return err
			} else {
				var objs []string
				for _, list := range lists.Objects {
					objs = append(objs, list.Key)
				}
				if len(objs) > 0 {
					this.DelFromOss(objs...)
				}
				this.DelFromOss(folder)
			}

		}
	}
	return
}
