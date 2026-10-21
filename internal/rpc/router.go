package rpc

import (
	"fmt"
	"iter"
	"maps"
	"slices"

	"mmm-revived/internal/protocol"
)

type Handler[S any] func(S) error

type Endpoint[S any] func(protocol.Call) (Handler[S], error)

type Route[S any] struct {
	Endpoint Endpoint[S]
	Public   bool
}

type Router[S any] struct {
	routes   map[string]Route[S]
	fallback Endpoint[S]
}

func NewRouter[S any](fallback Endpoint[S]) *Router[S] {
	return &Router[S]{routes: map[string]Route[S]{}, fallback: fallback}
}

func (r *Router[S]) Handle(name string, endpoint Endpoint[S]) {
	r.add(name, Route[S]{Endpoint: endpoint})
}

func (r *Router[S]) Public(name string, endpoint Endpoint[S]) {
	r.add(name, Route[S]{Endpoint: endpoint, Public: true})
}

func (r *Router[S]) HandleAll(endpoint Endpoint[S], names ...string) {
	for _, name := range names {
		r.Handle(name, endpoint)
	}
}

func (r *Router[S]) add(name string, route Route[S]) {
	if _, exists := r.routes[name]; exists {
		panic(fmt.Sprintf("rpc: duplicate route %q", name))
	}
	r.routes[name] = route
}

func (r *Router[S]) Route(name string) (Route[S], bool) {
	route, ok := r.routes[name]
	if !ok && r.fallback != nil {
		return Route[S]{Endpoint: r.fallback}, false
	}
	return route, ok
}

func (r *Router[S]) Resolve(call protocol.Call) (Handler[S], bool, error) {
	route, _ := r.Route(call.Name)
	if route.Endpoint == nil {
		return nil, false, fmt.Errorf("unhandled RPC %s", call.Name)
	}
	handler, err := route.Endpoint(call)
	return handler, route.Public, err
}

func (r *Router[S]) Names() iter.Seq[string] {
	return slices.Values(slices.Sorted(maps.Keys(r.routes)))
}

func Bind[S, T any](decode func(protocol.Call) (T, error), handle func(S, T) error) Endpoint[S] {
	return func(call protocol.Call) (Handler[S], error) {
		request, err := decode(call)
		if err != nil {
			return nil, err
		}
		return func(state S) error { return handle(state, request) }, nil
	}
}

func Bare[S any](handle func(S) error) Endpoint[S] {
	handler := Handler[S](handle)
	return func(protocol.Call) (Handler[S], error) { return handler, nil }
}
