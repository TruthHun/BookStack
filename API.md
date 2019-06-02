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