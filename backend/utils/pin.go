package utils

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

func NewEventPIN() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}
