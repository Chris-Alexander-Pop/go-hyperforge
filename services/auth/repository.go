package main

import (
	"context"

	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// User represents the standard database model for authentication credentials.
type User struct {
	ID           string `gorm:"primaryKey"`
	Username     string `gorm:"uniqueIndex;not null"`
	Email        string `gorm:"uniqueIndex"`
	PasswordHash string `gorm:"not null"`
	Role         string `gorm:"not null;default:'user'"`
}

// Repository defines interactions with the persistent layer.
type Repository interface {
	CreateUser(ctx context.Context, username, email, password, role string) (*User, error)
	GetUserByUsername(ctx context.Context, username string) (*User, error)
	ValidatePassword(hash, password string) bool
}

// DBRepository implements Repository using Gorm.
type DBRepository struct {
	db *gorm.DB
}

func NewDBRepository(db *gorm.DB) *DBRepository {
	return &DBRepository{db: db}
}

func (r *DBRepository) CreateUser(ctx context.Context, username, email, password, role string) (*User, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	if err != nil {
		return nil, errors.Internal("failed to hash password", err)
	}

	u := &User{
		ID:           username + "_id", // Simple ID generation for now
		Username:     username,
		Email:        email,
		PasswordHash: string(bytes),
		Role:         role,
	}

	if err := r.db.WithContext(ctx).Create(u).Error; err != nil {
		return nil, errors.Wrap(err, "failed to create user")
	}

	return u, nil
}

func (r *DBRepository) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	var u User
	if err := r.db.WithContext(ctx).Where("username = ?", username).First(&u).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) || err == gorm.ErrRecordNotFound {
			return nil, errors.NotFound("user not found", err)
		}
		return nil, errors.Wrap(err, "database error searching for user")
	}
	return &u, nil
}

func (r *DBRepository) ValidatePassword(hash, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
