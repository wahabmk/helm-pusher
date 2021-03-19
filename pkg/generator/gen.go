package generator

import (
	"encoding/base32"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Generator generates random chart names and versions.
// It is safe for concurrent use.
type Generator struct {
	names          []string
	rng            *rand.Rand
	rngMu          sync.Mutex
	ver, pre, npre *rand.Zipf
}

func New(charts int) Generator {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Generate a short random alphanumeric prefix for chart names to make
	// collisions with other generator instances unlikely.
	prefixBytes := make([]byte, 5)
	_, _ = rng.Read(prefixBytes)
	prefix := strings.ToLower(base32.StdEncoding.EncodeToString(prefixBytes))

	names := make([]string, charts)
	for n := range names {
		c := n
		name := []rune{'a' + rune(c%26)}
		c /= 26
		for c > 0 {
			name = append(name, 'a'+rune(c%26))
			c /= 26
		}
		names[n] = prefix + "-" + string(name)
	}

	return Generator{
		rng:   rng,
		names: names,
		// The parameters are chosen arbitrarily and through experimentation
		// to hit an acceptably low rate of duplicates.
		ver:  rand.NewZipf(rng, 1.1, 5, 30),
		pre:  rand.NewZipf(rng, 1.1, 10, 99),
		npre: rand.NewZipf(rng, 1.2, 3, 6),
	}
}

func (g *Generator) intn(n int) int {
	g.rngMu.Lock()
	defer g.rngMu.Unlock()
	return g.rng.Intn(n)
}

func (g *Generator) ChartName() string {
	return g.names[g.intn(len(g.names))]
}

var preReleaseStrings = []string{
	"a", "b", "c", "d",
	"alpha", "beta", "rc", "dev", "devel",
	"tp", "pre", "preview",
}

func (g *Generator) Semver() string {
	g.rngMu.Lock()
	defer g.rngMu.Unlock()

	maj, min, pat := g.ver.Uint64(), g.ver.Uint64(), g.ver.Uint64()
	var pre string
	if g.rng.Intn(20) < 19 { // 19/20 chance of pre-release version
		preParts := make([]string, g.npre.Uint64()+1)
		for i := range preParts {
			if g.rng.Intn(10) < 9 {
				preParts[i] = strconv.FormatUint(g.pre.Uint64(), 10)
			} else {
				preParts[i] = preReleaseStrings[g.rng.Intn(len(preReleaseStrings))]
			}
		}
		pre = "-" + strings.Join(preParts, ".")
	}
	return fmt.Sprintf("%d.%d.%d%s", maj, min, pat, pre)
}

/*
I want to generate a stream of unique chart (name, version) tuples. The
stream should be in random order to provide a worst-case scenario. Chart
names should be best-effort unique across multiple executions. I also want
the name and version to be somewhat pleasant on the eyes for debugging. Short
strings and small numbers would be nice.

The version needs to encompass the full range of values that'd be encountered
in the wild:
	- Released versions
	- Prereleases with varying numbers of components
	- Prerelease components with mixed strings and numbers

Generating short chart names is easy: short alphabetic/alphanumeric strings.
It doesn't even need to be random; just go in order "a", "b", ..., "aa", etc.
To make the names unique across runs, generate some random string and prepend
that onto every chart name.

The only interesting part of generating random versions is the prerelease
part.
	1. Pick a random number of prerelease segments in [0, n]
	2. For each prerelease segment, roll for whether it should be numeric or
	   string
	3. Generate a random number/string

There's probably enough entropy in the mix to make collisions unlikely. They
can be totally prevented by keeping track of all generated values and
retrying until a unique value is generated, if we care enough.

If I want 100,000 versions across 100 charts, I can pre-generate the 100
charts and randomly pick from the list when generating a version.
*/
