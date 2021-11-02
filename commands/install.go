package commands

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/TruthHun/BookStack/conf"
	"github.com/TruthHun/BookStack/models"
	"github.com/TruthHun/BookStack/utils"
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
	initSeo()
	resetCategoryUniqueIndex()
	migrateEbook()
	fmt.Println("Install Successfully!")
	os.Exit(0)
}

// 删除分类中的title的唯一索引（兼容旧版本）
func resetCategoryUniqueIndex() {
	var indexs []struct {
		Table      string `orm:"column(Table)"`
		NonUnique  int    `orm:"column(Non_unique)"`
		KeyName    string `orm:"column(Key_name)"`
		ColumnName string `orm:"column(Column_name)"`
	}
	showIndex := "SHOW INDEX FROM md_category"
	dropIndex := "ALTER TABLE `md_category` DROP INDEX `%s`;"
	addUniqueIndex := "ALTER TABLE `md_category` ADD UNIQUE( `pid`, `title`);"
	o := orm.NewOrm()
	o.Raw(showIndex).QueryRows(&indexs)
	for _, index := range indexs {
		if index.ColumnName == "title" {
			o.Raw(fmt.Sprintf(dropIndex, index.KeyName)).Exec()
		}
	}
	o.Raw(addUniqueIndex).Exec()
}

func Version() {
	if len(os.Args) >= 2 && os.Args[1] == "version" {
		fmt.Println(conf.VERSION)
		os.Exit(0)
	}
}

//初始化数据
func initialization() {
	models.InstallAdsPosition()
	err := models.NewOption().Init()
	if err != nil {
		panic(err.Error())
		os.Exit(1)
	}

	member, err := models.NewMember().FindByFieldFirst("account", "admin")
	if err == orm.ErrNoRows {
		member.Account = "admin"
		member.Avatar = beego.AppConfig.String("avatar")
		member.Password = "admin888"
		member.AuthMethod = "local"
		member.Nickname = "管理员"
		member.Role = 0
		member.Email = "bookstack@qq.cn"

		if err := member.Add(); err != nil {
			beego.Error("Member.Add => " + err.Error())
		}

		book := models.NewBook()
		book.MemberId = member.MemberId
		book.BookName = "BookStack"
		book.Status = 0
		book.Description = "这是一个BookStack演示书籍，该书籍是由系统初始化时自动创建。"
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
			beego.Error("Book.Insert => " + err.Error())
		}
	}
}

//初始化SEO
func initSeo() {
	sqlslice := []string{"insert ignore into `md_seo`(`id`,`page`,`statement`,`title`,`keywords`,`description`) values ('1','index','发现','书栈网(BookStack.CN)_分享，让知识传承更久远','{keywords}','{description}'),",
		"('2','label_list','标签列表页','{title} - 书栈网(BookStack.CN)','{keywords}','{description}'),",
		"('3','label_content','标签内容页','{title} - 书栈网(BookStack.CN)','{keywords}','{description}'),",
		"('4','book_info','文档信息页','{title} - 书栈网(BookStack.CN)','{keywords}','{description}'),",
		"('5','book_read','文档阅读页','{title} - 书栈网(BookStack.CN)','{keywords}','{description}'),",
		"('6','search_result','搜索结果页','{title} - 书栈网(BookStack.CN)','{keywords}','{description}'),",
		"('7','user_basic','用户基本信息设置页','{title}  - 书栈网(BookStack.CN)','{keywords}','{description}'),",
		"('8','user_pwd','用户修改密码页','{title}  - 书栈网(BookStack.CN)','{keywords}','{description}'),",
		"('9','project_list','书籍列表页','{title}  - 书栈网(BookStack.CN)','{keywords}','{description}'),",
		"('11','login','登录页','{title} - 书栈网(BookStack.CN)','{keywords}','{description}'),",
		"('12','reg','注册页','{title} - 书栈网(BookStack.CN)','{keywords}','{description}'),",
		"('13','findpwd','找回密码','{title} - 书栈网(BookStack.CN)','{keywords}','{description}'),",
		"('14','manage_dashboard','仪表盘','{title} - 书栈网(BookStack.CN)','{keywords}','{description}'),",
		"('15','manage_users','用户管理','{title} - 书栈网(BookStack.CN)','{keywords}','{description}'),",
		"('16','manage_users_edit','用户编辑','{title} - 书栈网(BookStack.CN)','{keywords}','{description}'),",
		"('17','manage_project_list','书籍列表','{title} - 书栈网(BookStack.CN)','{keywords}','{description}'),",
		"('18','manage_project_edit','书籍编辑','{title} - 书栈网(BookStack.CN)','{keywords}','{description}'),",
		"('19','cate','首页','{title} - 书栈网(BookStack.CN)','{keywords}','{description}'),",
		"('20','ucenter-share','用户主页','{title} - 书栈网(BookStack.CN)','{keywords}','{description}'),",
		"('21','ucenter-collection','用户收藏','{title} - 书栈网(BookStack.CN)','{keywords}','{description}'),",
		"('22','ucenter-fans','用户粉丝','{title} - 书栈网(BookStack.CN)','{keywords}','{description}'),",
		"('23','ucenter-follow','用户关注','{title} - 书栈网(BookStack.CN)','{keywords}','{description}');",
	}
	if _, err := orm.NewOrm().Raw(strings.Join(sqlslice, "")).Exec(); err != nil {
		beego.Error(err.Error())
	}

	// 为了兼容升级。以前的index表示首页，cate表示分类，现在反过来了。
	items := []models.Seo{
		models.Seo{
			Statement: "发现",
			Id:        1,
		},
		models.Seo{
			Statement: "首页",
			Id:        19,
		},
	}
	for _, item := range items {
		orm.NewOrm().Update(&item, "statement")
	}
}

// 电子书数据迁移
func migrateEbook() {
	// 1. 查找 md_ebook 表中不存在的书籍的电子书
	var (
		books    []models.Book
		sqlQuery = "SELECT book_id,generate_time,book_name,identify,label,description FROM `md_books` WHERE `generate_time`>'2010' and book_id not in (select book_id from md_ebook where book_id>0 group by book_id)"
	)
	o := orm.NewOrm()
	o.Raw(sqlQuery).QueryRows(&books)
	if len(books) == 0 {
		beego.Info("不存在需要同步的电子书数据")
		return
	}
	exts := []string{".pdf", ".epub", ".mobi"}
	for _, book := range books {
		beego.Info("迁移书籍电子书：", book.BookName)
		var ebooks []models.Ebook
		bookPrefix := fmt.Sprintf("/projects/%v/books/%v", book.Identify, book.GenerateTime.Unix())
		if utils.StoreType == utils.StoreLocal {
			bookPrefix = "/uploads" + bookPrefix
		}
		ebook := models.Ebook{
			Title:       book.BookName,
			Keywords:    book.Label,
			Description: beego.Substr(book.Description, 0, 255),
			BookID:      book.BookId,
			Status:      models.EBookStatusSuccess,
		}
		for _, ext := range exts {
			ebook.Ext = ext
			ebook.Path = bookPrefix + ext
			ebooks = append(ebooks, ebook)
		}
		if _, err := o.InsertMulti(len(ebooks), &ebooks); err != nil {
			beego.Error(err)
		}
	}
	beego.Info("电子书数据同步中...")
}
