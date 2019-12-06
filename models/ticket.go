package models

// 书票数
// 可设定书票有效期
type Ticket struct {
	Id        int
	Uid       int
	Status    int8   // 0，未使用；1，已使用；-1，已过期
	Message   string // 书票说明
	CreatedAt int
}
