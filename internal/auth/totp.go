package auth

import (
	"context"

	"github.com/pquerna/otp/totp"
)

type TOTPHandler struct{}

func NewTOTPHandler() *TOTPHandler {
	return &TOTPHandler{}
}

func (t *TOTPHandler) Generate(_ context.Context, account string) (*TOTPKey, error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "SecretHub",
		AccountName: account,
	})
	if err != nil {
		return nil, err
	}
	return &TOTPKey{
		Secret: key.Secret(),
		URL:    key.URL(),
	}, nil
}

func (t *TOTPHandler) Validate(_ context.Context, secret, code string) bool {
	return totp.Validate(code, secret)
}
