# BookStack API

BookStack 配套微信小程序 BookChat API。

## TODO

- [x] 评论管理
- [ ] 用户评论权限（默认都有评论权限，但是一旦被禁用评论权限，则不能再进行评论）
- [ ] 用户写作权限（默认所有人都有写作权限，被禁用之后，评论权限和点评权限一并无法使用）
- [ ] 列表数据查询，可自定义每页查询的数量
- [ ] 管理后台用户管理增加用户搜索，以便管理用户
- [ ] 相关书籍接口
- [ ] 404 页面，增加返回上一页
- [ ] 分类图标、用户头像图片裁剪功能
- [ ] 增加一次获取多个分类的书籍列表的接口
- [ ] 没有书籍封面、用户头像以及分类图标的时候，返回默认的图片。同时，微信小程序也做好没有图片时候的默认图片处理
- [ ] 书籍发布时间和更新时间不对。每次老是自动更新
- [ ] 采集程序，a标签，如果是以 mailto 或者是 tel 开头的，不要加 https 或者 http
- [ ] 存储第三方登录的用户头像（如 GitHub、Gitee、QQ 等用户头像，特别是GitHub,头像图片加载很慢）
- [ ] 微信小程序，按照最近阅读倒序排序，并返回最后的阅读时间（用户每浏览一个收藏的书籍的章节，则更新收藏的书籍的最后时间）

- [ ] 读书时长功能
    - [ ] 增加用户阅读时长，记录用户读书时间
    - [ ] 增加阅读排行榜，显示用户阅读时间榜单（总时长，最近一年、最近一个月、最近一周、最近一天）
    - [ ] 记录用户阅读每一本书的阅读时长，哪怕没有收藏这本书

- [ ] 增加语音朗读功能，用耳朵读书
- [ ] 增加发放通知的接口（或者微信小程序消息通知接口）
- [ ] API接口登录次数限制
- [ ] 导出markdown功能
- [ ] IP请求限制
- [ ] 小程序所有触底请求和下拉刷新，都加上`pending`,以表示数据正在请求，避免不断发送请求
- [ ] 书架还有问题
- [ ] 查询书籍信息的时候，顺便一同返回有没有收藏该书籍（即加入到书架）
- [ ] last-modified 实现 HTTP 缓存

## 功能

`BookChat` v2.0 微信小程序( https://gitee.com/truthhun/BookChat )已经发布了，需要配套 `BookStack` v2.0 以上版本的接口才能正常使用，
目前`BookStack`相关API已经开发完成，但是对API等的后台管理功能并未完善，
快的话也需要大半个月时间这样，所以先放出Beta版本，以方便需要调试和对BookChat进行二次开发的朋友。

## 本次主要升级日志

- [x] `BookStack` 配套微信小程序 `BookChat` API接口实现，累计20+个API接口
- [x] 修复删除书籍时误删默认封面的bug
- [x] HTML内容处理，以兼容微信小程序`rich-text`组件实现微信小程序文档内容渲染
- [ ] 开源书籍和文档收录提交入口以及收录管理
- [x] 增加网站小程序码功能，打通PC端与移动端一体化阅读浏览
- [x] 内容采集增强和优化
- [x] 评论审核与管理
- [x] 微信小程序配置(在 `app.conf` 文件中)
- [x] 横幅管理
- [x] 支持 `epub` 导入
- [x] 隐藏附件管理入口(因为不依赖于此管理附件)
- [x] 管理后台可根据用户名、昵称、邮箱和角色对用户进行检索和管理
- [x] 增加`作者`角色，用于控制普通用户创建书籍权限

> 更多升级内容，请查看源码仓库 commit 记录


- [ ] 管理后台
    - [ ] 管理员登录
    - [ ] 密钥管理
    - [ ] 书籍推荐管理
    - [ ] 横幅管理
    - [ ] 评论审核
    
- [x] API
    - [x] 用户登录 - /bookchat/api/v1/user/login
    - [x] 用户注册 - /bookchat/api/v1/user/register
    - [x] 找回密码 - /bookchat/api/v1/user/find-password
    - [x] 修改密码 - /bookchat/api/v1/user/change-password
    - [x] 用户信息 - /bookchat/api/v1/user/info
    - [x] 用户收藏 - /bookchat/api/v1/user/star
    - [x] 用户分享的书籍 - /bookchat/api/v1/user/release-book
    - [x] 用户粉丝 - /bookchat/api/v1/user/fans
    - [x] 用户关注 - /bookchat/api/v1/user/follow
    - [x] 书籍搜索 - /bookchat/api/v1/book/search
    - [x] 书籍分类 - /bookchat/api/v1/book/categories
    - [x] 书籍信息 - /bookchat/api/v1/book/info
    - [x] 书籍内容 - /bookchat/api/v1/book/read
    - [x] 书籍目录 - /bookchat/api/v1/book/menu
    - [x] 书籍点评 - /bookchat/api/v1/book/comment
    - [x] 书籍列表 - /bookchat/api/v1/book/lists
    - [x] 阅读进度 - /bookchat/api/v1/book/process
    - [x] 重置阅读进度 - /bookchat/api/v1/book/reset-process
    - [x] 书籍下载 - /bookchat/api/v1/book/download
    - [x] 添加/删除书签 - /bookchat/api/v1/book/bookmark
    - [x] 首页横幅
    
**后期改造：微信小程序404 页面不允许回退，使用redirect进行跳转。在404页面，增加一个返回首页的按钮**
    
    
## 小程序页面实现
> Promise.all() 改造

- [ ] 增加api cdn 模式的支持。如 /api/v1/lists/base64(page=1&size=2&sxx).json

- [x] 首页
- [ ] 书籍介绍页面
    - [x] 书籍信息获取
    - [x] 相关书籍获取
    - [ ] 评论获取
    - [ ] 书籍点评
    - [ ] 书籍下载？
    - [ ] 书籍收藏