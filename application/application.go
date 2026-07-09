package application

import (
	"context"
	"flag"
	"fmt"
	"iter"
	"reflect"
	"runtime"
)

// A Descriptor describes a long-running applicative service function and its
// deployment.
type Descriptor struct {
	// The Name of the application must be a valid Go identifier
	// as it may appear in command-line flags, URLs, and so on.
	Name string

	// Doc is the documentation for the application.
	// The part before the first "\n\n" is the title
	// (no capital or period, max ~60 letters).
	Doc string

	// URL holds an optional link to a web page with additional
	// documentation for this application.
	URL string

	// New constructs the application service for running. It is the
	// fallible, context-aware construction path: loaders call it at run
	// time, where construction may perform I/O and refuse to proceed.
	// Optional; when nil, loaders construct via Make.
	New func(context.Context) (Service, error)

	// Make is the pure factory: it returns a fresh Service on every call,
	// declares the service's flags, and does nothing else — no I/O, no
	// side effects, no failure. Tooling relies on this dry-instantiation
	// invariant to obtain the parameter surface without running anything
	// (help rendering, schema extraction, solution compilation). Every
	// catalogue citizen must set Make; [MakeFor] and [MakeFunc] adapt the
	// common construction shapes.
	Make func() Service
}

func MakeFunc(fn func() (Runner, *flag.FlagSet)) func() Service {
	return func() Service {
		rr, fs := fn()
		return serviceFunc{Runner: rr, flags: fs}
	}
}

type serviceFunc struct {
	Runner
	flags *flag.FlagSet
}

func (x serviceFunc) Flags() *flag.FlagSet {
	return x.flags
}

// Flags exposes the parameter surface the application declares, obtained by
// dry instantiation: Make constructs a fresh throwaway Service whose flag
// set is returned and whose runner is discarded. Each call yields a fresh
// set, so callers may mutate the result freely.
//
// Flags may return nil when the application declares no parameters (for
// example, [Main] adapters). It panics when Make is nil: a Descriptor
// without a pure factory has no inspectable surface and is not a catalogue
// citizen.
func (a *Descriptor) Flags() *flag.FlagSet {
	if a.Make == nil {
		panic(fmt.Sprintf("application: descriptor %q has no Make factory", a.Name))
	}
	return a.Make().Flags()
}

type Runner interface {
	Run(ctx context.Context) error
}

// The RunnerFunc type is an adapter to allow the use of ordinary functions as
// application programs. If f is a function with the appropriate signature,
// Main(f) is a [Runner] that calls f.
type RunnerFunc func(ctx context.Context) error

func (f RunnerFunc) Run(ctx context.Context) error {
	return f(ctx)
}

// The Main type is an adapter to allow the use of ordinary functions as
// application programs. If f is a function with the appropriate signature,
// Main(f) is a [Runner] that calls f and has no Flags.
type Main func(ctx context.Context) error

func (f Main) Run(ctx context.Context) error {
	return f(ctx)
}

func (f Main) Flags() *flag.FlagSet {
	return nil
}

// String implements the fmt.Stringer interface. It uses reflection and the
// runtime package to extract the function's identity.
//
// Note: The accuracy of this method depends on the presence of symbol
// information (.symtab) in the compiled binary. If the binary is compiled with
// -s or -w flags, this will return the function pointer address.
func (f Main) String() string {
	ptr := reflect.ValueOf(f).Pointer()
	fn := runtime.FuncForPC(ptr)
	if fn == nil {
		return fmt.Sprintf("<no symbol info at %#x>", ptr)
	}
	return fn.Name()
}

type Service interface {
	Runner
	Flags() *flag.FlagSet
}

type ParamParser interface {
	Parse(ctx context.Context, flags *flag.FlagSet) error
}

type ParseTo func(ctx context.Context, flags *flag.FlagSet) error

func (f ParseTo) Parse(ctx context.Context, flags *flag.FlagSet) error {
	return f(ctx, flags)
}

func MakeFor[T any, PT interface {
	*T
	Runner
	Flags() *flag.FlagSet
}]() func() Service {
	return func() Service {
		var val T         // Native zero-value allocation (escape analysis handles heap vs stack).
		var ptr PT = &val // Take the address to satisfy the pointer constraint.
		return ptr
	}
}

type HealthChecker interface {
	Ready(ctx context.Context) error
	Live(ctx context.Context) error
}

type Shutdowner interface {
	Shutdown(ctx context.Context) error
}

// TODO(@danielorbach): add application.Context. It implements context.Context and provides access to application-specific data and functionality.

// TODO(@danielorbach): a Shutdowner/Terminator mixin that's easy to embed in structs and use.

// TODO(@danielorbach): a Loop that takes context, and optionally a counter, and loops it.
func Loop(ctx context.Context) iter.Seq[int] {
	return func(yield func(int) bool) {
		var i int
		for ctx.Err() == nil {
			if !yield(i) {
				return
			}
			i++
		}
	}
}

func LoopCount(ctx context.Context, count int) iter.Seq[int] {
	return func(yield func(int) bool) {
		var i int
		for ctx.Err() == nil && i < count {
			if !yield(i) {
				return
			}
			i++
		}
	}
}
