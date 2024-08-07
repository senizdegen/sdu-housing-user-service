package user

import "context"

type Storage interface {
	FindOne(ctx context.Context, uuid string) (User, error)
	FindByPhoneNumber(ctx context.Context, phoneNumber string) (User, error)
	Create(ctx context.Context, user User) (string, error)
}
