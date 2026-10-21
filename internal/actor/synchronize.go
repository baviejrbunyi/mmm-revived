package actor

import (
	"errors"
	"reflect"
	"slices"

	"mmm-revived/internal/wire"
)

func (s *State) Synchronize(path []string, value any) ([][]byte, error) {
	previous, err := s.lookup(path)
	if err != nil {
		return nil, err
	}
	next, err := s.encoder.property(reflect.ValueOf(value))
	if err != nil {
		return nil, err
	}
	operations, err := diff(nil, previous, next.([]any))
	if err != nil {
		return nil, err
	}
	payloads := make([][]byte, len(operations))
	for i, operation := range operations {
		inner, err := wire.Pack([]any{operation})
		if err != nil {
			return nil, err
		}
		if payloads[i], err = Envelope(2, inner); err != nil {
			return nil, err
		}
	}
	return payloads, nil
}

func change(kind PropertyType, value any) []any {
	if kind == String || kind == Dict || kind == List {
		value = []any{1, value}
	}
	return []any{kind, value}
}

func changed(property []any) []any {
	kind, _ := propertyKind(property[0].([]any))
	return change(kind, property[2])
}

func sortedProperties(properties map[any]any) []entry[any] {
	entries := make([]entry[any], 0, len(properties))
	for key, value := range properties {
		entries = append(entries, entry[any]{wire.SortKey(key), key, value})
	}
	return sortEntries(entries)
}

func diff(operations [][]any, old, next []any) ([][]any, error) {
	metadata := old[0].([]any)
	if metadata[1] != next[0].([]any)[1] {
		return nil, errors.New("property type changed")
	}
	id := metadata[0]
	kind, whole := propertyKind(metadata)
	switch kind {
	case Dict:
		before, after := old[2].(map[any]any), next[2].(map[any]any)
		for _, e := range sortedProperties(before) {
			if _, ok := after[e.key]; !ok {
				operations = append(operations, []any{4, id, 0, e.value.([]any)[0].([]any)[0]})
				delete(before, e.key)
			}
		}
		for _, e := range sortedProperties(after) {
			property := e.value.([]any)
			if previous, ok := before[e.key]; ok {
				var err error
				if operations, err = diff(operations, previous.([]any), property); err != nil {
					return nil, err
				}
			} else {
				operations = append(operations, []any{3, id, 0, e.key, property[0], changed(property)})
				before[e.key] = property
			}
		}
	case List:
		before, after := old[2].([]any), next[2].([]any)
		if equal(before, after) {
			return operations, nil
		}
		if whole {
			old[2] = after
			return append(operations, []any{1, id, 0, changed(next)}), nil
		}
		for index, value := range after[:min(len(before), len(after))] {
			property := value.([]any)
			if !equal(before[index].([]any)[2], property[2]) {
				operations = append(operations, []any{8, id, 0, index, property[0], changed(property)})
				before[index] = property
			}
		}
		if len(after) > len(before) {
			operations = append(operations, []any{10, id, 0, []int{len(before)}, change(List, after[len(before):])})
			before = append(before, after[len(before):]...)
		} else {
			for index := len(before) - 1; index >= len(after); index-- {
				operations = append(operations, []any{9, id, 0, []int{index}})
			}
			before = before[:len(after)]
		}
		old[2] = before
	default:
		if !equal(old[2], next[2]) {
			operations = append(operations, []any{1, id, 0, changed(next)})
			old[2] = next[2]
		}
	}
	return operations, nil
}

func equal(a, b any) bool {
	switch a := a.(type) {
	case map[any]any:
		b, ok := b.(map[any]any)
		if !ok || len(a) != len(b) {
			return false
		}
		for key, x := range a {
			y, ok := b[key]
			if !ok || !equal(x.([]any)[2], y.([]any)[2]) {
				return false
			}
		}
		return true
	case []any:
		b, ok := b.([]any)
		return ok && slices.EqualFunc(a, b, func(x, y any) bool { return equal(x.([]any)[2], y.([]any)[2]) })
	}
	return a == b
}
