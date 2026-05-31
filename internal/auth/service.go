package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

type Service struct {
	repo     *UserRepository
	jtiStore *JTIStore
}

func NewService(repo *UserRepository, jtiStore *JTIStore) *Service {
	return &Service{repo: repo, jtiStore: jtiStore}
}

func (s *Service) CreateUserService(ctx context.Context, req *CreateUserRequest) (*SignupResult, error) {
	if !validateEmail(req.Email) {
		return nil, ErrInvalidEmail
	}

	if err := validatePassword(req.Password); err != nil {
		return nil, wrapWeakPassword(err)
	}

	exists, err := s.repo.CheckEmailExists(ctx, req.Email)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInternal, err)
	}
	if exists {
		return nil, ErrEmailExists
	}

	hashedPassword, err := HashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrHashingPassword, err)
	}

	rawRefresh, hashedRefresh, err := GenerateRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRefreshTokenGenerate, err)
	}

	user := &User{
		Email:        req.Email,
		PasswordHash: hashedPassword,
	}
	refresh := &RefreshToken{
		TokenHash: hashedRefresh,
		ExpiresAt: time.Now().Add(RefreshTokenTTL),
	}

	createdUser, err := s.repo.CreateUserWithRefreshToken(ctx, user, refresh)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrToCreateUser, err)
	}

	accessToken, err := CreateToken(createdUser.ID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrAccessTokenGenerate, err)
	}

	return &SignupResult{
		User:         createdUser,
		AccessToken:  accessToken,
		RefreshToken: rawRefresh,
	}, nil
}

func (s *Service) LoginUserService(ctx context.Context, req *LoginRequest) (*LoginResult, error) {
	if !validateEmail(req.Email) {
		return nil, ErrInvalidEmail
	}

	user, err := s.repo.GetUserPassword(ctx, req.Email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get user: %w", err)
	}

	if err := CheckPasswordHash(req.Password, user.PasswordHash); err != nil {
		return nil, ErrWrongPassword
	}

	rawRefresh, hashedRefresh, err := GenerateRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRefreshTokenGenerate, err)
	}

	refresh := &RefreshToken{
		UserID:    user.ID,
		TokenHash: hashedRefresh,
		ExpiresAt: time.Now().Add(RefreshTokenTTL),
	}
	if err := s.repo.CreateRefreshToken(ctx, refresh); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRefreshTokenStore, err)
	}

	accessToken, err := CreateToken(user.ID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrAccessTokenGenerate, err)
	}

	return &LoginResult{
		User:         user,
		AccessToken:  accessToken,
		RefreshToken: rawRefresh,
	}, nil
}

func (s *Service) RefreshAccessToken(ctx context.Context, refreshToken string) (*Tokens, error) {
	hash := HashToken(refreshToken)

	stored, err := s.repo.FindRefreshTokenByHash(ctx, hash)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidRefreshToken
		}
		return nil, fmt.Errorf("lookup refresh token: %w", err)
	}

	// Reuse detection: a revoked token being replayed means it was either
	// already rotated (and someone is replaying the old value) or stolen.
	// Kill the whole family — every active sibling/descendant — so the
	// attacker can't keep rotating. Other families (other devices, other
	// users) are unaffected.
	if stored.RevokedAt != nil {
		_ = s.repo.RevokeFamily(ctx, stored.FamilyID)
		return nil, ErrRefreshTokenReused
	}

	if time.Now().After(stored.ExpiresAt) {
		return nil, ErrInvalidRefreshToken
	}

	user, err := s.repo.GetUserByID(ctx, stored.UserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("get user: %w", err)
	}

	rawRefresh, hashedRefresh, err := GenerateRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRefreshTokenGenerate, err)
	}

	if _, err := s.repo.RotateRefreshToken(ctx, stored, hashedRefresh); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrTokenRotation, err)
	}

	accessToken, err := CreateToken(user.ID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrAccessTokenGenerate, err)
	}

	return &Tokens{
		AccessToken:  accessToken,
		RefreshToken: rawRefresh,
	}, nil
}

func (s *Service) LogoutService(ctx context.Context, jti string, accessExp time.Time, refreshToken string) error {
	if err := s.jtiStore.Revoke(ctx, jti, time.Until(accessExp)); err != nil {
		return fmt.Errorf("revoke jti: %w", err)
	}

	hash := HashToken(refreshToken)

	stored, err := s.repo.FindRefreshTokenByHash(ctx, hash)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return fmt.Errorf("lookup refresh token: %w", err)
	}

	if stored.RevokedAt != nil {
		_ = s.repo.RevokeFamily(ctx, stored.FamilyID)
		return ErrRefreshTokenReused
	}

	return s.repo.RevokeFamily(ctx, stored.FamilyID)
}
