package data

import (
	"context"
	"crypto/sha256"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/ggetzie/badwords_be/internal/validator"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID          int64     `json:"id"`
	Email       string    `json:"email"`
	Password    password  `json:"-"`
	CreatedAt   time.Time `json:"created_at"`
	Activated   bool      `json:"activated"`
	Version     int       `json:"-"`
	FullName    string    `json:"full_name"`
	DisplayName string    `json:"display_name"`
}

var AnonymousUser = &User{}

func (u *User) IsAnonymous() bool {
	return u == AnonymousUser
}

type password struct {
	plaintext *string
	hash      []byte
}

func (p *password) Set(plaintext string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(plaintext), 12)
	if err != nil {
		return err
	}
	p.plaintext = &plaintext
	p.hash = hash
	return nil
}

func (p *password) Matches(plaintext string) (bool, error) {
	err := bcrypt.CompareHashAndPassword(p.hash, []byte(plaintext))
	if err != nil {
		switch {
		case errors.Is(err, bcrypt.ErrMismatchedHashAndPassword):
			return false, nil
		default:
			return false, err
		}
	}
	return true, nil
}

func ValidateEmail(v *validator.Validator, email string) {
	v.Check(email != "", "email", "must be provided")
	v.Check(validator.Matches(email, validator.EmailRX), "email", "must be a valid email address")
}

func ValidatePasswordPlaintext(v *validator.Validator, password string) {
	v.Check(password != "", "password", "must be provided")
	v.Check(len(password) >= 8, "password", "must be at least 8 bytes long")
	v.Check(len(password) <= 72, "password", "must be no more than 72 bytes long")
}

func ValidateUser(v *validator.Validator, user *User) {
	ValidateEmail(v, user.Email)
	if user.Password.plaintext != nil {
		ValidatePasswordPlaintext(v, *user.Password.plaintext)
	}
	v.Check(user.FullName != "", "full_name", "must be provided")
	v.Check(user.DisplayName != "", "display_name", "must be provided")
	// check FullName is less than 200 runes
	v.Check(utf8.RuneCountInString(user.FullName) <= 200, "full_name", "must not be more than 200 characters")
	// check DisplayName is less than 200 runes
	v.Check(utf8.RuneCountInString(user.DisplayName) <= 200, "display_name", "must not be more than 200 characters")
}

var (
	ErrDuplicateEmail       = errors.New("duplicate email")
	ErrDuplicateDisplayName = errors.New("duplicate display name")
)

type UserModel struct {
	DB *pgxpool.Pool
}

func (m UserModel) Insert(user *User) error {
	query := `
		INSERT INTO users (email, password_hash, activated, full_name, display_name)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, version`
	args := []any{
		strings.Trim(user.Email, " "),
		user.Password.hash,
		user.Activated,
		strings.Trim(user.FullName, " "),
		strings.Trim(user.DisplayName, " "),
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err := m.DB.QueryRow(ctx, query, args...).Scan(&user.ID, &user.CreatedAt, &user.Version)
	if err != nil {
		switch {
		case strings.Contains(err.Error(), "users_email_key"):
			return ErrDuplicateEmail
		case strings.Contains(err.Error(), "unique_display_name"):
			return ErrDuplicateDisplayName
		default:
			return err
		}
	}
	return nil
}

func (m UserModel) Update(user *User) error {
	query := `
		UPDATE users
		SET full_name = $1, display_name = $2, email = $3, password_hash = $4, activated = $5, version = version + 1
		WHERE id = $6 AND version = $7
		RETURNING version`
	args := []any{
		strings.Trim(user.FullName, " "),
		strings.Trim(user.DisplayName, " "),
		strings.Trim(user.Email, " "),
		user.Password.hash,
		user.Activated,
		user.ID,
		user.Version,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err := m.DB.QueryRow(ctx, query, args...).Scan(&user.Version)
	if err != nil {
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			return ErrEditConflict
		case strings.Contains(err.Error(), "users_email_key"):
			return ErrDuplicateEmail
		case strings.Contains(err.Error(), "unique_display_name"):
			return ErrDuplicateDisplayName
		default:
			return err
		}
	}
	return nil
}

func (m UserModel) GetByEmail(email string) (*User, error) {
	query := `
		SELECT id, created_at, full_name, display_name, email, password_hash, activated, version
		FROM users
		WHERE email = $1`

	var user User
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	row := m.DB.QueryRow(ctx, query, email)
	err := row.Scan(
		&user.ID, &user.CreatedAt, &user.FullName, &user.DisplayName,
		&user.Email, &user.Password.hash, &user.Activated, &user.Version)
	if err != nil {
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}
	return &user, nil
}

func (m UserModel) GetForToken(tokenScope, tokenPlaintext string) (*User, error) {
	tokenHash := sha256.Sum256([]byte(tokenPlaintext))

	query := `
	SELECT users.id, users.created_at, users.full_name, users.display_name, users.email, users.password_hash, users.activated, users.version
	FROM users
	INNER JOIN tokens
	ON users.id = tokens.user_id
	WHERE tokens.hash = $1
	AND tokens.scope = $2
	AND tokens.expiry > $3`

	args := []any{tokenHash[:], tokenScope, time.Now()}

	var user User
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := m.DB.QueryRow(ctx, query, args...).Scan(
		&user.ID,
		&user.CreatedAt,
		&user.FullName,
		&user.DisplayName,
		&user.Email,
		&user.Password.hash,
		&user.Activated,
		&user.Version,
	)

	if err != nil {
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}
	return &user, nil
}

func (m UserModel) GetByID(id int64) (*User, error) {
	query := `
		SELECT id, created_at, full_name, display_name, email, password_hash, activated, version
		FROM users
		WHERE id = $1`

	var user User
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	row := m.DB.QueryRow(ctx, query, id)
	err := row.Scan(
		&user.ID, &user.CreatedAt, &user.FullName, &user.DisplayName,
		&user.Email, &user.Password.hash, &user.Activated, &user.Version)
	if err != nil {
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}
	return &user, nil
}
