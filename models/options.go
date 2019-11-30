package models

import (
	"github.com/TruthHun/BookStack/conf"
	"github.com/astaxie/beego/orm"
)

// Option struct .
type Option struct {
	OptionId    int    `orm:"column(option_id);pk;auto;unique;" json:"option_id"`
	OptionTitle string `orm:"column(option_title);size(500)" json:"option_title"`
	OptionName  string `orm:"column(option_name);unique;size(80)" json:"option_name"`
	OptionValue string `orm:"column(option_value);type(text);null" json:"option_value"`
	Remark      string `orm:"column(remark);type(text);null" json:"remark"`
}

// TableName 获取对应数据库表名.
func (m *Option) TableName() string {
	return "options"
}

// TableEngine 获取数据使用的引擎.
func (m *Option) TableEngine() string {
	return "INNODB"
}

func (m *Option) TableNameWithPrefix() string {
	return conf.GetDatabasePrefix() + m.TableName()
}

func NewOption() *Option {
	return &Option{}
}

func (p *Option) Find(id int) (*Option, error) {
	o := orm.NewOrm()

	p.OptionId = id

	if err := o.Read(p); err != nil {
		return p, err
	}
	return p, nil
}

func (p *Option) FindByKey(key string) (*Option, error) {
	o := orm.NewOrm()
	if err := o.QueryTable(p).Filter("option_name", key).One(p); err != nil {
		return p, err
	}
	return p, nil
}

func GetOptionValue(key, def string) string {

	if option, err := NewOption().FindByKey(key); err == nil {
		return option.OptionValue
	}
	return def
}

func (p *Option) InsertOrUpdate() error {

	o := orm.NewOrm()

	var err error

	if p.OptionId > 0 || o.QueryTable(p.TableNameWithPrefix()).Filter("option_name", p.OptionName).Exist() {
		_, err = o.Update(p)
	} else {
		_, err = o.Insert(p)
	}
	return err
}

func (p *Option) InsertMulti(option ...Option) error {

	o := orm.NewOrm()

	_, err := o.InsertMulti(len(option), option)
	return err
}

func (p *Option) All() ([]*Option, error) {
	o := orm.NewOrm()
	var options []*Option

	_, err := o.QueryTable(p.TableNameWithPrefix()).All(&options)

	if err != nil {
		return options, err
	}
	return options, nil
}

func (m *Option) Init() error {

	o := orm.NewOrm()
	options := []Option{
		{
			OptionValue: "true",
			OptionName:  "ENABLED_REGISTER",
			OptionTitle: "是否启用注册",
		},
		{
			OptionValue: "100",
			OptionName:  "ENABLE_DOCUMENT_HISTORY",
			OptionTitle: "版本控制",
		}, {
			OptionValue: "true",
			OptionName:  "ENABLED_CAPTCHA",
			OptionTitle: "是否启用验证码",
		}, {
			OptionValue: "true",
			OptionName:  "ENABLE_ANONYMOUS",
			OptionTitle: "启用匿名访问",
		}, {
			OptionValue: "BookStack",
			OptionName:  "SITE_NAME",
			OptionTitle: "站点名称",
		}, {
			OptionValue: "",
			OptionName:  "ICP",
			OptionTitle: "网站备案",
		}, {
			OptionValue: "",
			OptionName:  "TONGJI",
			OptionTitle: "站点统计",
		}, {
			OptionValue: "true",
			OptionName:  "SPIDER",
			OptionTitle: "采集器，是否只对管理员开放",
		}, {
			OptionValue: "false",
			OptionName:  "ELASTICSEARCH_ON",
			OptionTitle: "是否开启全文搜索",
		}, {
			OptionValue: "http://localhost:9200/",
			OptionName:  "ELASTICSEARCH_HOST",
			OptionTitle: "ElasticSearch Host",
		}, {
			OptionValue: "book",
			OptionName:  "DEFAULT_SEARCH",
			OptionTitle: "默认搜索",
		}, {
			OptionValue: "50",
			OptionName:  "SEARCH_ACCURACY",
			OptionTitle: "搜索精度",
		}, {
			OptionValue: "true",
			OptionName:  "LOGIN_QQ",
			OptionTitle: "是否允许使用QQ登录",
		}, {
			OptionValue: "true",
			OptionName:  "LOGIN_GITHUB",
			OptionTitle: "是否允许使用Github登录",
		}, {
			OptionValue: "true",
			OptionName:  "LOGIN_GITEE",
			OptionTitle: "是否允许使用码云登录",
		}, {
			OptionValue: "0",
			OptionName:  "RELATE_BOOK",
			OptionTitle: "是否开始关联书籍",
		}, {
			OptionValue: "true",
			OptionName:  "ALL_CAN_WRITE_BOOK",
			OptionTitle: "是否都可以创建项目",
		}, {
			OptionValue: "false",
			OptionName:  "CLOSE_SUBMIT_ENTER",
			OptionTitle: "是否关闭收录入口",
		}, {
			OptionValue: "true",
			OptionName:  "CLOSE_OPEN_SOURCE_LINK",
			OptionTitle: "是否关闭开源项目入口",
		}, {
			OptionValue: "0",
			OptionName:  "HOUR_REG_NUM",
			OptionTitle: "同一IP每小时允许注册人数",
		}, {
			OptionValue: "0",
			OptionName:  "DAILY_REG_NUM",
			OptionTitle: "同一IP每天允许注册人数",
		}, {
			OptionValue: "X-Real-Ip",
			OptionName:  "REAL_IP_FIELD",
			OptionTitle: "request中获取访客真实IP的header",
		}, {
			OptionValue: "",
			OptionName:  "APP_PAGE",
			OptionTitle: "手机APP下载单页",
		}, {
			OptionValue: "false",
			OptionName:  "HIDE_TAG",
			OptionTitle: "是否隐藏标签在导航栏显示",
		}, {
			OptionValue: "",
			OptionName:  "DOWNLOAD_LIMIT",
			OptionTitle: "是否需要登录才能下载电子书",
		}, {
			OptionValue: "",
			OptionName:  "MOBILE_BANNER_SIZE",
			OptionTitle: "手机端横幅宽高比",
		}, {
			OptionValue: "false",
			OptionName:  "AUTO_HTTPS",
			OptionTitle: "图片链接HTTP转HTTPS",
		}, {
			OptionValue: "0",
			OptionName:  "APP_VERSION",
			OptionTitle: "Android APP版本号（数字）",
		}, {
			OptionValue: "",
			OptionName:  "APP_QRCODE",
			OptionTitle: "是否在用户下载电子书的时候显示APP下载二维码",
		},
	}

	for _, op := range options {
		if !o.QueryTable(m.TableNameWithPrefix()).Filter("option_name", op.OptionName).Exist() {
			if _, err := o.Insert(&op); err != nil {
				return err
			}
		}
	}

	return nil
}
