package generator_test

import (
	"testing"

	"github.com/wahabmk/helm-pusher/pkg/generator"
)

func TestUniqueEnough(t *testing.T) {
	// Generate, on average, 100 versions per chart.
	g := generator.New(1000)
	seen := make(map[string]int)
	for i := 0; i < 100000; i++ {
		c := g.ChartName() + "-" + g.Semver()
		seen[c]++
	}

	repeats, max := 0, 1
	var maxc string
	for c, n := range seen {
		if n > 1 {
			t.Logf("value %s was generated %d times", c, n)
			repeats++
			if n > max {
				max = n
				maxc = c
			}
		}
	}

	t.Logf("%d values were generated more than once", repeats)
	t.Logf("The value %s was generated the most times (%d)", maxc, max)
	if repeats > 10 || max > 2 {
		t.Error("Too many repeats")
	}
}
