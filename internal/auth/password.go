package auth

import (
	"context"
	"errors"

	"golang.org/x/crypto/bcrypt"
)

type BCryptHasher struct {
	cost int
}

func NewBCryptHasher(cost int) *BCryptHasher {
	if cost < bcrypt.MinCost {
		cost = bcrypt.DefaultCost
	}
	return &BCryptHasher{cost: cost}
}

func (h *BCryptHasher) Hash(_ context.Context, password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), h.cost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func (h *BCryptHasher) Verify(_ context.Context, password, hash string) error {
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return errors.New("invalid password")
	}
	return nil
}
