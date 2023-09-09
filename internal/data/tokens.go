package data

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base32"
	"fmt"
	"time"

	"github.com/Crocmagnon/greenlight/internal/validator"
)

// Scopes are used to limit the use cases of tokens.
const (
	// ScopeActivation is used to activate a user account.
	ScopeActivation = "activation"
	// ScopeAuthentication is used to authenticate users.
	ScopeAuthentication = "authentication"
)

// A Token is used to activate a User account.
type Token struct {
	Plaintext string    `json:"token"`
	Hash      []byte    `json:"-"`
	UserID    int64     `json:"-"`
	Expiry    time.Time `json:"expiry"`
	Scope     string    `json:"-"`
}

func generateToken(userID int64, ttl time.Duration, scope string) (*Token, error) {
	token := &Token{
		UserID: userID,
		Expiry: time.Now().Add(ttl),
		Scope:  scope,
	}

	randomBytes := make([]byte, 16) //nolint:gomnd

	_, err := rand.Read(randomBytes)
	if err != nil {
		return nil, fmt.Errorf("generating bytes: %w", err)
	}

	token.Plaintext = base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(randomBytes)
	hash := sha256.Sum256([]byte(token.Plaintext))
	token.Hash = hash[:]

	return token, nil
}

// ValidateTokenPlaintext validates a token.
// The passed validator will contain all detected errors.
// The caller is expected to call [validator.Validator.Valid]
// after this method.
//
//nolint:gomnd
func ValidateTokenPlaintext(v *validator.Validator, tokenPlaintext string) {
	v.Check(tokenPlaintext != "", "token", "must be provided")
	v.Check(len(tokenPlaintext) == 26, "token", "must be 26 bytes long")
}

// TokenModel implements methods to query the database.
type TokenModel struct {
	DB *sql.DB
}

// New creates a token and stores it in the DB.
func (m TokenModel) New(userID int64, ttl time.Duration, scope string) (*Token, error) {
	token, err := generateToken(userID, ttl, scope)
	if err != nil {
		return nil, err
	}

	err = m.Insert(token)

	return token, err
}

// Insert inserts a token in the DB.
func (m TokenModel) Insert(token *Token) error {
	query := `
        INSERT INTO tokens (hash, user_id, expiry, scope) 
        VALUES ($1, $2, $3, $4)`

	args := []any{token.Hash, token.UserID, token.Expiry, token.Scope}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if _, err := m.DB.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("inserting token: %w", err)
	}

	return nil
}

// DeleteAllForUser deletes all tokens for a specific user and scope.
func (m TokenModel) DeleteAllForUser(scope string, userID int64) error {
	query := `
        DELETE FROM tokens 
        WHERE scope = $1 AND user_id = $2`

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if _, err := m.DB.ExecContext(ctx, query, scope, userID); err != nil {
		return fmt.Errorf("deleting all tokens for user: %w", err)
	}

	return nil
}
