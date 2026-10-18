package command_test

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/modern-engineering/prototype/bind"
	"github.com/modern-engineering/prototype/bind/input"
	"github.com/modern-engineering/prototype/program"
	"github.com/modern-engineering/prototype/program/command"
)

// A command may mix its own flags with a descriptor's inputs without resetting
// constructor values or changing how the original setters handle assignments.
func TestCommandInputs(t *testing.T) {
	var inputs input.List
	name := inputs.String("name", "team", "audience")
	enabled := inputs.Bool("enabled", false, "enable delivery")
	count := inputs.Int("count", 7, "delivery count")
	var labels []string
	inputs.Var(program.SetterFunc(func(text string) error {
		labels = append(labels, text)
		return nil
	}), "label", "delivery label")
	var source flag.FlagSet
	region := source.String("region", "east", "deployment region")
	inputs.Var(source.Lookup("region").Value, "region", "deployment region")
	body := []byte("prepared")
	bodyHandle := bind.Bytes(&body)
	inputs.Var(bodyHandle, "body", "message body")

	flags := flag.NewFlagSet("send", flag.ContinueOnError)
	trace := flags.Bool("trace", false, "trace the command")
	if err := command.Parse(flags, &inputs, []string{
		"-trace", "-enabled", "-label=one", "-label", "two", "recipient", "-name=ignored",
	}); err != nil {
		t.Fatal(err)
	}
	if *name != "team" || *count != 7 || *region != "east" || string(body) != "prepared" {
		t.Fatalf("omitted inputs changed: name=%q count=%d region=%q body=%q", *name, *count, *region, body)
	}
	if !*enabled || !*trace || !slices.Equal(labels, []string{"one", "two"}) {
		t.Fatalf("assignments: enabled=%t trace=%t labels=%q", *enabled, *trace, labels)
	}
	if !slices.Equal(flags.Args(), []string{"recipient", "-name=ignored"}) {
		t.Fatalf("remaining arguments = %q", flags.Args())
	}
	if flags.Lookup("name").DefValue != "team" || flags.Lookup("region").DefValue != "east" {
		t.Fatal("defaults do not reflect constructor values")
	}
	field, _ := inputs.Lookup("body")
	if field.Value != bodyHandle {
		t.Fatal("input handle replaced")
	}
	reader, ok := field.Value.(io.ReaderFrom)
	if !ok {
		t.Fatal("input lost its reader capability")
	}
	if _, err := reader.ReadFrom(strings.NewReader("replacement")); err != nil {
		t.Fatal(err)
	}
	if string(body) != "replacement" {
		t.Fatalf("body = %q", body)
	}
}

// Help may render both command and descriptor flags, but neither registration nor
// usage may read a sensitive value, even when it implements Get and String.
func TestHelpDoesNotReadSensitiveDefaults(t *testing.T) {
	var inputs input.List
	inputs.String("region", "east", "deployment region")
	inputs.Var(secretValue{t: t}, "token", "access token", input.Sensitive)
	var output bytes.Buffer
	flags := flag.NewFlagSet("send", flag.ContinueOnError)
	flags.SetOutput(&output)
	flags.Bool("trace", false, "trace the command")
	usageCalls := 0
	flags.Usage = func() {
		usageCalls++
		if _, err := fmt.Fprintln(flags.Output(), "Send a message:"); err != nil {
			t.Fatal(err)
		}
		flags.PrintDefaults()
	}
	if err := command.Parse(flags, &inputs, []string{"-help"}); !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("Parse error = %v; want flag.ErrHelp", err)
	}
	if usageCalls != 1 || flags.Output() != &output {
		t.Fatalf("help handling: usage calls=%d output restored=%t", usageCalls, flags.Output() == &output)
	}
	for _, text := range []string{"Send a message:", "-region", "east", "-token", "-trace"} {
		if !strings.Contains(output.String(), text) {
			t.Errorf("help lacks %q: %s", text, &output)
		}
	}
	if strings.Contains(output.String(), "panic") {
		t.Fatalf("help could not inspect flag defaults: %s", &output)
	}
	if flags.Lookup("token").DefValue != "" {
		t.Fatalf("sensitive default = %q", flags.Lookup("token").DefValue)
	}
}

