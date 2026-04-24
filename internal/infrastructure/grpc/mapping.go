package grpc

import (
	"context"
	"errors"
	"expire-share/internal/domain/entities"
	domainErrors "expire-share/internal/domain/entities/errors"
	"fmt"

	authv1 "github.com/mkaascs/AuthProto/gen/go/auth"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func pbRolesToDomain(roles []string) []entities.UserRole {
	result := make([]entities.UserRole, 0, len(roles))
	for _, role := range roles {
		result = append(result, entities.UserRole(role))
	}

	return result
}

func pbUserToDomain(user *authv1.UserInfo) entities.User {
	return entities.User{
		ID:        user.UserId,
		Login:     user.Login,
		Roles:     pbRolesToDomain(user.Roles),
		IsAdmin:   user.IsAdmin,
		CreatedAt: user.CreatedAt.AsTime(),
	}
}

func mapGrpcError(err error) error {
	st, ok := status.FromError(err)
	if !ok {
		return errors.New("failed to get grpc status from error")
	}

	if st.Code() == codes.Unauthenticated {
		return domainErrors.ErrInvalidCredentials
	}

	if st.Code() == codes.AlreadyExists {
		return domainErrors.ErrUserAlreadyExists
	}

	if st.Code() == codes.InvalidArgument {
		return fmt.Errorf("%w: %s", domainErrors.ErrInvalidArgument, st.Message())
	}

	if st.Code() == codes.Canceled {
		return context.Canceled
	}

	if st.Code() == codes.DeadlineExceeded {
		return context.DeadlineExceeded
	}

	return st.Err()
}
