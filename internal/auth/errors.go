package auth

import (
	"errors"
)

var (
	ErrEmailExists          = errors.New("email already exists")
	ErrInvalidEmail         = errors.New("invalid email address")
	ErrWeakPassword         = errors.New("weak password")
	ErrInternal             = errors.New("internal error")
	ErrHashingPassword      = errors.New("problem occurred while hashing the password")
	ErrToCreateUser         = errors.New("failed to create user")
	ErrAccessTokenGenerate  = errors.New("failed to generate access token")
	ErrRefreshTokenGenerate = errors.New("failed to generate refresh token")
	ErrRefreshTokenStore    = errors.New("failed to create refresh token entry in database")
	ErrWrongPassword        = errors.New("the password is incorrect")
	ErrInvalidRefreshToken  = errors.New("invalid or expired refresh token")
	ErrRefreshTokenReused   = errors.New("refresh token reuse detected; family revoked")
	ErrTokenRotation        = errors.New("failed to rotate refresh token")
	ErrUserNotFound         = errors.New("user not found")
)

type WeakPasswordError struct{ Reason string }

func (e *WeakPasswordError) Error() string { return e.Reason }

func wrapWeakPassword(err error) error {
	return &WeakPasswordError{Reason: err.Error()}
}