// Parser errors normally quote the rejected argument. The command boundary
// keeps it out of both returned text and automatic diagnostics while retaining
// the cause for deliberate programmatic handling.
func TestSensitiveFailuresRemainInspectable(t *testing.T) {
	for _, boolean := range []bool{false, true} {
		t.Run(fmt.Sprint(boolean), func(t *testing.T) {
			cause := errors.New("credential rejected")
			var inputs input.List
			inputs.Var(secretValue{t: t, cause: cause, boolean: boolean}, "token", "access token", input.Sensitive)
			var output bytes.Buffer
			flags := flag.NewFlagSet("send", flag.ContinueOnError)
			flags.SetOutput(&output)
			usageCalls := 0
			flags.Usage = func() { usageCalls++ }
			err := command.Parse(flags, &inputs, []string{"-token=private-value"})
			failure, ok := errors.AsType[*command.InputError](err)
			if !ok || failure.Name != "token" || !failure.Sensitive || !errors.Is(err, cause) {
				t.Fatalf("error does not preserve input and cause: %v", err)
			}
			if strings.Contains(err.Error(), "private-value") || strings.Contains(err.Error(), cause.Error()) {
				t.Fatalf("sensitive diagnostic: %v", err)
			}
			if output.Len() != 0 || usageCalls != 0 || flags.Output() != &output {
				t.Fatalf("parse failure emitted output or changed output policy: output=%q usage=%d", &output, usageCalls)
			}
			flags.Usage()
			if usageCalls != 1 {
				t.Fatal("caller usage function was not restored")
			}
		})
	}
}

// An argument may be rejected before the parser identifies its input. Sensitive
// text in such an argument must remain confined to the deliberately unwrapped
// cause rather than leaking through the ordinary command diagnostic.
func TestMalformedArgumentsKeepValuesPrivate(t *testing.T) {
	var inputs input.List
	inputs.Var(secretValue{t: t}, "token", "access token", input.Sensitive)
	var output bytes.Buffer
	flags := flag.NewFlagSet("send", flag.ContinueOnError)
	flags.SetOutput(&output)
	err := command.Parse(flags, &inputs, []string{"---token=private-value"})
	if err == nil {
		t.Fatal("Parse accepted malformed flag syntax")
	}
	if strings.Contains(err.Error(), "private-value") || output.Len() != 0 {
		t.Fatalf("malformed argument leaked: error=%v output=%q", err, &output)
	}
	if errors.Unwrap(err) == nil {
		t.Fatal("parser diagnostic is unavailable through Unwrap")
	}
	if _, ok := errors.AsType[*command.InputError](err); ok {
		t.Fatalf("syntax rejection incorrectly identifies a field: %v", err)
	}
}

// Parsing uses the original setters, so earlier assignments survive a later
// rejection and the caller can inspect the actual validation error.
func TestParsingStopsAtRejectedInput(t *testing.T) {
	var inputs input.List
	name := inputs.String("name", "original", "audience")
	count := inputs.Int("count", int8(7), "delivery count")
	var output bytes.Buffer
	flags := flag.NewFlagSet("send", flag.ContinueOnError)
	flags.SetOutput(&output)
	err := command.Parse(flags, &inputs, []string{"-name=changed", "-count=999", "-name=unseen"})
	failure, ok := errors.AsType[*command.InputError](err)
	if !ok || failure.Name != "count" || failure.Sensitive {
		t.Fatalf("error lacks rejected input: %v", err)
	}
	if _, ok := errors.AsType[*strconv.NumError](err); !ok {
		t.Fatalf("error lost numeric validation cause: %v", err)
	}
	if *name != "changed" || *count != 7 {
		t.Fatalf("state after rejected assignment: name=%q count=%d", *name, *count)
	}
	if output.Len() != 0 {
		t.Fatalf("automatic error output = %q", &output)
	}
}

func TestContinueOnErrorRequired(t *testing.T) {
	flags := flag.NewFlagSet("send", flag.PanicOnError)
	defer func() {
		if recover() == nil {
			t.Error("Parse accepted flag.PanicOnError")
		}
	}()
	_ = command.Parse(flags, nil, nil)
}

type secretValue struct {
	t       *testing.T
	cause   error
	boolean bool
}

func (v secretValue) Get() string {
	v.t.Fatal("sensitive Get called")
	return ""
}

func (v secretValue) String() string {
	v.t.Fatal("sensitive String called")
	return ""
}

func (v secretValue) Set(text string) error {
	if v.cause == nil {
		v.t.Fatal("setter called without an assignment")
	}
	return fmt.Errorf("rejected %q: %w", text, v.cause)
}

func (v secretValue) IsBoolFlag() bool { return v.boolean }
