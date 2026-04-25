package results

import "expire-share/internal/domain/entities"

type GetAllUsers struct {
	Users []entities.User
	Total int
}
