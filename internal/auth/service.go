package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"golang.org/x/crypto/bcrypt"

	"taskflow/internal/models"
)

// Claims holds the parsed contents of a validated JWT.
type Claims struct {
	UserID    uuid.UUID
	ExpiresAt time.Time
}

// UserRepository defines the persistence operations the auth service needs.
type UserRepository interface {
	Insert(ctx context.Context, user *models.User) error
	FindByEmail(ctx context.Context, email string) (*models.User, error)
}

// AuthService defines the public interface for authentication operations.
type AuthService interface {
	Register(ctx context.Context, req models.RegisterRequest) (*models.User, error)
	Login(ctx context.Context, req models.LoginRequest) (string, error)
	ValidateToken(tokenString string) (*Claims, error)
}

// authService is the concrete implementation of AuthService.
type authService struct {
	repo           UserRepository
	jwtSecret      []byte
	jwtExpiryHours int
}

// NewAuthService constructs an AuthService backed by the given UserRepository.
func NewAuthService(repo UserRepository, jwtSecret string, jwtExpiryHours int) AuthService {
	return &authService{
		repo:           repo,
		jwtSecret:      []byte(jwtSecret),
		jwtExpiryHours: jwtExpiryHours,
	}
}

// Register validates the request, checks for email uniqueness, hashes the
// password, and inserts the new user into the repository.
func (s *authService) Register(ctx context.Context, req models.RegisterRequest) (*models.User, error) {
	if req.Email == "" {
		return nil, fmt.Errorf("%w: email is required", models.ErrValidation)
	}
	if len(req.Password) < 8 {
		return nil, fmt.Errorf("%w: password must be at least 8 characters", models.ErrValidation)
	}

	existing, err := s.repo.FindByEmail(ctx, req.Email)
	if err != nil && !errors.Is(err, models.ErrNotFound) {
		return nil, fmt.Errorf("checking existing email: %w", err)
	}
	if existing != nil {
		return nil, fmt.Errorf("%w: email already registered", models.ErrConflict)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), 12)
	if err != nil {
		return nil, fmt.Errorf("hashing password: %w", err)
	}

	user := &models.User{
		ID:           uuid.New(),
		Email:        req.Email,
		PasswordHash: string(hash),
		CreatedAt:    time.Now().UTC(),
	}

	if err := s.repo.Insert(ctx, user); err != nil {
		return nil, fmt.Errorf("inserting user: %w", err)
	}

	return user, nil
}

// Login validates credentials and returns a signed JWT on success.
func (s *authService) Login(ctx context.Context, req models.LoginRequest) (string, error) {
	user, err := s.repo.FindByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			return "", models.ErrUnauthorized
		}
		return "", fmt.Errorf("finding user: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return "", models.ErrUnauthorized
	}

	now := time.Now().UTC()
	expiresAt := now.Add(time.Duration(s.jwtExpiryHours) * time.Hour)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": user.ID.String(),
		"exp": expiresAt.Unix(),
	})

	signed, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return "", fmt.Errorf("signing token: %w", err)
	}

	return signed, nil
}

// ValidateToken parses and verifies a JWT string, returning the extracted Claims.
func (s *authService) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.jwtSecret, nil
	}, jwt.WithValidMethods([]string{"HS256"}))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", models.ErrUnauthorized, err)
	}

	mapClaims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("%w: invalid token claims", models.ErrUnauthorized)
	}

	subStr, err := mapClaims.GetSubject()
	if err != nil || subStr == "" {
		return nil, fmt.Errorf("%w: missing subject claim", models.ErrUnauthorized)
	}

	userID, err := uuid.Parse(subStr)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid subject UUID", models.ErrUnauthorized)
	}

	expTime, err := mapClaims.GetExpirationTime()
	if err != nil || expTime == nil {
		return nil, fmt.Errorf("%w: missing expiry claim", models.ErrUnauthorized)
	}

	return &Claims{
		UserID:    userID,
		ExpiresAt: expTime.Time,
	}, nil
}

// ─── PostgreSQL UserRepository ─────────────────────────────────────────────────

// postgresUserRepository is a sqlx-backed implementation of UserRepository.
type postgresUserRepository struct {
	db *sqlx.DB
}

// NewPostgresUserRepository constructs a UserRepository backed by PostgreSQL.
func NewPostgresUserRepository(db *sqlx.DB) UserRepository {
	return &postgresUserRepository{db: db}
}

// Insert persists a new user row.
func (r *postgresUserRepository) Insert(ctx context.Context, user *models.User) error {
	const q = `INSERT INTO users (id, email, password_hash, created_at)
	           VALUES (:id, :email, :password_hash, :created_at)`
	if _, err := r.db.NamedExecContext(ctx, q, user); err != nil {
		return fmt.Errorf("insert user: %w", err)
	}
	return nil
}

// FindByEmail retrieves a user by email address, returning models.ErrNotFound
// when no matching row exists.
func (r *postgresUserRepository) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	const q = `SELECT * FROM users WHERE email = $1`
	var user models.User
	if err := r.db.GetContext(ctx, &user, q, email); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, models.ErrNotFound
		}
		return nil, fmt.Errorf("find user by email: %w", err)
	}
	return &user, nil
}
