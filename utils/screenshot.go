package utils

import (
	"os"
	"sync"
)

// 截图信息
type ScreenShotInfo struct {
	Data [][]struct {
		Height float64 `json:"height"`
		Width  float64 `json:"width"`
		X      float64 `json:"x"`
		Y      float64 `json:"y"`
	} `json:"data"`
	Height float64 `json:"height"`
}

var ScreenShotProjects sync.Map // map[bookIdentify]selector，书籍标识和书籍元素选择器

func DeleteScreenShot(bookIdentify string) {
	os.RemoveAll("cache/screenshots/" + bookIdentify)
}
