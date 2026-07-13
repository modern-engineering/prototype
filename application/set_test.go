package application_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/modern-engineering/prototype/application"
)

func named(name string) *application.Descriptor {
	return &application.Descriptor{
		Name: name,
		Make: func() application.Service {
			return application.Main(func(ctx context.Context) error { return nil })
		},
	}
}

// NewSet refuses what descriptor values cannot check about each other:
// nil entries, non-identifier names, the same descriptor listed twice,
// and two descriptors colliding on a name.
func TestNewSet(t *testing.T) {
	ping, pong := named("ping"), named("pong")

	tests := []struct {
		name    string
		ds      []*application.Descriptor
		wantErr string // substring; empty means success
	}{
		{name: "empty", ds: nil},
		{name: "two", ds: []*application.Descriptor{ping, pong}},
		{name: "nil descriptor", ds: []*application.Descriptor{ping, nil}, wantErr: "descriptor 1 is nil"},
		{name: "empty name", ds: []*application.Descriptor{named("")}, wantErr: "invalid name"},
		{name: "keyword name", ds: []*application.Descriptor{named("func")}, wantErr: "invalid name"},
		{name: "dashed name", ds: []*application.Descriptor{named("pi-ng")}, wantErr: "invalid name"},
		{name: "same descriptor twice", ds: []*application.Descriptor{ping, ping}, wantErr: `"ping" listed twice`},
		{name: "name collision", ds: []*application.Descriptor{ping, named("ping")}, wantErr: `duplicate descriptor name "ping"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := application.NewSet(tt.ds...)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("NewSet error = %v, want containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("NewSet: %v", err)
			}
			if s.Len() != len(tt.ds) {
				t.Fatalf("Len = %d, want %d", s.Len(), len(tt.ds))
			}
		})
	}
}

// A Set assembled from descriptors resolves names back to the very
// same descriptor values, answers nil for strangers, and iterates in
// the order the descriptors were given.
func ExampleNewSet() {
	ping, pong := named("ping"), named("pong")
	set, err := application.NewSet(ping, pong)
	if err != nil {
		fmt.Println("assemble:", err)
		return
	}
	fmt.Println("len:", set.Len())
	fmt.Println("lookup ping is ping:", set.Lookup("ping") == ping)
	fmt.Println("lookup gone:", set.Lookup("gone"))
	for d := range set.All() {
		fmt.Println("member:", d.Name)
	}

	// Output:
	// len: 2
	// lookup ping is ping: true
	// lookup gone: <nil>
	// member: ping
	// member: pong
}
