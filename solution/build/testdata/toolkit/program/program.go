// Package program supplies the registration surface needed by generated hosts.
package program

type Registry struct{}

type MakeFunc func()

func (*Registry) Register(string, string, any) error { return nil }
