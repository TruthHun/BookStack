package models

type PrintBook struct {
	Id          int    `json:"id"`
	Cover       string `json:"cover"`
	Title       string `json:"title"`
	Keywords    string `json:"keywords"`
	Description string `json:"description"`
	Author      string `json:"author"`
	Publish     string `json:"publish"` //出版社
	Lang        string `json:"lang"`    //语种
	Page        int    `json:"page"`    //页数
	View        int    `json:"view"`
	Support     int    `json:"support"`
	ShopLinks   []struct {
		Name string `json:"name"`
		Href string `json:"href"`
	} `orm:"-" json:"shop_links"`
	ShopLinksJSON string `orm:"type(text)"`
	Content       string `json:"content" orm:"type(longtext)"` // markdown content
}
