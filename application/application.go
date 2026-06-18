package application

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"reflect"
	"runtime"
	"strings"
	"sync"

	"github.com/modern-engineering/prototype/application/loader/loaderflags"
)

// --- loader.go ---

var ResolveParam = func(name string, hasValue bool) (string, bool) {
	v, ok := os.LookupEnv(name)
	if ok {
		return v, true
	}
	return "", false
}

// TODO: copy more parts of "flag/flag.go" (Flag.parseOne).
func setFlagArg(f *flag.Flag) error {
	// TODO: check for invalid param names

	var (
		long    = "--" + f.Name
		short   = "-" + f.Name
		boolean = false
	)

	// TODO: comment on boolean flags
	if bf, ok := f.Value.(interface{ IsBoolFlag() bool }); ok {
		boolean = bf.IsBoolFlag()
	}

	for i := 1; i < len(os.Args); i++ {
		arg := os.Args[i]
		// Handle '-name=value' and '--name=value' forms.
		before, after, found := strings.Cut(arg, "=")
		if found {
			if before == short || before == long {
				return f.Value.Set(after)
			}
			continue
		}

		if boolean {
			return f.Value.Set("true")
		}

		// Handle '-name value' or '--name value' forms.
		if arg == short || arg == long {
			if i+1 < len(os.Args) {
				return f.Value.Set(os.Args[i+1])
			}
			return fmt.Errorf("flag needs an argument: -%s", f.Name)
		}
	}
	return nil
	// TODO: return fmt.Errorf("flag defined but not provided: -%s", f.Name)
}

func LoadApp(ctx context.Context, app Descriptor) {
	env := &Environment{
		appName: app.name,
		parameters: []loaderflags.Parser{
			cmdlineParser{},
			envParser{},
		},
	}

	main, err := app.newFunc(env)
	if err != nil {
		panic(err) // TODO: improve.
	}

	var wg sync.WaitGroup
	wg.Go(func() {
		main(ctx)
	})
	wg.Wait()
}

func LoadHelp(app Descriptor) {
	env := &Environment{
		appName: app.name,
		parameters: []loaderflags.Parser{
			helpParser{},
		},
	}

	_, err := app.newFunc(env)
	if !errors.Is(err, flag.ErrHelp) {
		panic(err) // TODO: improve.
	}
	env.FlagSet().Usage()
}

type helpParser struct{}

func (p helpParser) String() string {
	return "help"
}

func (p helpParser) ParseParameters(fs *flag.FlagSet) error {
	return flag.ErrHelp
}

// --- end ---

// --- environment.go ---

type Environment struct {
	appName string

	fs         *flag.FlagSet // lazy-initialized
	parameters []loaderflags.Parser
}

func (e *Environment) AppName() string {
	return e.appName
}

// --- end ---

// --- registry.go ---

// TODO: add application.Context. It implements context.Context and provides access to application-specific data and functionality.

type MainFunc func(ctx context.Context)

type NewFunc func(env *Environment) (MainFunc, error)

func Register(name string, init NewFunc) Descriptor {
	return Descriptor{
		name:    name,
		newFunc: init,
	}
}

// --- end ---

// --- descriptor.go ---

type Descriptor struct {
	name   string
	doc    string
	docURL string

	newFunc  NewFunc
	metadata any
}

// The Name of the analyzer must be a valid Go identifier
// as it may appear in command-line flags, URLs, and so on.
func (d *Descriptor) Name() string {
	return d.name
}

// Doc is the documentation for the analyzer.
// The part before the first "\n\n" is the title
// (no capital or period, max ~60 letters).
func (d *Descriptor) Doc() string {
	return d.doc
}

// DocURL holds an optional link to a web page with additional
// documentation for this analyzer.
func (d *Descriptor) DocURL() string {
	return d.docURL
}

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
//func (d *Descriptor) Run() func(*Pass) (any, error) {
//	return d.run
//}

// --- end ---

// --- flags.go ---

func (e *Environment) FlagSet() *flag.FlagSet {
	if e.fs == nil {
		e.fs = loaderflags.NewFlagSet(e.AppName())
	}
	return e.fs
}

func (e *Environment) Parameterize() error {
	return loaderflags.Parse(e.FlagSet(), e.parameters...)
}

type cmdlineParser struct{}

func (p cmdlineParser) String() string {
	return "command-line"
}

func (p cmdlineParser) ParseParameters(fs *flag.FlagSet) error {
	return fs.Parse(os.Args[1:])
}

type envParser struct{}

func (p envParser) String() string {
	return "environment"
}

func (p envParser) ParseParameters(fs *flag.FlagSet) error {
	for f := range loaderflags.PendingParams(fs) {
		varName := strings.Map(p.varSafeName, f.Name)
		s, ok := os.LookupEnv(varName)
		if ok {
			if err := fs.Set(f.Name, s); err != nil {
				return fmt.Errorf("set %q: %w", f.Name, err)
			}
		}
	}
	return nil
}

func (p envParser) varSafeName(r rune) rune {
	switch {
	case 'a' <= r && r <= 'z':
		return 'A' + r - 'a'
	case 'A' <= r && r <= 'Z':
		return r
	case '0' <= r && r <= '9':
		return r
	default:
		return '_'
	}
}

// --- end ---

type Program interface {
	Run(ctx context.Context) error
}

// The Func type is an adapter to allow the use of ordinary functions as
// application programs. If f is a function with the appropriate signature,
// Func(f) is a [Program] that calls f.
type Func func(ctx context.Context) error

func (f Func) Run(ctx context.Context) error {
	return f(ctx)
}

// String implements the fmt.Stringer interface. It uses reflection and the
// runtime package to extract the function's identity.
//
// Note: The accuracy of this method depends on the presence of symbol
// information (.symtab) in the compiled binary. If the binary is compiled with
// -s or -w flags, this will return the function pointer address.
func (f Func) String() string {
	ptr := reflect.ValueOf(f).Pointer()
	fn := runtime.FuncForPC(ptr)
	if fn == nil {
		return fmt.Sprintf("<no symbol info at %#x>", ptr)
	}
	return fn.Name()
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
