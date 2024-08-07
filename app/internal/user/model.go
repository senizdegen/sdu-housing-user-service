package user

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

type User struct {
	UUID        string `json:"uuid" bson:"_id,omitempty"`
	PhoneNumber string `json:"phone_number" bson:"phone_number,omitempty"`
	Password    string `json:"-" bson:"password,omitempty"`
	Role        string `json:"role" bson:"role"`
	JWTToken    string `json:"jwt" bson:"-"`
}

type CreateUserDTO struct {
	PhoneNumber    string `json:"phone_number"`
	Password       string `json:"password"`
	RepeatPassword string `json:"repeat_password"`
}

func NewUser(dto CreateUserDTO) User {
	return User{
		PhoneNumber: dto.PhoneNumber,
		Password:    dto.Password,
	}
}

func (u *User) CheckPassword(password string) error {
	err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte([]byte(password)))
	if err != nil {
		return fmt.Errorf("password does not match")
	}
	return nil
}

func (u *User) GeneratePasswordHash() error {
	pwd, err := generatePasswordHash(u.Password)
	if err != nil {
		return err
	}
	u.Password = pwd
	return nil
}

func generatePasswordHash(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password due to error: %w", err)
	}
	return string(hash), nil
}
