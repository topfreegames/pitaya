package route

import (
	"errors"
	"fmt"
	"strings"

	"github.com/topfreegames/pitaya/logger"
)

var (
	// ErrRouteFieldCantEmpty error
	ErrRouteFieldCantEmpty = errors.New("route field can not empty")
	// ErrInvalidRoute error
	ErrInvalidRoute = errors.New("invalid route")

	log = logger.Log
)

// Route struct
type Route struct {
	SvType  string
	Service string
	Method  string
}

// NewRoute creates a new route
func NewRoute(server, service, method string) *Route {
	return &Route{server, service, method}
}

// String transforms the route into a string
func (r *Route) String() string {
	if r.SvType != "" {
		return fmt.Sprintf("%s.%s.%s", r.SvType, r.Service, r.Method)
	}
	return r.Short()
}

// Short transforms the route into a string without the server type
func (r *Route) Short() string {
	return fmt.Sprintf("%s.%s", r.Service, r.Method)
}

// Decode decodes the route
func Decode(route string) (*Route, error) {
	r := strings.Split(route, ".")
	for _, s := range r {
		if strings.TrimSpace(s) == "" {
			return nil, ErrRouteFieldCantEmpty
		}
	}
	switch len(r) {
	case 3:
		return NewRoute(r[0], r[1], r[2]), nil
	case 2:
		return NewRoute("", r[0], r[1]), nil
	default:
		log.Errorf("invalid route: " + route)
		return nil, ErrInvalidRoute
	}
}
