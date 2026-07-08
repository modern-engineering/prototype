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

func (Bool) Run(context.Context) error       { return nil }
func (Bool) Flags() *flag.FlagSet            { return sentinelFlags }
func (Int) Run(context.Context) error        { return nil }
func (Int) Flags() *flag.FlagSet             { return sentinelFlags }
func (Int8) Run(context.Context) error       { return nil }
func (Int8) Flags() *flag.FlagSet            { return sentinelFlags }
func (Int16) Run(context.Context) error      { return nil }
func (Int16) Flags() *flag.FlagSet           { return sentinelFlags }
func (Int32) Run(context.Context) error      { return nil }
func (Int32) Flags() *flag.FlagSet           { return sentinelFlags }
func (Int64) Run(context.Context) error      { return nil }
func (Int64) Flags() *flag.FlagSet           { return sentinelFlags }
func (Uint) Run(context.Context) error       { return nil }
func (Uint) Flags() *flag.FlagSet            { return sentinelFlags }
func (Uint8) Run(context.Context) error      { return nil }
func (Uint8) Flags() *flag.FlagSet           { return sentinelFlags }
func (Uint16) Run(context.Context) error     { return nil }
func (Uint16) Flags() *flag.FlagSet          { return sentinelFlags }
func (Uint32) Run(context.Context) error     { return nil }
func (Uint32) Flags() *flag.FlagSet          { return sentinelFlags }
func (Uint64) Run(context.Context) error     { return nil }
func (Uint64) Flags() *flag.FlagSet          { return sentinelFlags }
func (Uintptr) Run(context.Context) error    { return nil }
func (Uintptr) Flags() *flag.FlagSet         { return sentinelFlags }
func (Float32) Run(context.Context) error    { return nil }
func (Float32) Flags() *flag.FlagSet         { return sentinelFlags }
func (Float64) Run(context.Context) error    { return nil }
func (Float64) Flags() *flag.FlagSet         { return sentinelFlags }
func (Complex64) Run(context.Context) error  { return nil }
func (Complex64) Flags() *flag.FlagSet       { return sentinelFlags }
func (Complex128) Run(context.Context) error { return nil }
func (Complex128) Flags() *flag.FlagSet      { return sentinelFlags }
func (Array) Run(context.Context) error      { return nil }
func (Array) Flags() *flag.FlagSet           { return sentinelFlags }
func (Chan) Run(context.Context) error       { return nil }
func (Chan) Flags() *flag.FlagSet            { return sentinelFlags }
func (Func) Run(context.Context) error       { return nil }
func (Func) Flags() *flag.FlagSet            { return sentinelFlags }
func (Map) Run(context.Context) error        { return nil }
func (Map) Flags() *flag.FlagSet             { return sentinelFlags }
func (Slice) Run(context.Context) error      { return nil }
func (Slice) Flags() *flag.FlagSet           { return sentinelFlags }
func (String) Run(context.Context) error     { return nil }
func (String) Flags() *flag.FlagSet          { return sentinelFlags }
func (Struct) Run(context.Context) error     { return nil }
func (Struct) Flags() *flag.FlagSet          { return sentinelFlags }

type PointerReceiver struct{}

func (*PointerReceiver) Run(context.Context) error { return nil }
func (*PointerReceiver) Flags() *flag.FlagSet      { return sentinelFlags }

