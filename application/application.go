package application

import (
	"context"
	"flag"
	"fmt"
	"reflect"
	"runtime"
	"sync"
)

func LoadApp(ctx context.Context, app Descriptor) {
	//env := &Environment{
	//	appName: app.name,
	//	parameters: []loaderflags.Parser{
	//		cmdlineParser{},
	//		envParser{},
	//	},
	//}
	//
	//main, err := app.newFunc(env)
	//if err != nil {
	//	panic(err) // TODO: improve.
	//}

	var wg sync.WaitGroup
	wg.Go(func() {
		//main(ctx)
	})
	wg.Wait()
}

func LoadHelp(app Descriptor) {
	//env := &Environment{
	//	appName: app.name,
	//	parameters: []loaderflags.Parser{
	//		helpParser{},
	//	},
	//}
	//
	//_, err := app.newFunc(env)
	//if !errors.Is(err, flag.ErrHelp) {
	//	panic(err) // TODO: improve.
	//}
	//env.FlagSet().Usage()
}

// --- registry.go ---

// TODO: add application.Context. It implements context.Context and provides access to application-specific data and functionality.

// --- descriptor.go ---

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

	// Run applies the analyzer to a package.
	// It returns an error if the analyzer failed.
	//
	// On success, the Run function may return a result
	// computed by the Analyzer; its type must match ResultType.
	// The driver makes this result available as an input to
	// another Analyzer that depends directly on this one (see
	// Requires) when it analyzes the same package.
	//
	// To pass analysis results between packages (and thus
	// potentially between address spaces), use Facts, which are
	// serializable.
	New func(context.Context) (Service, error)

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

// Flags exposes any flags accepted by the application. The manner in which these
// flags are exposed to the user depends on the driver which runs the analyzer.
func (a *Descriptor) Flags() *flag.FlagSet {
	panic("not implemented")
}

// --- end ---

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

func NewFor[T Service]() func() Service {
	return NewOf(reflect.TypeFor[T]())
}

func NewOf(typ reflect.Type) func() Service {
	return func() Service {
		if typ.Kind() == reflect.Ptr {
			typ = typ.Elem()
		}

		// While we support types with pointer-receivers, the type-param cannot be of
		// kind interface, unsafe-pointer, or pointer to another pointer.
		switch typ.Kind() {
		case reflect.Interface:
			fallthrough
		case reflect.UnsafePointer:
			fallthrough
		case reflect.Pointer:
			panic(fmt.Errorf("unsupported type %s of kind %s", typ, typ.Kind()))
		}

		ri := reflect.TypeFor[Service]()
		if !reflect.PointerTo(typ).Implements(ri) {
			panic(fmt.Errorf("type %s does not implement the application.Service interface", typ))
		}

		// We always return the pointer because that's the most common use of types that
		// implement Program. Also, any type that implements Program with a
		// value-receiver implicitly implements it on the pointer values too.
		v := reflect.New(typ).Interface().(Service)
		return v, v.Flags()
	}
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

type Instance struct {
	*Descriptor

	// Flags defines any flags accepted by the analyzer.
	// The manner in which these flags are exposed to the user
	// depends on the driver which runs the analyzer.
	Flags flag.FlagSet
}

type HealthChecker interface {
	Ready(ctx context.Context) error
	Live(ctx context.Context) error
}

type Shutdowner interface {
	Shutdown(ctx context.Context) error
}
