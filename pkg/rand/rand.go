package rand

import (
	cryptorand "crypto/rand"
	"encoding/hex"
	"fmt"
	"math/rand"
	"time"
)

func String(strLen int64) (string, error) {
	b := make([]byte, strLen/2)
	n, err := cryptorand.Read(b)
	if int64(n) != strLen/2 {
		return "", fmt.Errorf("only generated %d random bytes", n)
	}
	if err != nil {
		return "", err
	}
	choice := hex.EncodeToString(b)
	return choice, nil
}

func Int64(min, max int64) int64 {
	rand.Seed(time.Now().UnixNano())
	return rand.Int63n(max-min) + min
}
