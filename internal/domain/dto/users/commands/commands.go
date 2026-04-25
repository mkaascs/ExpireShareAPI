package commands

import "expire-share/internal/domain/entities"

type GetAllUsers struct {
	Page  int
	Limit int
	Role  *entities.UserRole
}

type AssignRole struct {
	UserID int64
	Role   entities.UserRole
}

type RevokeRole struct {
	UserID int64
	Role   entities.UserRole
}
