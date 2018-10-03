package conf

import (
	"strings"

	"github.com/astaxie/beego"
)

type SmtpConf struct {
	EnableMail    bool
	MailNumber    int
	SmtpUserName  string
	SmtpHost      string
	SmtpPassword  string
	SmtpPort      int
	FormUserName  string
	ReplyUserName string
	MailExpired   int
}

func GetMailConfig() *SmtpConf {
	username := beego.AppConfig.String("smtp_user_name")
	password := beego.AppConfig.String("smtp_password")
	smtpHost := beego.AppConfig.String("smtp_host")
	smtpPort := beego.AppConfig.DefaultInt("smtp_port", 25)
	formUsername := beego.AppConfig.String("form_user_name")
	replyUsername := beego.AppConfig.String("reply_user_name")
	enableMail := beego.AppConfig.String("enable_mail")
	mailNumber := beego.AppConfig.DefaultInt("mail_number", 5)

	c := &SmtpConf{
		EnableMail:    strings.EqualFold(enableMail, "true"),
		MailNumber:    mailNumber,
		SmtpUserName:  username,
		SmtpHost:      smtpHost,
		SmtpPassword:  password,
		FormUserName:  formUsername,
		ReplyUserName: replyUsername,
		SmtpPort:      smtpPort,
	}
	return c
}
