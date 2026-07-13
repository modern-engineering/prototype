package loaderflags_test

import (
	"flag"
	"fmt"
	"strings"
	"testing"

	"github.com/modern-engineering/prototype/application/loader/loaderflags"
)

// A greedySource ignores the priority contract: it writes every flag
// whether or not an earlier source claimed it.
type greedySource struct{}

func (greedySource) ParseParameters(fs *flag.FlagSet) error {
	var err error
	fs.VisitAll(func(f *flag.Flag) {
		if e := fs.Set(f.Name, "mine"); e != nil && err == nil {
			err = e
		}
	})
	return err
}

func (greedySource) String() string { return "greedy" }

// A failer records what a harness reported, so the harness self-test
// can assert on rejection without failing itself; Fatal* panics with a
// sentinel so the checked flow stops the way testing's Goexit would.
type failer struct {
	testing.TB
	failures []string
}

type failerStop struct{}

func (f *failer) Helper() {}
func (f *failer) Errorf(format string, args ...any) {
	f.failures = append(f.failures, fmt.Sprintf(format, args...))
}
func (f *failer) Fatal(args ...any) {
	f.failures = append(f.failures, fmt.Sprint(args...))
	panic(failerStop{})
}
func (f *failer) Fatalf(format string, args ...any) {
	f.failures = append(f.failures, fmt.Sprintf(format, args...))
	panic(failerStop{})
}

// checkFails runs CheckParser under the failer and returns what it
// reported; an empty slice means the harness accepted the parser.
func checkFails(t *testing.T, p loaderflags.Parser, name string) []string {
	t.Helper()
	f := &failer{}
	func() {
		defer func() {
			if r := recover(); r != nil {
				if _, ok := r.(failerStop); !ok {
					panic(r)
				}
			}
		}()
		loaderflags.CheckParser(f, p, name)
	}()
	return f.failures
}

// CheckParser accepts a source that leaves claimed flags alone and
// rejects one that stomps a higher-priority parser's claim, naming the
// stomped flag.
func TestCheckParserJudgesThePriorityContract(t *testing.T) {
	polite := mapSource{"polite", map[string]string{"claimed": "later"}}
	loaderflags.CheckParser(t, polite, "claimed")

	failures := checkFails(t, greedySource{}, "claimed")
	if len(failures) == 0 {
		t.Fatal("CheckParser accepted a parser that overwrites claimed flags")
	}
	if !strings.Contains(failures[0], `"claimed"`) {
		t.Errorf("rejection %q does not name the stomped flag", failures[0])
	}
}
