package utils

import (
	"time"

	"github.com/astaxie/beego"
)

var installedDependencies []installedDependency

type installedDependency struct {
	Name        string // 依赖名称
	IsInstalled bool   // 是否已安装
	Message     string // 相关信息
	Error       string
	CheckedAt   string // 上次检测时间
}

func init() {
	go checkInstalledDependencyData()
}

func GetInstalledDependencies() []installedDependency {
	return installedDependencies
}

func checkInstalledDependencyData() {
	var (
		err        error
		dateLayout = "2006-01-02 15:04:05"
	)

	nameCalibre := "calibre"
	errCalibre := "-"
	if err = IsInstalledCalibre("ebook-convert"); err != nil {
		errCalibre = err.Error()
	}
	installedDependencies = append(installedDependencies, installedDependency{
		Name:        nameCalibre,
		IsInstalled: err == nil,
		Error:       errCalibre,
		Message:     "calibre 用于将书籍转换成PDF、epub和mobi ==> <a class='text-danger' target='_blank' href='https://www.bookstack.cn/read/help/Ubuntu.md'>安装教程</a>。如果未安装该模块，则无法生成电子书和提供电子书下载。",
		CheckedAt:   time.Now().Format(dateLayout),
	})

	errGit := "-"
	nameGit := "git"
	if err = IsInstalledGit(); err != nil {
		errGit = err.Error()
	}
	installedDependencies = append(installedDependencies, installedDependency{
		Name:        nameGit,
		IsInstalled: err == nil,
		Error:       errGit,
		Message:     "git，用于git clone方式导入项目。如果未安装该模块，则无法使用该方式导入项目。",
		CheckedAt:   time.Now().Format(dateLayout),
	})

	errChrome := "-"
	nameChrome := "chrome"
	if err = IsInstalledChrome(beego.AppConfig.DefaultString("chrome", "chrome")); err != nil {
		errChrome = err.Error()
	}
	installedDependencies = append(installedDependencies, installedDependency{
		Name:        nameChrome,
		IsInstalled: err == nil,
		Error:       errChrome,
		Message:     "chrome浏览器，即谷歌浏览器，或者chromium-browser，用于渲染markdown内容为HTML。",
		CheckedAt:   time.Now().Format(dateLayout),
	})

	namePuppeteer := "puppeteer"
	errPuppeteer := "-"
	if err = IsInstalledPuppetter(); err != nil {
		errPuppeteer = err.Error()
	}
	installedDependencies = append(installedDependencies, installedDependency{
		Name:        namePuppeteer,
		IsInstalled: err == nil,
		Error:       errPuppeteer,
		Message:     "node.js的模块，用于将markdown渲染为HTML以及生成电子书封面。 <a class='text-danger' target='_blank' href='https://www.bookstack.cn/read/help/Ubuntu.md'>安装教程</a>",
		CheckedAt:   time.Now().Format(dateLayout),
	})
}

// IsInstalledPuppetter 是否安装了puppeteer
func IsInstalledPuppetter() (err error) {
	// 检测全局是否安装了puppeteer
	_, err = ExecCommand("npm", []string{"ls", "-g", "--depth=0", "puppeteer"})
	if err == nil {
		return
	}

	// 检测项目是否安装了puppeteer
	_, err = ExecCommand("npm", []string{"ls", "--depth=0", "puppeteer"})
	return
}

// IsInstalledGit 是否安装了Git
func IsInstalledGit() (err error) {
	_, err = ExecCommand("git", []string{"--version"})
	return
}

// IsInstalledChrome 是否安装了Chrome
func IsInstalledChrome(chrome string) (err error) {
	_, err = ExecCommand(chrome, []string{"--version"})
	return
}

// IsInstalledCalibre 是否安装了calibre
func IsInstalledCalibre(calibre string) (err error) {
	_, err = ExecCommand(calibre, []string{"--version"})
	return
}
