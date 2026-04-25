package entities

import "time"

const (
	RoleUser  = "user"
	RoleVip   = "vip"
	RoleAdmin = "admin"
)

type UserRole string

type User struct {
	ID        int64
	Login     string
	Email     string
	Roles     []UserRole
	IsAdmin   bool
	CreatedAt time.Time
}
