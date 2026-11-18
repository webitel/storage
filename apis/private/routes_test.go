package private

import (
	"testing"

	"github.com/gorilla/mux"

	"github.com/webitel/storage/pkg/sysurl"
)

func TestSysRoutesNamed(t *testing.T) {
	root := mux.NewRouter()
	Init(nil, root)

	seen := make(map[sysurl.Route]bool)
	err := root.Walk(func(route *mux.Route, _ *mux.Router, _ []*mux.Route) error {
		if route.GetHandler() == nil {
			return nil
		}
		path, _ := route.GetPathTemplate()
		methods, _ := route.GetMethods()
		name := sysurl.Route(route.GetName())
		if !name.Valid() {
			t.Errorf("%v %s: route name %q is not a sysurl.Route", methods, path, name)
		}
		seen[name] = true
		return nil
	})
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}

	for _, route := range sysurl.Routes() {
		if !seen[route] {
			t.Errorf("sysurl route %q has no storage handler", route)
		}
	}
}
