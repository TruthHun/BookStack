package routers

import (
	"github.com/TruthHun/BookStack/controllers"
	"github.com/astaxie/beego"
)

func webRouter() {
	beego.Router("/", &controllers.CateController{}, "get:Index")
	beego.Router("/app", &controllers.StaticController{}, "get:APP")
	beego.Router("/cate", &controllers.CateController{}, "get:List")
	//beego.Router("/", &controllers.HomeController{}, "*:Index")
	beego.Router("/explore", &controllers.HomeController{}, "*:Index")

	beego.Router("/submit", &controllers.SubmitController{}, "get:Index")
	beego.Router("/submit", &controllers.SubmitController{}, "post:Post")

	beego.Router("/login", &controllers.AccountController{}, "*:Login")
	beego.Router("/login/:oauth", &controllers.AccountController{}, "*:Oauth")
	beego.Router("/logout", &controllers.AccountController{}, "*:Logout")
	beego.Router("/bind", &controllers.AccountController{}, "post:Bind")
	beego.Router("/note", &controllers.AccountController{}, "get,post:Note")
	beego.Router("/find_password", &controllers.AccountController{}, "*:FindPassword")
	beego.Router("/valid_email", &controllers.AccountController{}, "post:ValidEmail")
	//beego.Router("/captcha", &controllers.AccountController{}, "*:Captcha")

	beego.Router("/manager", &controllers.ManagerController{}, "*:Index")
	beego.Router("/manager/users", &controllers.ManagerController{}, "*:Users")
	beego.Router("/manager/users/edit/:id", &controllers.ManagerController{}, "*:EditMember")
	beego.Router("/manager/member/create", &controllers.ManagerController{}, "post:CreateMember")
	beego.Router("/manager/member/delete", &controllers.ManagerController{}, "post:DeleteMember")
	beego.Router("/manager/member/update-member-status", &controllers.ManagerController{}, "post:UpdateMemberStatus")
	beego.Router("/manager/member/update-member-no-rank", &controllers.ManagerController{}, "post:UpdateMemberNoRank")
	beego.Router("/manager/member/change-member-role", &controllers.ManagerController{}, "post:ChangeMemberRole")
	beego.Router("/manager/books", &controllers.ManagerController{}, "*:Books")
	beego.Router("/manager/books/edit/:key", &controllers.ManagerController{}, "*:EditBook")
	beego.Router("/manager/books/delete", &controllers.ManagerController{}, "*:DeleteBook")
	beego.Router("/manager/comments", &controllers.ManagerController{}, "*:Comments")
	beego.Router("/manager/comments/delete", &controllers.ManagerController{}, "*:DeleteComment")
	beego.Router("/manager/comments/clear", &controllers.ManagerController{}, "*:ClearComments")
	beego.Router("/manager/comments/set", &controllers.ManagerController{}, "*:SetCommentStatus")
	beego.Router("/manager/books/token", &controllers.ManagerController{}, "post:CreateToken")
	beego.Router("/manager/setting", &controllers.ManagerController{}, "*:Setting")
	beego.Router("/manager/books/transfer", &controllers.ManagerController{}, "post:Transfer")
	beego.Router("/manager/books/sort", &controllers.ManagerController{}, "get:UpdateBookSort")
	beego.Router("/manager/books/open", &controllers.ManagerController{}, "post:PrivatelyOwned")
	beego.Router("/manager/attach/list", &controllers.ManagerController{}, "*:AttachList")
	beego.Router("/manager/attach/detailed/:id", &controllers.ManagerController{}, "*:AttachDetailed")
	beego.Router("/manager/attach/delete", &controllers.ManagerController{}, "*:AttachDelete")
	beego.Router("/manager/seo", &controllers.ManagerController{}, "post,get:Seo")
	beego.Router("/manager/ads", &controllers.ManagerController{}, "post,get:Ads")
	beego.Router("/manager/update-ads", &controllers.ManagerController{}, "post,get:UpdateAds")
	beego.Router("/manager/del-ads", &controllers.ManagerController{}, "get:DelAds")
	beego.Router("/manager/category", &controllers.ManagerController{}, "post,get:Category")
	beego.Router("/manager/update-cate", &controllers.ManagerController{}, "get:UpdateCate")
	beego.Router("/manager/del-cate", &controllers.ManagerController{}, "get:DelCate")
	beego.Router("/manager/icon-cate", &controllers.ManagerController{}, "post:UpdateCateIcon")
	beego.Router("/manager/sitemap", &controllers.ManagerController{}, "get:Sitemap")       //更新站点地图
	beego.Router("/manager/friendlink", &controllers.ManagerController{}, "get:FriendLink") //友链管理
	beego.Router("/manager/add_friendlink", &controllers.ManagerController{}, "post:AddFriendlink")
	beego.Router("/manager/update_friendlink", &controllers.ManagerController{}, "get:UpdateFriendlink")
	beego.Router("/manager/rebuild-index", &controllers.ManagerController{}, "get:RebuildAllIndex")
	beego.Router("/manager/del_friendlink", &controllers.ManagerController{}, "get:DelFriendlink")
	beego.Router("/manager/banners", &controllers.ManagerController{}, "get:Banners")
	beego.Router("/manager/banners/upload", &controllers.ManagerController{}, "post:UploadBanner")
	beego.Router("/manager/banners/delete", &controllers.ManagerController{}, "get:DeleteBanner")
	beego.Router("/manager/banners/update", &controllers.ManagerController{}, "get:UpdateBanner")
	beego.Router("/manager/submit-book", &controllers.ManagerController{}, "get:SubmitBook")
	beego.Router("/manager/submit-book/update", &controllers.ManagerController{}, "get:UpdateSubmitBook")
	beego.Router("/manager/submit-book/delete", &controllers.ManagerController{}, "get:DeleteSubmitBook")
	beego.Router("/manager/tags", &controllers.ManagerController{}, "get:Tags")
	beego.Router("/manager/add-tags", &controllers.ManagerController{}, "post:AddTags")
	beego.Router("/manager/del-tags", &controllers.ManagerController{}, "get:DelTags")

	beego.Router("/manager/versions", &controllers.ManagerController{}, "*:Version")
	beego.Router("/manager/add-versions", &controllers.ManagerController{}, "post:AddVersions")
	beego.Router("/manager/delete-version", &controllers.ManagerController{}, "get:DeleteVersion")
	beego.Router("/manager/update-version", &controllers.ManagerController{}, "*:UpdateVersion")

	beego.Router("/setting", &controllers.SettingController{}, "*:Index")
	beego.Router("/setting/password", &controllers.SettingController{}, "*:Password")
	beego.Router("/setting/upload", &controllers.SettingController{}, "*:Upload")
	beego.Router("/setting/star", &controllers.SettingController{}, "*:Star")
	beego.Router("/setting/qrcode", &controllers.SettingController{}, "*:Qrcode")

	beego.Router("/book", &controllers.BookController{}, "*:Index")
	beego.Router("/book/star/:id", &controllers.BookController{}, "*:Star")          // 收藏
	beego.Router("/book/score/:id", &controllers.BookController{}, "*:Score")        // 评分
	beego.Router("/book/comment/:id", &controllers.BookController{}, "post:Comment") // 点评
	beego.Router("/book/uploadProject", &controllers.BookController{}, "post:UploadProject")
	beego.Router("/book/downloadProject", &controllers.BookController{}, "post:DownloadProject")
	beego.Router("/book/git-pull", &controllers.BookController{}, "post:GitPull")
	beego.Router("/book/:key/dashboard", &controllers.BookController{}, "*:Dashboard")
	beego.Router("/book/:key/setting", &controllers.BookController{}, "*:Setting")
	beego.Router("/book/:key/users", &controllers.BookController{}, "*:Users")
	beego.Router("/book/:key/release", &controllers.BookController{}, "post:Release")
	beego.Router("/book/:key/generate", &controllers.BookController{}, "get,post:Generate")
	beego.Router("/book/:key/sort", &controllers.BookController{}, "post:SaveSort")
	beego.Router("/book/:key/replace", &controllers.BookController{}, "get,post:Replace")

	beego.Router("/book/create", &controllers.BookController{}, "post:Create")
	beego.Router("/book/copy", &controllers.BookController{}, "post:Copy")
	beego.Router("/book/users/create", &controllers.BookMemberController{}, "post:AddMember")
	beego.Router("/book/users/change", &controllers.BookMemberController{}, "post:ChangeRole")
	beego.Router("/book/users/delete", &controllers.BookMemberController{}, "post:RemoveMember")

	beego.Router("/book/setting/save", &controllers.BookController{}, "post:SaveBook")
	beego.Router("/book/setting/open", &controllers.BookController{}, "post:PrivatelyOwned")
	beego.Router("/book/setting/transfer", &controllers.BookController{}, "post:Transfer")
	beego.Router("/book/setting/upload", &controllers.BookController{}, "post:UploadCover")
	beego.Router("/book/setting/token", &controllers.BookController{}, "post:CreateToken")
	beego.Router("/book/setting/delete", &controllers.BookController{}, "post:Delete")

	beego.Router("/bookmark/:id", &controllers.BookmarkController{}, "get:Bookmark")
	beego.Router("/bookmark/list/:book_id", &controllers.BookmarkController{}, "get:List")
	//阅读记录
	beego.Router("/record/:book_id", &controllers.RecordController{}, "get:List")
	beego.Router("/record/:book_id/clear", &controllers.RecordController{}, "get:Clear")
	beego.Router("/record/delete/:doc_id", &controllers.RecordController{}, "get:Delete")

	beego.Router("/api/attach/remove/", &controllers.DocumentController{}, "post:RemoveAttachment")
	beego.Router("/api/:key/edit/?:id", &controllers.DocumentController{}, "*:Edit")
	beego.Router("/api/upload", &controllers.DocumentController{}, "post:Upload")
	beego.Router("/api/:key/create", &controllers.DocumentController{}, "post:Create")
	beego.Router("/api/create_multi", &controllers.DocumentController{}, "post:CreateMulti")
	beego.Router("/api/:key/delete", &controllers.DocumentController{}, "post:Delete")
	beego.Router("/api/:key/content/?:id", &controllers.DocumentController{}, "*:Content")
	beego.Router("/api/:key/compare/:id", &controllers.DocumentController{}, "*:Compare")

	beego.Router("/history/get", &controllers.DocumentController{}, "get:History")
	beego.Router("/history/delete", &controllers.DocumentController{}, "*:DeleteHistory")
	beego.Router("/history/restore", &controllers.DocumentController{}, "*:RestoreHistory")

	beego.Router("/books/:key", &controllers.DocumentController{}, "*:Index")
	beego.Router("/read/:key", &controllers.DocumentController{}, "*:ReadBook")
	beego.Router("/read/:key/:id", &controllers.DocumentController{}, "*:Read")
	beego.Router("/read/:key/search", &controllers.DocumentController{}, "post:Search")

	beego.Router("/export/:key", &controllers.DocumentController{}, "*:Export")
	beego.Router("/export2markdown", &controllers.BookController{}, "get:Export2Markdown")
	beego.Router("/qrcode/:key.png", &controllers.DocumentController{}, "get:QrCode")

	beego.Router("/attach_files/:key/:attach_id", &controllers.DocumentController{}, "get:DownloadAttachment")

	beego.Router("/comment/create", &controllers.CommentController{}, "post:Create")
	beego.Router("/comment/lists", &controllers.CommentController{}, "get:Lists")
	beego.Router("/comment/index", &controllers.CommentController{}, "*:Index")

	beego.Router("/search", &controllers.SearchController{}, "get:Search")
	beego.Router("/search/result", &controllers.SearchController{}, "get:Result")
	beego.Router("/crawl", &controllers.BaseController{}, "post:Crawl")

	//用户中心 【start】
	beego.Router("/user/:username", &controllers.UserController{}, "get:Index")
	beego.Router("/user/:username/collection", &controllers.UserController{}, "get:Collection")
	beego.Router("/user/:username/follow", &controllers.UserController{}, "get:Follow")
	beego.Router("/user/:username/fans", &controllers.UserController{}, "get:Fans")
	beego.Router("/follow/:uid", &controllers.BaseController{}, "get:SetFollow") //关注或取消关注
	beego.Router("/user/sign", &controllers.BaseController{}, "get:SignToday")   //关注或取消关注
	//用户中心 【end】

	beego.Router("/tag/:key", &controllers.LabelController{}, "get:Index")
	beego.Router("/tag", &controllers.LabelController{}, "get:List")
	beego.Router("/tags", &controllers.LabelController{}, "get:List")

	beego.Router("/rank", &controllers.RankController{}, "get:Index")

	beego.Router("/sitemap.html", &controllers.BaseController{}, "get:Sitemap")
	beego.Router("/local-render", &controllers.LocalhostController{}, "get,post:RenderMarkdown")
	beego.Router("/local-render-cover", &controllers.LocalhostController{}, "get:RenderCover")
	beego.Router("/projects/*", &controllers.StaticController{}, "get:ProjectsFile")
	beego.Router("/uploads/*", &controllers.StaticController{}, "get:Uploads")
	beego.Router("/test", &controllers.StaticController{}, "get:Test")
	beego.Router("/*", &controllers.StaticController{}, "get:StaticFile")
}
