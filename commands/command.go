package commands

import (
	"database/sql"
	"encoding/gob"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"encoding/json"
	"log"

	"github.com/TruthHun/BookStack/commands/migrate"
	"github.com/TruthHun/BookStack/conf"
	"github.com/TruthHun/BookStack/models"
	"github.com/TruthHun/BookStack/utils"
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/logs"
	"github.com/astaxie/beego/orm"
	"github.com/lifei6671/gocaptcha"
)

var (
	ConfigurationFile = "./conf/app.conf"
	LogFile           = "./logs"
)

// RegisterDataBase 注册数据库
func RegisterDataBase() {

	host := beego.AppConfig.String("db_host")
	database := beego.AppConfig.String("db_database")
	username := beego.AppConfig.String("db_username")
	password := beego.AppConfig.String("db_password")

	port := beego.AppConfig.String("db_port")

	createDB := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s` DEFAULT CHARSET utf8mb4 COLLATE utf8mb4_general_ci", database)
	conn := fmt.Sprintf("%s:%s@tcp(%s:%s)/", username, password, host, port)
	db, err := sql.Open("mysql", conn)
	if err != nil {
		panic(err)
	}
	_, err = db.Exec(createDB)
	if err != nil {
		panic(err)
	}

	dataSource := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=true&loc=Local", username, password, host, port, database)

	orm.RegisterDataBase("default", "mysql", dataSource)

	if beego.AppConfig.String("runmode") == "dev" {
		orm.Debug = true
	}
}

// RegisterModel 注册Model
func RegisterModel() {
	orm.RegisterModelWithPrefix(conf.GetDatabasePrefix(),
		new(models.Member),
		new(models.Book),
		new(models.Relationship),
		new(models.Option),
		new(models.Document),
		new(models.Attachment),
		new(models.Logger),
		new(models.MemberToken),
		new(models.DocumentHistory),
		new(models.Migration),
		new(models.Label),
		new(models.Seo),
		new(models.Star),
		new(models.Score),
		new(models.Comments),
		new(models.Gitee),
		new(models.Github),
		new(models.QQ),
		new(models.DocumentStore),
		new(models.Category),
		new(models.BookCategory),
		new(models.Fans),
		new(models.FriendLink),
		new(models.ReadCount),
		new(models.ReadRecord),
		new(models.Bookmark),
		new(models.SubmitBooks),
		models.NewRelateBook(),
		models.NewAuth(),
		models.NewBanner(),
		models.NewWechatCode(),
		models.NewWechat(),
		new(models.RegLimit),
		models.NewAdsPosition(),
		models.NewAdsCont(),
		models.NewReadingTime(),
		models.NewSign(),
		models.NewBookCounter(),
		models.NewDownloadCounter(),
	)
	migrate.RegisterMigration()
}

// RegisterLogger 注册日志
func RegisterLogger(log string) {

	logs.SetLogFuncCall(true)
	logs.SetLogger("console")
	logs.EnableFuncCallDepth(true)
	logs.Async()

	logPath := filepath.Join(log, "log.log")

	if _, err := os.Stat(logPath); os.IsNotExist(err) {

		os.MkdirAll(log, 0777)

		if f, err := os.Create(logPath); err == nil {
			f.Close()
			config := make(map[string]interface{}, 1)

			config["filename"] = logPath

			b, _ := json.Marshal(config)

			beego.SetLogger("file", string(b))
		}
	}

	beego.SetLogFuncCall(true)
	beego.BeeLogger.Async()
}

// RunCommand 注册orm命令行工具
func RegisterCommand() {

	if len(os.Args) >= 2 && os.Args[1] == "install" {
		ResolveCommand(os.Args[2:])
		Install()
	} else if len(os.Args) >= 2 && os.Args[1] == "version" {
		ResolveCommand(os.Args[2:])
		CheckUpdate()
	} else if len(os.Args) >= 2 && os.Args[1] == "migrate" {
		ResolveCommand(os.Args[2:])
		migrate.RunMigration()
	}
}

func RegisterFunction() {
	beego.AddFuncMap("config", models.GetOptionValue)

	beego.AddFuncMap("cdn", func(p string) string {
		cdn := beego.AppConfig.DefaultString("cdn", "")
		if strings.HasPrefix(p, "/") && strings.HasSuffix(cdn, "/") {
			return cdn + string(p[1:])
		}
		if !strings.HasPrefix(p, "/") && !strings.HasSuffix(cdn, "/") {
			return cdn + "/" + p
		}
		return cdn + p
	})

	beego.AddFuncMap("cdnjs", func(p string) string {
		cdn := beego.AppConfig.DefaultString("cdnjs", "")
		if strings.HasPrefix(p, "/") && strings.HasSuffix(cdn, "/") {
			return cdn + string(p[1:])
		}
		if !strings.HasPrefix(p, "/") && !strings.HasSuffix(cdn, "/") {
			return cdn + "/" + p
		}
		return cdn + p
	})
	beego.AddFuncMap("cdncss", func(p string) string {
		cdn := beego.AppConfig.DefaultString("cdncss", "")
		if strings.HasPrefix(p, "/") && strings.HasSuffix(cdn, "/") {
			return cdn + string(p[1:])
		}
		if !strings.HasPrefix(p, "/") && !strings.HasSuffix(cdn, "/") {
			return cdn + "/" + p
		}
		return cdn + p
	})
	beego.AddFuncMap("cdnimg", func(p string) string {
		cdn := beego.AppConfig.DefaultString("cdnimg", "")
		if strings.HasPrefix(p, "/") && strings.HasSuffix(cdn, "/") {
			return cdn + string(p[1:])
		}
		if !strings.HasPrefix(p, "/") && !strings.HasSuffix(cdn, "/") {
			return cdn + "/" + p
		}
		return cdn + p
	})
	beego.AddFuncMap("getUsernameByUid", func(id interface{}) string {
		return new(models.Member).GetUsernameByUid(id)
	})
	beego.AddFuncMap("getNicknameByUid", func(id interface{}) string {
		return new(models.Member).GetNicknameByUid(id)
	})
	beego.AddFuncMap("inMap", utils.InMap)
	//将标签转成a链接
	beego.AddFuncMap("tagsToLink", func(tags string) (links string) {
		var linkArr []string
		if slice := strings.Split(tags, ","); len(slice) > 0 {
			for _, tag := range slice {
				if tag = strings.TrimSpace(tag); len(tag) > 0 {
					linkArr = append(linkArr, fmt.Sprintf(`<a target="_blank" title="%v" href="%v">%v</a>`, tag, beego.URLFor("LabelController.Index", ":key", tag), tag))
				}
			}
		}
		return strings.Join(linkArr, "")
	})

	//用户是否收藏了文档
	beego.AddFuncMap("doesStar", new(models.Star).DoesStar)
	beego.AddFuncMap("scoreFloat", utils.ScoreFloat)
	beego.AddFuncMap("showImg", utils.ShowImg)
	beego.AddFuncMap("IsFollow", new(models.Fans).Relation)
	beego.AddFuncMap("isubstr", utils.Substr)
	beego.AddFuncMap("ads", models.GetAdsCode)
	beego.AddFuncMap("formatReadingTime", utils.FormatReadingTime)
	beego.AddFuncMap("add", func(a, b int) int { return a + b })
}

func ResolveCommand(args []string) {
	flagSet := flag.NewFlagSet("MinDoc command: ", flag.ExitOnError)
	flagSet.StringVar(&ConfigurationFile, "config", "", "BookStack configuration file.")
	flagSet.StringVar(&LogFile, "log", "", "BookStack log file path.")

	flagSet.Parse(args)

	if ConfigurationFile == "" {
		ConfigurationFile = filepath.Join("conf", "app.conf")
		config := filepath.Join("conf", "app.conf.example")
		if !utils.FileExists(ConfigurationFile) && utils.FileExists(config) {
			utils.CopyFile(ConfigurationFile, config)
		}
	}
	gocaptcha.ReadFonts(filepath.Join("static", "fonts"), ".ttf")

	err := beego.LoadAppConfig("ini", ConfigurationFile)

	if err != nil {
		log.Println("An error occurred:", err)
		os.Exit(1)
	}
	uploads := filepath.Join("uploads")

	os.MkdirAll(uploads, 0666)

	beego.BConfig.WebConfig.StaticDir["/static"] = filepath.Join("static")
	// beego.BConfig.WebConfig.StaticDir["/uploads"] = uploads
	beego.BConfig.WebConfig.ViewsPath = filepath.Join("views")

	fonts := filepath.Join("static", "fonts")

	if !utils.FileExists(fonts) {
		log.Fatal("Font path not exist.")
	}
	gocaptcha.ReadFonts(filepath.Join("static", "fonts"), ".ttf")

	RegisterDataBase()
	RegisterModel()
	RegisterLogger(LogFile)

}

func init() {

	gocaptcha.ReadFonts("./static/fonts", ".ttf")
	gob.Register(models.Member{})
}