// TODO(@claude): carefully explain why we test both value and pointer receivers. Ask me about it.
func TestNewForTypes(t *testing.T) {
	t.Run("Receiver=Value", func(t *testing.T) {
		// The underlying types interface{}, pointer, and unsafe-pointer, cannot hold a
		// method-set, so these would cause compilation errors because types with them as
		// underlying layout cannot meet the type constraint of Program.
		t.Run("TypeParam=Invalid", func(t *testing.T) {
			//_ = application.NewFor[Interface]()
			t.Logf("NewFor[Interface]() should fail to compile (check manually)")
			//_ = application.NewFor[Pointer]()
			t.Logf("NewFor[Pointer]() should fail to compile (check manually)")
			//_ = application.NewFor[UnsafePointer]()
			t.Logf("NewFor[UnsafePointer]() should fail to compile (check manually)")
			//_ = application.NewFor[string]()
			t.Logf("NewFor[arbitrary-type]() should fail to compile (check manually)")
		})
		t.Run("TypeParam=Value", func(t *testing.T) {
			tests := []struct {
				kind     reflect.Kind
				makeFunc func() (application.Runner, *flag.FlagSet)
			}{
				{kind: reflect.Bool, makeFunc: application.NewFor[Bool]()},
				{kind: reflect.Int, makeFunc: application.NewFor[Int]()},
				{kind: reflect.Int8, makeFunc: application.NewFor[Int8]()},
				{kind: reflect.Int16, makeFunc: application.NewFor[Int16]()},
				{kind: reflect.Int32, makeFunc: application.NewFor[Int32]()},
				{kind: reflect.Int64, makeFunc: application.NewFor[Int64]()},
				{kind: reflect.Uint, makeFunc: application.NewFor[Uint]()},
				{kind: reflect.Uint8, makeFunc: application.NewFor[Uint8]()},
				{kind: reflect.Uint16, makeFunc: application.NewFor[Uint16]()},
				{kind: reflect.Uint32, makeFunc: application.NewFor[Uint32]()},
				{kind: reflect.Uint64, makeFunc: application.NewFor[Uint64]()},
				{kind: reflect.Uintptr, makeFunc: application.NewFor[Uintptr]()},
				{kind: reflect.Float32, makeFunc: application.NewFor[Float32]()},
				{kind: reflect.Float64, makeFunc: application.NewFor[Float64]()},
				{kind: reflect.Complex64, makeFunc: application.NewFor[Complex64]()},
				{kind: reflect.Complex128, makeFunc: application.NewFor[Complex128]()},
				{kind: reflect.Array, makeFunc: application.NewFor[Array]()},
				{kind: reflect.Chan, makeFunc: application.NewFor[Chan]()},
				{kind: reflect.Func, makeFunc: application.NewFor[Func]()},
				{kind: reflect.Map, makeFunc: application.NewFor[Map]()},
				{kind: reflect.Slice, makeFunc: application.NewFor[Slice]()},
				{kind: reflect.String, makeFunc: application.NewFor[String]()},
				{kind: reflect.Struct, makeFunc: application.NewFor[Struct]()},
				// Compilation errors ("Type does not implement...")
				//{kind: reflect.Interface, makeFunc: application.NewFor[Interface]()},
				//{kind: reflect.Pointer, makeFunc: application.NewFor[Pointer]()},
				//{kind: reflect.UnsafePointer, makeFunc: application.NewFor[UnsafePointer]()},
			}

			for _, tt := range tests {
				v, fs := tt.makeFunc()
				t.Logf("Make(%v) = %#v", tt.kind, v)
				if v == nil || fs != sentinelFlags {
					t.Errorf("Make(%v) returned nil", tt.kind)
				}
			}
		})

		t.Run("TypeParam=Pointer", func(t *testing.T) {
			tests := []struct {
				kind     reflect.Kind
				makeFunc func() (application.Runner, *flag.FlagSet)
			}{
				{kind: reflect.Bool, makeFunc: application.NewFor[*Bool]()},
				{kind: reflect.Int, makeFunc: application.NewFor[*Int]()},
				{kind: reflect.Int8, makeFunc: application.NewFor[*Int8]()},
				{kind: reflect.Int16, makeFunc: application.NewFor[*Int16]()},
				{kind: reflect.Int32, makeFunc: application.NewFor[*Int32]()},
				{kind: reflect.Int64, makeFunc: application.NewFor[*Int64]()},
				{kind: reflect.Uint, makeFunc: application.NewFor[*Uint]()},
				{kind: reflect.Uint8, makeFunc: application.NewFor[*Uint8]()},
				{kind: reflect.Uint16, makeFunc: application.NewFor[*Uint16]()},
				{kind: reflect.Uint32, makeFunc: application.NewFor[*Uint32]()},
				{kind: reflect.Uint64, makeFunc: application.NewFor[*Uint64]()},
				{kind: reflect.Uintptr, makeFunc: application.NewFor[*Uintptr]()},
				{kind: reflect.Float32, makeFunc: application.NewFor[*Float32]()},
				{kind: reflect.Float64, makeFunc: application.NewFor[*Float64]()},
				{kind: reflect.Complex64, makeFunc: application.NewFor[*Complex64]()},
				{kind: reflect.Complex128, makeFunc: application.NewFor[*Complex128]()},
				{kind: reflect.Array, makeFunc: application.NewFor[*Array]()},
				{kind: reflect.Chan, makeFunc: application.NewFor[*Chan]()},
				{kind: reflect.Func, makeFunc: application.NewFor[*Func]()},
				{kind: reflect.Map, makeFunc: application.NewFor[*Map]()},
				{kind: reflect.Slice, makeFunc: application.NewFor[*Slice]()},
				{kind: reflect.String, makeFunc: application.NewFor[*String]()},
				{kind: reflect.Struct, makeFunc: application.NewFor[*Struct]()},
				// Compilation errors ("Type does not implement...")
				//{kind: reflect.Interface, makeFunc: application.NewFor[*Interface]()},
				//{kind: reflect.Pointer, makeFunc: application.NewFor[*Pointer]()},
				//{kind: reflect.UnsafePointer, makeFunc: application.NewFor[*UnsafePointer]()},
			}

			for _, tt := range tests {
				v, fs := tt.makeFunc()
				t.Logf("Make(*%v) = %#v", tt.kind, v)
				if v == nil || fs != sentinelFlags {
					t.Errorf("Make(*%v) returned nil", tt.kind)
				}
			}
		})
	})

	t.Run("Receiver=Pointer", func(t *testing.T) {
		t.Run("TypeParam=ValueIsInvalid", func(t *testing.T) {
			// Compilation error: "Type does not implement Program as the Run method has a pointer receiver"
			// application.NewFor[PointerReceiver]()
			t.Logf("NewFor[PointerReceiver]() should fail to compile (check manually)")
		})
		t.Run("TypeParam=Pointer", func(t *testing.T) {
			testNewFor[*PointerReceiver](t)
			//makeFunc := application.NewFor[*PointerReceiver]()
			//v, fs := makeFunc()
			//t.Logf("NewFor[*PointerReceiver]() = %#v", v)
			//if v == nil || fs != sentinelFlags {
			//	t.Errorf("NewFor[PointerReceiver]() returned nil")
			//}
		})
	})
}

func testNewFor[T application.Service](t *testing.T) {
	t.Helper()

	makeFunc := application.NewFor[T]()
	v, fs := makeFunc()

	rt := reflect.TypeFor[T]()
	t.Logf("application.NewFor[%v]() = %#v", rt, v)
	if v == nil {
		t.Errorf("NewFor[%v]() = nil Program", rt)
	}
	if fs != sentinelFlags {
		t.Errorf("NewFor[%v]() = unexpected Flags", rt)
	}

}

func TestMakeForTypes(t *testing.T) {
	makeFunc := application.MakeFor[PointerReceiver]()
	v, fs := makeFunc()

	t.Logf("application.MakeFor[PointerReceiver]() = %#v", v)
	if v == nil {
		t.Errorf("MakeFor[PointerReceiver]() = nil Program")
	}
	if fs != sentinelFlags {
		t.Errorf("MakeFor[PointerReceiver]() = unexpected Flags")
	}

}
