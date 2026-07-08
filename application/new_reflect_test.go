package application_test

import (
	"context"
	"flag"
	"reflect"
	"testing"
	"unsafe"

	"github.com/modern-engineering/prototype/application"
)

// The underlying types interface{}, pointer, and unsafe-pointer, cannot hold a
// method-set, so these would of-course cause compilation errors because the type
// constraint demands a Runner.
type (
	Bool          bool
	Int           int
	Int8          int8
	Int16         int16
	Int32         int32
	Int64         int64
	Uint          uint
	Uint8         uint8
	Uint16        uint16
	Uint32        uint32
	Uint64        uint64
	Uintptr       uintptr
	Float32       float32
	Float64       float64
	Complex64     complex64
	Complex128    complex128
	Array         []struct{}
	Chan          chan struct{}
	Func          func()
	Interface     interface{}
	Map           map[struct{}]struct{}
	Pointer       *Struct
	Slice         []struct{}
	String        string
	Struct        struct{}
	UnsafePointer unsafe.Pointer
)

var sentinelFlags = new(flag.FlagSet)

func (*Bool) Run(context.Context) error       { return nil }
func (*Bool) Flags() *flag.FlagSet            { return sentinelFlags }
func (*Int) Run(context.Context) error        { return nil }
func (*Int) Flags() *flag.FlagSet             { return sentinelFlags }
func (*Int8) Run(context.Context) error       { return nil }
func (*Int8) Flags() *flag.FlagSet            { return sentinelFlags }
func (*Int16) Run(context.Context) error      { return nil }
func (*Int16) Flags() *flag.FlagSet           { return sentinelFlags }
func (*Int32) Run(context.Context) error      { return nil }
func (*Int32) Flags() *flag.FlagSet           { return sentinelFlags }
func (*Int64) Run(context.Context) error      { return nil }
func (*Int64) Flags() *flag.FlagSet           { return sentinelFlags }
func (*Uint) Run(context.Context) error       { return nil }
func (*Uint) Flags() *flag.FlagSet            { return sentinelFlags }
func (*Uint8) Run(context.Context) error      { return nil }
func (*Uint8) Flags() *flag.FlagSet           { return sentinelFlags }
func (*Uint16) Run(context.Context) error     { return nil }
func (*Uint16) Flags() *flag.FlagSet          { return sentinelFlags }
func (*Uint32) Run(context.Context) error     { return nil }
func (*Uint32) Flags() *flag.FlagSet          { return sentinelFlags }
func (*Uint64) Run(context.Context) error     { return nil }
func (*Uint64) Flags() *flag.FlagSet          { return sentinelFlags }
func (*Uintptr) Run(context.Context) error    { return nil }
func (*Uintptr) Flags() *flag.FlagSet         { return sentinelFlags }
func (*Float32) Run(context.Context) error    { return nil }
func (*Float32) Flags() *flag.FlagSet         { return sentinelFlags }
func (*Float64) Run(context.Context) error    { return nil }
func (*Float64) Flags() *flag.FlagSet         { return sentinelFlags }
func (*Complex64) Run(context.Context) error  { return nil }
func (*Complex64) Flags() *flag.FlagSet       { return sentinelFlags }
func (*Complex128) Run(context.Context) error { return nil }
func (*Complex128) Flags() *flag.FlagSet      { return sentinelFlags }
func (*Array) Run(context.Context) error      { return nil }
func (*Array) Flags() *flag.FlagSet           { return sentinelFlags }
func (*Chan) Run(context.Context) error       { return nil }
func (*Chan) Flags() *flag.FlagSet            { return sentinelFlags }
func (*Func) Run(context.Context) error       { return nil }
func (*Func) Flags() *flag.FlagSet            { return sentinelFlags }
func (*Map) Run(context.Context) error        { return nil }
func (*Map) Flags() *flag.FlagSet             { return sentinelFlags }
func (*Slice) Run(context.Context) error      { return nil }
func (*Slice) Flags() *flag.FlagSet           { return sentinelFlags }
func (*String) Run(context.Context) error     { return nil }
func (*String) Flags() *flag.FlagSet          { return sentinelFlags }
func (*Struct) Run(context.Context) error     { return nil }
func (*Struct) Flags() *flag.FlagSet          { return sentinelFlags }

