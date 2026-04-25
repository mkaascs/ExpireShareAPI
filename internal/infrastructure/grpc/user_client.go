package grpc

import (
	"context"
	"expire-share/internal/domain/dto/users/commands"
	"expire-share/internal/domain/dto/users/results"
	"expire-share/internal/domain/entities"
	domainErrors "expire-share/internal/domain/entities/errors"
	authv1 "github.com/mkaascs/AuthProto/gen/go/auth"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type UserClient struct {
	userClient authv1.UserClient
}

func NewUserClient(grpcConn *grpc.ClientConn) *UserClient {
	return &UserClient{
		userClient: authv1.NewUserClient(grpcConn),
	}
}

func (us *UserClient) GetUserByID(ctx context.Context, userID int64) (*entities.User, error) {
	result, err := us.userClient.GetUser(ctx, &authv1.GetUserRequest{
		UserId: userID,
	})

	if err != nil {
		return nil, mapGrpcError(err)
	}

	user := pbUserToDomain(result.User)
	return &user, nil
}

func (us *UserClient) GetAllUsers(ctx context.Context, command commands.GetAllUsers) (*results.GetAllUsers, error) {
	result, err := us.userClient.GetUsers(ctx, &authv1.GetUsersRequest{
		Role:  (*string)(command.Role),
		Page:  int32(command.Page),
		Limit: int32(command.Limit),
	})

	if err != nil {
		return nil, mapGrpcError(err)
	}

	users := make([]entities.User, 0, len(result.Users))
	for _, user := range result.Users {
		users = append(users, pbUserToDomain(user))
	}

	return &results.GetAllUsers{
		Total: int(result.Total),
		Users: users,
	}, nil
}

func (us *UserClient) AssignRole(ctx context.Context, command commands.AssignRole) error {
	_, err := us.userClient.AssignRole(ctx, &authv1.AssignRoleRequest{
		UserId: command.UserID,
		Role:   string(command.Role),
	})

	return mapGrpcError(err)
}

func (us *UserClient) RevokeRole(ctx context.Context, command commands.RevokeRole) error {
	_, err := us.userClient.RevokeRole(ctx, &authv1.RevokeRoleRequest{
		UserId: command.UserID,
		Role:   string(command.Role),
	})

	if status.Code(err) == codes.NotFound {
		return domainErrors.ErrRoleNotExist
	}

	return mapGrpcError(err)
}
