package utils

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
)

const (
	pnrCharset = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

	PNRLength = 6
)

func GeneratePNR() (string, error) {
	var sb strings.Builder
	sb.Grow(PNRLength)

	charsetLen := big.NewInt(int64(len(pnrCharset)))

	for i := 0; i < PNRLength; i++ {
		index, err := rand.Int(rand.Reader, charsetLen)
		if err != nil {
			return "", fmt.Errorf("crypto/rand failed: %w", err)
		}
		sb.WriteByte(pnrCharset[index.Int64()])
	}

	return sb.String(), nil
}

func MustGeneratePNR() string {
	pnr, err := GeneratePNR()
	if err != nil {
		panic(err)
	}
	return pnr
}
