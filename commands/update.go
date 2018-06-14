package commands

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/TruthHun/BookStack/conf"
	"github.com/astaxie/beego"
)

//检查最新版本.
func CheckUpdate() {

	resp, err := http.Get("https://api.github.com/repos/TruthHun/BookStack/tags")

	if err != nil {
		beego.Error("CheckUpdate => ", err)
		os.Exit(1)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		beego.Error("CheckUpdate => ", err)
		os.Exit(1)
	}

	var result []*struct {
		Name string `json:"name"`
	}

	err = json.Unmarshal(body, &result)
	fmt.Println("MinDoc current version => ", conf.VERSION)
	if err != nil {
		beego.Error("CheckUpdate => ", err)
		os.Exit(0)
	}

	if len(result) > 0 {
		fmt.Println("MinDoc last version => ", result[0].Name)
	}

	os.Exit(0)

}
