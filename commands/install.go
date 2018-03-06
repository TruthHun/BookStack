package commands

import (
	"fmt"
	"os"
	"time"

	"strings"

	"github.com/TruthHun/BookStack/conf"
	"github.com/TruthHun/BookStack/models"
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/orm"
)

//系统安装.
func Install() {

	fmt.Println("Initializing...")

	err := orm.RunSyncdb("default", false, true)
	if err == nil {
		initialization()
	} else {
		panic(err.Error())
		os.Exit(1)
	}
	fmt.Println("Install Successfully!")
	os.Exit(0)

}

func Version() {
	if len(os.Args) >= 2 && os.Args[1] == "version" {
		fmt.Println(conf.VERSION)
		os.Exit(0)
	}
}

//初始化数据
func initialization() {

	err := models.NewOption().Init()

	if err != nil {
		panic(err.Error())
		os.Exit(1)
	}

	member, err := models.NewMember().FindByFieldFirst("account", "admin")
	if err == orm.ErrNoRows {

		member.Account = "admin"
		member.Avatar = beego.AppConfig.String("avatar")
		member.Password = "admin"
		member.AuthMethod = "local"
		member.Nickname = "管理员"
		member.Role = 0
		member.Email = "bookstack@qq.cn"

		if err := member.Add(); err != nil {
			panic("Member.Add => " + err.Error())
			os.Exit(0)
		}

		book := models.NewBook()
		book.MemberId = member.MemberId
		book.BookName = "BookStack"
		book.Status = 0
		book.Description = "这是一个BookStack演示项目，该项目是由系统初始化时自动创建。"
		book.CommentCount = 0
		book.PrivatelyOwned = 0
		book.CommentStatus = "closed"
		book.Identify = "bookstack"
		book.DocCount = 0
		book.CommentCount = 0
		book.Version = time.Now().Unix()
		book.Cover = conf.GetDefaultCover()
		book.Editor = "markdown"
		book.Theme = "default"
		//设置默认时间，因为beego的orm好像无法设置datetime的默认值
		defaultTime, _ := time.Parse("2006-01-02 15:04:05", "2006-01-02 15:04:05")
		book.LastClickGenerate = defaultTime
		book.GenerateTime = defaultTime
		//book.ReleaseTime = defaultTime
		book.ReleaseTime, _ = time.Parse("2006-01-02 15:04:05", "2000-01-02 15:04:05")
		book.Score = 40

		if err := book.Insert(); err != nil {
			panic("Book.Insert => " + err.Error())
			os.Exit(0)
		}
		initSeo()
	}
}

//初始化SEO
func initSeo() {
	sqlslice := []string{"insert ignore into `md_seo`(`id`,`page`,`statement`,`title`,`keywords`,`description`) values ('1','index','首页','书栈网(BookStack.CN)_分享，让知识传承更久远','{keywords}','{description}'),",
		"('2','label_list','标签列表页','{title} - 书栈网(BookStack.CN)','{keywords}','{description}'),",
		"('3','label_content','标签内容页','{title} - 书栈网(BookStack.CN)','{keywords}','{description}'),",
		"('4','book_info','文档信息页','{title} - 书栈网(BookStack.CN)','{keywords}','{description}'),",
		"('5','book_read','文档阅读页','{title} - 书栈网(BookStack.CN)','{keywords}','{description}'),",
		"('6','search_result','搜索结果页','{title} - 书栈网(BookStack.CN)','{keywords}','{description}'),",
		"('7','user_basic','用户基本信息设置页','{title}  - 书栈网(BookStack.CN)','{keywords}','{description}'),",
		"('8','user_pwd','用户修改密码页','{title}  - 书栈网(BookStack.CN)','{keywords}','{description}'),",
		"('9','project_list','项目列表页','{title}  - 书栈网(BookStack.CN)','{keywords}','{description}'),",
		"('11','login','登录页','{title} - 书栈网(BookStack.CN)','{keywords}','{description}'),",
		"('12','reg','注册页','{title} - 书栈网(BookStack.CN)','{keywords}','{description}'),",
		"('13','findpwd','找回密码','{title} - 书栈网(BookStack.CN)','{keywords}','{description}'),",
		"('14','manage_dashboard','仪表盘','{title} - 书栈网(BookStack.CN)','{keywords}','{description}'),",
		"('15','manage_users','用户管理','{title} - 书栈网(BookStack.CN)','{keywords}','{description}'),",
		"('16','manage_users_edit','用户编辑','{title} - 书栈网(BookStack.CN)','{keywords}','{description}'),",
		"('17','manage_project_list','项目列表','{title} - 书栈网(BookStack.CN)','{keywords}','{description}'),",
		"('18','manage_project_edit','项目编辑','{title} - 书栈网(BookStack.CN)','{keywords}','{description}');",
	}
	if _, err := orm.NewOrm().Raw(strings.Join(sqlslice, "")).Exec(); err != nil {
		beego.Error(err.Error())
	}
}
