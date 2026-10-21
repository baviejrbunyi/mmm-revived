package actor

import (
	"errors"
	"fmt"

	"mmm-revived/internal/wire"
)

type State struct {
	encoder    Encoder
	properties map[any]any
}

func NewState(value any) (*State, error) {
	s := &State{}
	var err error
	s.properties, err = s.encoder.Properties(value)
	return s, err
}

func (s *State) Create(id []byte, class string) ([]byte, error) {
	return create(id, class, s.properties)
}

func (s *State) lookup(path []string) ([]any, error) {
	parent := s.properties
	var property []any
	for i, part := range path {
		key := findKey(parent, part)
		if key == nil {
			return nil, fmt.Errorf("unknown property %v", path)
		}
		property = parent[key].([]any)
		if i+1 < len(path) {
			var ok bool
			if parent, ok = property[2].(map[any]any); !ok {
				return nil, errors.New("invalid dictionary path")
			}
		}
	}
	if property == nil {
		return nil, errors.New("empty synchronization path")
	}
	return property, nil
}

func findKey(properties map[any]any, name string) any {
	if _, ok := properties[name]; ok {
		return name
	}
	for key := range properties {
		if wire.SortKey(key) == name {
			return key
		}
	}
	return nil
}
