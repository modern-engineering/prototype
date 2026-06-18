module github.com/modern-engineering/prototype/samples/kubepod

go 1.26.0

require (
	github.com/modern-engineering/prototype v0.0.0
	sigs.k8s.io/controller-runtime v0.24.1
)

require (
	github.com/go-logr/logr v1.4.3 // indirect
	k8s.io/apimachinery v0.36.0 // indirect
)

replace github.com/modern-engineering/prototype => ../../
