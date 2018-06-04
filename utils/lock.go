package utils

import "sync"

var (
	BooksRelease  = BooksLock{Books: make(map[int]bool)}
	BooksGenerate = BooksLock{Books: make(map[int]bool)}
)

//书籍发布锁和书籍离线文档生成锁
type BooksLock struct {
	Books map[int]bool
	Lock  sync.RWMutex
}

//查询是否存在
func (this BooksLock) Exist(bookId int) (exist bool) {
	this.Lock.RLock()
	defer this.Lock.RUnlock()
	_, exist = this.Books[bookId]
	return
}

//设置
func (this BooksLock) Set(bookId int) {
	this.Lock.RLock()
	defer this.Lock.RUnlock()
	this.Books[bookId] = true
}

//删除
func (this BooksLock) Delete(bookId int) {
	this.Lock.RLock()
	defer this.Lock.RUnlock()
	delete(this.Books, bookId)
}
