package application_test

import (
	"context"
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

func TestSetLookupAndOrder(t *testing.T) {
	ping, pong := named("ping"), named("pong")
	s, err := application.NewSet(ping, pong)
	if err != nil {
		t.Fatal(err)
	}
	if got := s.Lookup("ping"); got != ping {
		t.Fatalf("Lookup(ping) = %v, want the ping descriptor", got)
	}
	if got := s.Lookup("gone"); got != nil {
		t.Fatalf("Lookup(gone) = %v, want nil", got)
	}
	var order []string
	for d := range s.All() {
		order = append(order, d.Name)
	}
	if len(order) != 2 || order[0] != "ping" || order[1] != "pong" {
		t.Fatalf("All order = %v, want [ping pong]", order)
	}
}
