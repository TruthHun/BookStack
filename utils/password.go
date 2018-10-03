package utils

import (
	"crypto/md5"
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"io"
	mt "math/rand"
	"strconv"
	"strings"
)

const (
	saltSize           = 16
	delimiter          = "$"
	stretchingPassword = 500
	saltLocalSecret    = "ahfw*&TGdsfnbi*^Wt"
)

//加密密码
func PasswordHash(pass string) (string, error) {

	saltSecret, err := saltSecret()
	if err != nil {
		return "", err
	}

	salt, err := salt(saltLocalSecret + saltSecret)
	if err != nil {
		return "", err
	}

	interation := randInt(1, 20)

	hash, err := hash(pass, saltSecret, salt, int64(interation))
	if err != nil {
		return "", err
	}

	interationStr := strconv.Itoa(interation)
	password := saltSecret + delimiter + interationStr + delimiter + hash + delimiter + salt

	return password, nil

}

//校验密码是否有效
func PasswordVerify(hashing string, pass string) (bool, error) {
	data := trimSaltHash(hashing)

	interation, _ := strconv.ParseInt(data["interation_string"], 10, 64)

	has, err := hash(pass, data["salt_secret"], data["salt"], int64(interation))
	if err != nil {
		return false, err
	}

	if (data["salt_secret"] + delimiter + data["interation_string"] + delimiter + has + delimiter + data["salt"]) == hashing {
		return true, nil
	}
	return false, nil
}

func hash(pass string, saltSecret string, salt string, interation int64) (string, error) {
	var passSalt = saltSecret + pass + salt + saltSecret + pass + salt + pass + pass + salt
	var i int

	hashPass := saltLocalSecret
	hashStart := sha512.New()
	hashCenter := sha256.New()
	hashOutput := sha256.New224()

	i = 0
	for i <= stretchingPassword {
		i = i + 1
		hashStart.Write([]byte(passSalt + hashPass))
		hashPass = hex.EncodeToString(hashStart.Sum(nil))
	}

	i = 0
	for int64(i) <= interation {
		i = i + 1
		hashPass = hashPass + hashPass
	}

	i = 0
	for i <= stretchingPassword {
		i = i + 1
		hashCenter.Write([]byte(hashPass + saltSecret))
		hashPass = hex.EncodeToString(hashCenter.Sum(nil))
	}
	hashOutput.Write([]byte(hashPass + saltLocalSecret))
	hashPass = hex.EncodeToString(hashOutput.Sum(nil))

	return hashPass, nil
}

func trimSaltHash(hash string) map[string]string {
	str := strings.Split(hash, delimiter)
	return map[string]string{
		"salt_secret":       str[0],
		"interation_string": str[1],
		"hash":              str[2],
		"salt":              str[3],
	}
}
func salt(secret string) (string, error) {

	buf := make([]byte, saltSize, saltSize+md5.Size)
	_, err := io.ReadFull(rand.Reader, buf)
	if err != nil {
		return "", err
	}

	hash := md5.New()
	hash.Write(buf)
	hash.Write([]byte(secret))
	return hex.EncodeToString(hash.Sum(buf)), nil
}

func saltSecret() (string, error) {
	rb := make([]byte, randInt(10, 100))
	_, err := rand.Read(rb)
	if err != nil {
		return "", err
	}

	return base64.URLEncoding.EncodeToString(rb), nil
}

func randInt(min int, max int) int {
	return min + mt.Intn(max-min)
}
