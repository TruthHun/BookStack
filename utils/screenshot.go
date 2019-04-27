package utils

import "sync"

// 截图信息
type ScreenShotInfo struct {
	Data []struct {
		Height float64 `json:"height"`
		Width  float64 `json:"width"`
		X      float64 `json:"x"`
		Y      float64 `json:"y"`
	} `json:"data"`
	BodyHeight float64 `json:"height"`
}

//
type ScreenShot struct {
	Selector string
}

var screenShotProjects sync.Map
