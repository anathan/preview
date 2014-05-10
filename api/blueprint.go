package api

import (
	"github.com/codegangsta/martini"
)

// Blueprint structures represent collections of HTTP handlers that can be configured to hook into martini.
type Blueprint interface {
	// ConfigureMartini configures martini with the HTTP handlers provided by the blueprint.
	ConfigureMartini(m *martini.ClassicMartini) error
}
