package kubeloader

import (
	"context"
	"net/http"

	"github.com/modern-engineering/prototype/application"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
)

type Health struct {
}

func (h *Health) Ready(ctx context.Context) error {
	//TODO implement me
	panic("implement me")
}

func (h *Health) Live(ctx context.Context) error {
	//TODO implement me
	panic("implement me")
}

// HandleHealth registers the given liveness and readiness handlers on the
// returned mux. The returned handler must be called on a prefix-stripped path
// (using [http.StripPrefix]).
//
// It mounts both /<path> and /<path>/ so kubelet's trailing-slash-less probe
// reaches the aggregated endpoint without an HTTP redirect.
func HandleHealth(liveness, readiness http.Handler) http.Handler {
	var mux http.ServeMux
	mux.Handle("/livez", http.StripPrefix("/livez", liveness))
	mux.Handle("/livez/", http.StripPrefix("/livez", liveness))
	mux.Handle("/readyz", http.StripPrefix("/readyz", readiness))
	mux.Handle("/readyz/", http.StripPrefix("/readyz", readiness))
	return &mux
}

func Liveness(checker application.HealthChecker) {
	var liveness = &healthz.Handler{Checks: map[string]healthz.Checker{}}
}

// Loader package globals

// TODO: healthz.Handler is *not* thread-safe, so we need our own wrapper/copy. maybe with a wrapper, instead of exposing
// package-level Register* functions, we can expose two package-level values (liveness, readiness)
var (
	instanceLiveness  healthz.Handler
	instanceReadiness healthz.Handler
)

func RegisterLiveness(instance string, checker healthz.Checker) {
	if instanceLiveness.Checks == nil {
		instanceLiveness.Checks = make(map[string]healthz.Checker)
	}
	instanceLiveness.Checks[instance] = checker
}

func RegisterReadiness(instance string, checker healthz.Checker) {
	if instanceReadiness.Checks == nil {
		instanceReadiness.Checks = make(map[string]healthz.Checker)
	}
	instanceReadiness.Checks[instance] = checker
}
