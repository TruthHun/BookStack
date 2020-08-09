package models

import (
	"strings"
	"sync"

	"github.com/TruthHun/BookStack/conf"
	"github.com/astaxie/beego/orm"
)

var optionCache sync.Map // map[int || string]*Option

// Option struct .
type Option struct {
	OptionId    int    `orm:"column(option_id);pk;auto;unique;" json:"option_id"`
	OptionTitle string `orm:"column(option_title);size(500)" json:"option_title"`
	OptionName  string `orm:"column(option_name);unique;size(80)" json:"option_name"`
	OptionValue string `orm:"column(option_value);type(text);null" json:"option_value"`
	Remark      string `orm:"column(remark);type(text);null" json:"remark"`
}

func initOptionCache() {
	opts, _ := NewOption().All()
	for _, opt := range opts {
		optionCache.Store(opt.OptionName, opt)
		optionCache.Store(opt.OptionId, opt)
	}
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

	if val, ok := optionCache.Load(id); ok {
		p = val.(*Option)
		return p, nil
	}

	o := orm.NewOrm()
	p.OptionId = id
	if err := o.Read(p); err != nil {
		return p, err
	}
	return p, nil
}

func (p *Option) FindByKey(key string) (*Option, error) {

	if val, ok := optionCache.Load(key); ok {
		p = val.(*Option)
		return p, nil
	}

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
	defer func() {
		initOptionCache()
	}()
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
	initOptionCache()
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
			OptionName:  "SHOW_CATEGORY_INDEX",
			OptionTitle: "首页是否显示分类索引",
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
			OptionTitle: "是否都可以创建书籍",
		}, {
			OptionValue: "false",
			OptionName:  "CLOSE_SUBMIT_ENTER",
			OptionTitle: "是否关闭收录入口",
		}, {
			OptionValue: "true",
			OptionName:  "CLOSE_OPEN_SOURCE_LINK",
			OptionTitle: "是否关闭开源书籍入口",
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
		{
			OptionValue: "5",
			OptionName:  "SIGN_BASIC_REWARD",
			OptionTitle: "用户每次签到基础奖励阅读时长(秒)",
		},
		{
			OptionValue: "10",
			OptionName:  "SIGN_APP_REWARD",
			OptionTitle: "使用APP签到额外奖励阅读时长(秒)",
		},
		{
			OptionValue: "0",
			OptionName:  "SIGN_CONTINUOUS_REWARD", //
			OptionTitle: "用户连续签到奖励阅读时长(秒)",
		}, {
			OptionValue: "0",
			OptionName:  "SIGN_CONTINUOUS_MAX_REWARD",
			OptionTitle: "连续签到奖励阅读时长上限(秒)",
		},
		{
			OptionValue: "0",
			OptionName:  "READING_MIN_INTERVAL",
			OptionTitle: "内容最小阅读计时间隔(秒)",
		},
		{
			OptionValue: "600",
			OptionName:  "READING_MAX_INTERVAL",
			OptionTitle: "内容最大阅读计时间隔(秒)",
		},
		{
			OptionValue: "1200",
			OptionName:  "READING_INVALID_INTERVAL",
			OptionTitle: "内容阅读无效计时间隔(秒)",
		},
		{
			OptionValue: "600",
			OptionName:  "READING_INTERVAL_MAX_REWARD",
			OptionTitle: "内容阅读计时间隔最大奖励(秒)",
		},
		{
			OptionValue: "false",
			OptionName:  "COLLAPSE_HIDE",
			OptionTitle: "目录是否默认收起",
		},
		{
			OptionValue: "",
			OptionName:  "FORBIDDEN_REFERER",
			OptionTitle: "禁止的Referer",
		},
		{
			OptionValue: "",
			OptionName:  "CheckingAppVersion",
			OptionTitle: "审核中的APP版本",
		},
		{
			OptionValue: "Android, 安卓",
			OptionName:  "CheckingForbidWords",
			OptionTitle: "iOS APP提交审核时屏蔽的关键字",
		},
		{
			OptionValue: "1",
			OptionName:  "DOWNLOAD_INTERVAL",
			OptionTitle: "每阅读多少秒可以下载一个电子书",
		},
	}

	for _, op := range options {
		if !o.QueryTable(m.TableNameWithPrefix()).Filter("option_name", op.OptionName).Exist() {
			if _, err := o.Insert(&op); err != nil {
				return err
			}
		}
	}
	initOptionCache()
	return nil
}

func (m *Option) ForbiddenReferer() []string {
	return strings.Split(GetOptionValue("FORBIDDEN_REFERER", ""), "\n")
}

func (m *Option) IsResponseEmptyForAPP(requestVersion, word string) (yes bool) {
	version := GetOptionValue("CheckingAppVersion", "")
	if version == "" {
		return
	}
	if strings.ToLower(strings.TrimSpace(requestVersion)) == version {
		words := strings.Split(GetOptionValue("CheckingForbidWords", ""), ",")
		word = strings.ToLower(strings.TrimSpace(word))
		for _, item := range words {
			item = strings.ToLower(strings.TrimSpace(item))
			if strings.Contains(word, item) {
				return true
			}
		}
	}
	return
}
