# BookStack API

BookStack 配套微信小程序 BookChat API。

## TODO

- [ ] 评论管理
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

## 功能


- [ ] 管理后台
    - [ ] 管理员登录
    - [ ] 密钥管理
    - [ ] 书籍推荐管理
    - [ ] 横幅管理
    - [ ] 评论审核
    
- [ ] API
    - [ ] 关于我们 - /bookchat/api/v1/about-us
    - [ ] 用户登录 - /bookchat/api/v1/user/login
    - [ ] 用户注册 - /bookchat/api/v1/user/register
    - [ ] 找回密码 - /bookchat/api/v1/user/find-password
    - [ ] 修改密码 - /bookchat/api/v1/user/change-password
    - [ ] 用户信息 - /bookchat/api/v1/user/info
    - [ ] 用户收藏 - /bookchat/api/v1/user/star
    - [ ] 用户分享的书籍 - /bookchat/api/v1/user/release-book
    - [ ] 用户粉丝 - /bookchat/api/v1/user/fans
    - [ ] 用户关注 - /bookchat/api/v1/user/follow
    - [ ] 书籍搜索 - /bookchat/api/v1/book/search
    - [ ] 书籍分类 - /bookchat/api/v1/book/categories
    - [ ] 书籍信息 - /bookchat/api/v1/book/info
    - [ ] 书籍内容 - /bookchat/api/v1/book/read
    - [ ] 书籍目录 - /bookchat/api/v1/book/menu
    - [ ] 书籍点评 - /bookchat/api/v1/book/comment
    - [ ] 书籍列表 - /bookchat/api/v1/book/lists
    - [ ] 阅读进度 - /bookchat/api/v1/book/process
    - [ ] 重置阅读进度 - /bookchat/api/v1/book/reset-process
    - [ ] 书籍下载 - /bookchat/api/v1/book/download
    - [ ] 添加/删除书签 - /bookchat/api/v1/book/bookmark
    - [ ] 首页横幅