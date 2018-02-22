package models

var TableSeo = "md_seo"

// SEO struct .
type Seo struct {
	Id          int    //自增主键
	Page        string `orm:"unique;size(50)"` //页面
	Statement   string //页面说明
	Title       string `orm:"default({title})"`       //SEO标题
	Keywords    string `orm:"default({keywords})"`    //SEO关键字
	Description string `orm:"default({description})"` //SEO摘要
}
