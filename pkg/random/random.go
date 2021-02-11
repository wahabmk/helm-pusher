package random

import (
	"math/rand"
	"time"

	"github.com/oklog/ulid/v2"
)

type Entropy struct {
	*rand.Rand
	*ulid.MonotonicEntropy
}

// New returns a new Rand that uses random values from src to generate other random values.
// NOTE: Not safe for concurrent use by multiple goroutines.
func New(seed int64) *Entropy {
	entropy := rand.New(rand.NewSource(time.Now().UnixNano() * seed))
	return &Entropy{
		Rand:             entropy,
		MonotonicEntropy: ulid.Monotonic(entropy, 0),
	}
}

func (r *Entropy) String() (string, error) {
	u, err := ulid.New(ulid.Timestamp(time.Now()), r.MonotonicEntropy)
	if err != nil {
		return "", err
	}

	return u.String(), nil
}

func Int63n(min, max int64) int64 {
	return rand.Int63n(max-min) + min
}