// TODO(@claude): carefully explain why we test both value and pointer receivers. Ask me about it.
func TestMakeForKinds(t *testing.T) {
	// The underlying types interface{}, pointer, and unsafe-pointer, cannot hold a
	// method-set, so these would cause compilation errors because types with them as
	// underlying layout cannot meet the type constraint of Program.
	t.Run("TypeParam=Invalid", func(t *testing.T) {
		//_ = application.MakeFor[Interface]()
		t.Logf("MakeFor[Interface]() should fail to compile (check manually)")
		//_ = application.MakeFor[Pointer]()
		t.Logf("MakeFor[Pointer]() should fail to compile (check manually)")
		//_ = application.MakeFor[UnsafePointer]()
		t.Logf("MakeFor[UnsafePointer]() should fail to compile (check manually)")
		//_ = application.MakeFor[string]()
		t.Logf("MakeFor[arbitrary-type]() should fail to compile (check manually)")
	})

	tests := []struct {
		typ      reflect.Type
		makeFunc func() application.Service
	}{
		{typ: reflect.TypeFor[Bool](), makeFunc: application.MakeFor[Bool]()},
		{typ: reflect.TypeFor[Int](), makeFunc: application.MakeFor[Int]()},
		{typ: reflect.TypeFor[Int8](), makeFunc: application.MakeFor[Int8]()},
		{typ: reflect.TypeFor[Int16](), makeFunc: application.MakeFor[Int16]()},
		{typ: reflect.TypeFor[Int32](), makeFunc: application.MakeFor[Int32]()},
		{typ: reflect.TypeFor[Int64](), makeFunc: application.MakeFor[Int64]()},
		{typ: reflect.TypeFor[Uint](), makeFunc: application.MakeFor[Uint]()},
		{typ: reflect.TypeFor[Uint8](), makeFunc: application.MakeFor[Uint8]()},
		{typ: reflect.TypeFor[Uint16](), makeFunc: application.MakeFor[Uint16]()},
		{typ: reflect.TypeFor[Uint32](), makeFunc: application.MakeFor[Uint32]()},
		{typ: reflect.TypeFor[Uint64](), makeFunc: application.MakeFor[Uint64]()},
		{typ: reflect.TypeFor[Uintptr](), makeFunc: application.MakeFor[Uintptr]()},
		{typ: reflect.TypeFor[Float32](), makeFunc: application.MakeFor[Float32]()},
		{typ: reflect.TypeFor[Float64](), makeFunc: application.MakeFor[Float64]()},
		{typ: reflect.TypeFor[Complex64](), makeFunc: application.MakeFor[Complex64]()},
		{typ: reflect.TypeFor[Complex128](), makeFunc: application.MakeFor[Complex128]()},
		{typ: reflect.TypeFor[Array](), makeFunc: application.MakeFor[Array]()},
		{typ: reflect.TypeFor[Chan](), makeFunc: application.MakeFor[Chan]()},
		{typ: reflect.TypeFor[Func](), makeFunc: application.MakeFor[Func]()},
		{typ: reflect.TypeFor[Map](), makeFunc: application.MakeFor[Map]()},
		{typ: reflect.TypeFor[Slice](), makeFunc: application.MakeFor[Slice]()},
		{typ: reflect.TypeFor[String](), makeFunc: application.MakeFor[String]()},
		{typ: reflect.TypeFor[Struct](), makeFunc: application.MakeFor[Struct]()},
		// Compilation errors ("Type does not implement...")
		//{kind: reflect.Interface, makeFunc: application.MakeFor[Interface]()},
		//{kind: reflect.Pointer, makeFunc: application.MakeFor[Pointer]()},
		//{kind: reflect.UnsafePointer, makeFunc: application.MakeFor[UnsafePointer]()},
	}

	for _, tt := range tests {
		svc := tt.makeFunc()
		t.Logf("MakeFor[%v]() of kind %q = %#v", tt.typ, tt.typ.Kind(), svc)
		if svc == nil {
			t.Errorf("MakeFor(%v) = nil Service", tt.typ)
		}
		if svc.Flags() != sentinelFlags {
			t.Errorf("MakeFor(%v) = unexpected Flags", tt.typ)
		}
		// The returned Service value is always a pointer, but it must point to the
		// zero-value of the type parameter passed to MakeFor. The most important test is
		// that the pointer points to a value; we just know it is a zero one, so we test
		// for it too.
		rv := reflect.ValueOf(svc)
		if !rv.Elem().IsZero() {
			t.Errorf("MakeFor(%v) points to non-zero-value of type %v", tt.typ.Kind(), rv.Type())
		}
	}
}
