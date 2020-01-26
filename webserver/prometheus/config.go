package prometheus

import (
	"net/http"

	"yanic/lib/duration"

	"yanic/respond"
	"yanic/runtime"
)

type Config struct {
	Enable   bool              `toml:"enable"`
	Wait     duration.Duration `toml:"wait"`
	Outdated duration.Duration `toml:"outdated"`
}

func CreateExporter(config Config, srv *http.Server, coll *respond.Collector, nodes *runtime.Nodes) {
	mux := http.NewServeMux()
	ex := &Exporter{
		config: config,
		srv:    srv,
		coll:   coll,
		nodes:  nodes,
	}
	mux.Handle("/metric", ex)
	if srv.Handler != nil {
		mux.Handle("/", srv.Handler)
	}
	srv.Handler = mux
}
