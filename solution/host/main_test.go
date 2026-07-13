// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package host_test

import (
	"context"
	"errors"
	"flag"
	"strings"
	"testing"

	"github.com/modern-engineering/prototype/application"
	"github.com/modern-engineering/prototype/solution"
	"github.com/modern-engineering/prototype/solution/host"
	"github.com/modern-engineering/prototype/solution/image"
)

// ----------------------------------------------------------------------------
// The park package: the process-skin test citizens main_test drives
// beside the mill. A parker proves the graceful sequence end to end; a
// faulty instance proves the wet-failure verdict.

const parkPath = "example.com/acme/park"

// A parker parks until its Shutdown is called: the graceful path.
// Flagless on purpose — a nil parameter surface must host fine.
type parker struct {
	started     chan<- struct{}
	sawDeadline chan<- bool
	stop        chan struct{} // closed by Shutdown to release Run
	done        chan struct{} // closed by Run on its way out
}

func (p *parker) Flags() *flag.FlagSet { return nil }

func (p *parker) Run(ctx context.Context) error {
	p.started <- struct{}{}
	defer close(p.done)
	select {
	case <-p.stop:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Shutdown records whether the grace context arrived with a deadline —
// the budget Main promises to pass verbatim — then stops the runner
// and waits for it to leave, bounded by the grace window.
func (p *parker) Shutdown(ctx context.Context) error {
	_, ok := ctx.Deadline()
	p.sawDeadline <- ok
	close(p.stop)
	select {
	case <-p.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func parkCatalogue(started chan<- struct{}, sawDeadline chan<- bool) []solution.Package {
	return []solution.Package{{
		Path: parkPath,
		Name: "park",
		Elements: []solution.Element{
			solution.App("Parker", &application.Descriptor{
				Name: "parker",
				Doc:  "park until asked to stop",
				Make: func() application.Service {
					return &parker{
						started:     started,
						sawDeadline: sawDeadline,
						stop:        make(chan struct{}),
						done:        make(chan struct{}),
					}
				},
			}),
			solution.App("Faulty", &application.Descriptor{
				Name: "faulty",
				Doc:  "fail as soon as it runs",
				Make: application.MakeFunc(func() (application.Runner, *flag.FlagSet) {
					return application.RunnerFunc(func(context.Context) error {
						return errors.New("burned out")
					}), nil
				}),
			}),
		},
	}}
}

// parkImage crafts a one-record image deploying the named park
// element. The park package is deliberately not pinned in the image's
// catalogue section: records resolve against the live catalogue, so
// the plan stands all the same.
func parkImage(elem string) *image.Image {
	img := millImage(nil, image.Record{
		Verb:    image.VerbDeploy,
		Element: image.Ref{Package: parkPath, Name: elem},
		Name:    "one",
	})
	return img
}

// ----------------------------------------------------------------------------
// The exit contract

// TestMainExitCodes pins the 0/1/2 verdicts: clean completion is 0,
// wet failures — a refusing driver, a failing instance — are 1, and
// input faults — unbound externs, a broken plan, no image at all —
// are 2, printed before anything wet runs.
func TestMainExitCodes(t *testing.T) {
	t.Run("clean completion", func(t *testing.T) {
		ctl := newStandControl()
		reports := make(chan report, 8)
		var log strings.Builder
		code := host.Main(host.Config{
			Image:     pingpongImage(),
			Catalogue: millCatalogue(ctl, reports),
			Externs:   externs(),
			Log:       &log,
		})
		if code != 0 {
			t.Errorf("Main() = %d, want 0; log:\n%s", code, log.String())
		}
		drain(t, reports, 3)
	})

	t.Run("instance failure", func(t *testing.T) {
		var log strings.Builder
		code := host.Main(host.Config{
			Image:     parkImage("Faulty"),
			Catalogue: append(millCatalogue(newStandControl(), nil), parkCatalogue(nil, nil)...),
			Log:       &log,
		})
		if code != 1 {
			t.Errorf("Main() = %d, want 1; log:\n%s", code, log.String())
		}
		wantLine(t, log.String(), `instance failed: burned out`)
	})

	t.Run("driver refusal", func(t *testing.T) {
		ctl := newStandControl()
		ctl.attachErr = errors.New("substrate said no")
		var log strings.Builder
		code := host.Main(host.Config{
			Image:     pingpongImage(),
			Catalogue: millCatalogue(ctl, make(chan report, 8)),
			Externs:   externs(),
			Log:       &log,
		})
		if code != 1 {
			t.Errorf("Main() = %d, want 1; log:\n%s", code, log.String())
		}
		wantLine(t, log.String(), `provision standIn: substrate said no`)
	})

	t.Run("unbound externs", func(t *testing.T) {
		var log strings.Builder
		code := host.Main(host.Config{
			Image:     pingpongImage(),
			Catalogue: millCatalogue(newStandControl(), make(chan report, 8)),
			Log:       &log,
		})
		if code != 2 {
			t.Errorf("Main() = %d, want 2; log:\n%s", code, log.String())
		}
		wantLine(t, log.String(), `unbound extern endpoint (mill.Endpoint): pass -extern endpoint=<value>`)
		wantLine(t, log.String(), `unbound extern credential (mill.Secret): pass -extern credential=<value>`)
	})

	t.Run("broken plan", func(t *testing.T) {
		var log strings.Builder
		code := host.Main(host.Config{
			Image:     millImage(nil, deployRec("ghost1", literal("subject", image.String("s")))),
			Catalogue: nil, // an empty catalogue registers nothing to resolve against
			Log:       &log,
		})
		if code != 2 {
			t.Errorf("Main() = %d, want 2; log:\n%s", code, log.String())
		}
		wantLine(t, log.String(), `image references mill.Echo but the catalogue registers no such element`)
	})

	t.Run("nil image", func(t *testing.T) {
		var log strings.Builder
		code := host.Main(host.Config{Log: &log})
		if code != 2 {
			t.Errorf("Main() = %d, want 2; log:\n%s", code, log.String())
		}
		wantLine(t, log.String(), `enact: nil image`)
	})

	t.Run("wet set refusal", func(t *testing.T) {
		var log strings.Builder
		code := host.Main(host.Config{
			Image:     millImage(nil, deployRec("Ping1", literal("count", image.String("many")))),
			Catalogue: millCatalogue(newStandControl(), nil),
			Log:       &log,
		})
		if code != 2 {
			t.Errorf("Main() = %d, want 2; log:\n%s", code, log.String())
		}
	})
}
