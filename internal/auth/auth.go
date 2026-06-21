package auth

import "context"

type PasswordHasher interface {
	Hash(ctx context.Context, password string) (string, error)
	Verify(ctx context.Context, password, hash string) error
}

type TOTPManager interface {
	Generate(ctx context.Context, account string) (*TOTPKey, error)
	Validate(ctx context.Context, secret, code string) bool
}

type RecoveryManager interface {
	Generate(ctx context.Context) ([]string, error)
	Validate(ctx context.Context, code string) bool
	Use(ctx context.Context, code string) bool
	Hashes() []string
}

type TOTPKey struct {
	Secret  string
	URL     string
}
