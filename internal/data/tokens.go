package data

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"time"

	"github.com/ggetzie/badwords_be/internal/validator"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	ScopeAuthentication = "authentication"
	ScopeInvitation     = "invitation"
	ScopeActivation     = "activation"
)

type Token struct {
	Plaintext string    `json:"token"`
	Hash      []byte    `json:"-"`
	UserID    int       `json:"-"`
	Expiry    time.Time `json:"expiry"`
	Scope     string    `json:"-"`
}

type TokenModel struct {
	DB *pgxpool.Pool
}

func generateToken(userID int, ttl time.Duration, scope string) (*Token, error) {
	token := &Token{
		UserID: userID,
		Expiry: time.Now().Add(ttl),
		Scope:  scope,
	}

	randomBytes := make([]byte, 16)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return nil, err
	}

	token.Plaintext = base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(randomBytes)
	hash := sha256.Sum256([]byte(token.Plaintext))
	token.Hash = hash[:]
	return token, nil
}

func ValidateTokenPlaintext(v *validator.Validator, input string) {
	v.Check(input != "", "token", "must be provided")
	v.Check(len(input) == 26, "token", "must be 26 bytes long")
}

func (m TokenModel) New(userID int, ttl time.Duration, scope string) (*Token, error) {

	token, err := generateToken(userID, ttl, scope)
	if err != nil {
		return nil, err
	}
	err = m.Insert(token)
	if err != nil {
		return nil, err
	}
	return token, nil
}

func (m TokenModel) Insert(token *Token) error {
	query := `
		INSERT INTO tokens (hash, user_id, expiry, scope)
		VALUES ($1, $2, $3, $4)`
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	args := []any{token.Hash, token.UserID, token.Expiry, token.Scope}
	_, err := m.DB.Exec(ctx, query, args...)
	return err
}

func (m TokenModel) DeleteAllForUser(scope string, userID int) error {
	query := `
		DELETE FROM tokens
		WHERE scope = $1 AND user_id = $2`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err := m.DB.Exec(ctx, query, scope, userID)
	return err
}

func (m TokenModel) GetForText(plainText, scope string) (*Token, error) {
	hash := sha256.Sum256([]byte(plainText))
	query := `
		SELECT hash, user_id, expiry, scope
		FROM tokens
		WHERE hash = $1 AND scope = $2 AND expiry > now()`
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	row := m.DB.QueryRow(ctx, query, hash, scope)
	token := &Token{}
	err := row.Scan(&token.Hash, &token.UserID, &token.Expiry, &token.Scope)
	return token, err
}

func (m TokenModel) DeleteForText(plainText string) error {
	hash := sha256.Sum256([]byte(plainText))
	query := `
		DELETE FROM tokens
		WHERE hash = $1`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err := m.DB.Exec(ctx, query, hash[:])
	return err
}
