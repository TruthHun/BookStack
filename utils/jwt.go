package utils

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/astaxie/beego"
	"github.com/dgrijalva/jwt-go"
)

// MediaClaims Media Claims
type MediaClaims struct {
	Path     string
	UnixNano int64
	jwt.StandardClaims
}

var (
	mediaJwtSecret = beego.AppConfig.DefaultString("jwt_secret", "bookstack.cn")
	usedSign       sync.Map
	MediaDuration  int64 = beego.AppConfig.DefaultInt64("media_duration", 3600*5)
)

func init() {
	go func() {
		for {
			time.Sleep(600 * time.Second)
			clearExpireSign()
		}
	}()
}

func clearExpireSign() {
	// 清除超过24小时
	usedSign.Range(func(signMD5, timestamp interface{}) bool {
		if time.Now().Unix()-timestamp.(int64) > MediaDuration {
			usedSign.Delete(signMD5)
		}
		return true
	})
}

// IsSignUsed 签名是否已被使用
func IsSignUsed(sign string) bool {
	signMD5 := MD5Sub16(sign)
	if _, ok := usedSign.Load(signMD5); ok {
		return true
	}
	usedSign.Store(signMD5, time.Now().Unix())
	return false
}

// GenerateSign 生成token
func GenerateMediaSign(path string, unixNano int64, expire ...time.Duration) (sign string, err error) {
	path = strings.TrimLeft(path, "/")
	// 默认过期时间为一个月
	expireDuration := time.Now().Add(30 * 24 * time.Hour)
	if len(expire) > 0 {
		expireDuration = time.Now().Add(expire[0])
	}

	customClaims := &MediaClaims{
		path,
		unixNano,
		jwt.StandardClaims{
			ExpiresAt: expireDuration.Unix(),
		},
	}

	// 将 uid，过期时间作为数据写入 token 中
	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, customClaims)

	// secret 用于对用户数据进行签名，不能暴露
	return jwtToken.SignedString([]byte(mediaJwtSecret))
}

// ParseSign 解析jwt token
func ParseMediaSign(sign string) (path string, err error) {
	var token = &jwt.Token{}
	token, err = jwt.ParseWithClaims(sign, &MediaClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(mediaJwtSecret), nil
	})

	if err != nil {
		return
	}

	if claims, ok := token.Claims.(*MediaClaims); ok && token.Valid {
		return strings.TrimLeft(claims.Path, "/"), nil
	}
	return "", err
}
