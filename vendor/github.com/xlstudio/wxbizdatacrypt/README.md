### 微信小程序加密数据解密算法Go语言版

**微信小程序开发官方说明文档 - 签名加密篇文档链接**

***1. 最早的文档链接（已跳转，可能随时会失效）***
> http://mp.weixin.qq.com/debug/wxadoc/dev/api/signature.html?t=20161107

***2. 旧文档链接（已暂停更新，可能随时会失效）***
> https://developers.weixin.qq.com/miniprogram/dev/api/signature.html

***3. 新文档链接（当前官方最新文档）***
> https://developers.weixin.qq.com/miniprogram/dev/framework/open-ability/signature.html

**使用方法**

> go get github.com/xlstudio/wxbizdatacrypt

**引入方法**
```Go
import (
	"github.com/xlstudio/wxbizdatacrypt"
)
```
**使用示例**

```Go
package main

import (
	"fmt"
	"github.com/xlstudio/wxbizdatacrypt"
)

func main() {
	appID := "wx4f4bc4dec97d474b"
	sessionKey := "tiihtNczf5v6AKRyjwEUhQ=="
	encryptedData := "CiyLU1Aw2KjvrjMdj8YKliAjtP4gsMZMQmRzooG2xrDcvSnxIMXFufNstNGTyaGS9uT5geRa0W4oTOb1WT7fJlAC+oNPdbB+3hVbJSRgv+4lGOETKUQz6OYStslQ142dNCuabNPGBzlooOmB231qMM85d2/fV6ChevvXvQP8Hkue1poOFtnEtpyxVLW1zAo6/1Xx1COxFvrc2d7UL/lmHInNlxuacJXwu0fjpXfz/YqYzBIBzD6WUfTIF9GRHpOn/Hz7saL8xz+W//FRAUid1OksQaQx4CMs8LOddcQhULW4ucetDf96JcR3g0gfRK4PC7E/r7Z6xNrXd2UIeorGj5Ef7b1pJAYB6Y5anaHqZ9J6nKEBvB4DnNLIVWSgARns/8wR2SiRS7MNACwTyrGvt9ts8p12PKFdlqYTopNHR1Vf7XjfhQlVsAJdNiKdYmYVoKlaRv85IfVunYzO0IKXsyl7JCUjCpoG20f0a04COwfneQAGGwd5oa+T8yO5hzuyDb/XcxxmK01EpqOyuxINew=="
	iv := "r7BXXKkLb8qrSNn05n0qiA=="

	pc := wxbizdatacrypt.WxBizDataCrypt{AppID: appID, SessionKey: sessionKey}
	result, err := pc.Decrypt(encryptedData, iv, true) //第三个参数解释： 需要返回 JSON 数据类型时 使用 true, 需要返回 map 数据类型时 使用 false
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(result)
	}
}
```


