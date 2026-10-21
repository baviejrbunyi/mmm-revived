package actor

import (
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"

	"mmm-revived/internal/wire"
)

type PropertyType uint8

const (
	Boolean PropertyType = 2
	Integer PropertyType = 3
	Float   PropertyType = 4
	String  PropertyType = 6
	Dict    PropertyType = 15
	List    PropertyType = 16
)

type wholeList PropertyType

type WholeReplaced interface{ ReplacesWhole() }

var wholeReplaced = reflect.TypeFor[WholeReplaced]()

func propertyKind(metadata []any) (PropertyType, bool) {
	switch kind := metadata[1].(type) {
	case wholeList:
		return PropertyType(kind), true
	default:
		return kind.(PropertyType), false
	}
}

type Encoder struct {
	NextID uint64
}

type entry[V any] struct {
	order string
	key   any
	value V
}

func sortEntries[V any](entries []entry[V]) []entry[V] {
	slices.SortFunc(entries, func(a, b entry[V]) int { return strings.Compare(a.order, b.order) })
	return entries
}

func (e *Encoder) Properties(value any) (map[any]any, error) {
	var entries []entry[reflect.Value]
	switch rv := reflect.ValueOf(value); rv.Kind() {
	case reflect.Struct:
		fields, err := wire.SortedStructFields(value)
		if err != nil {
			return nil, err
		}
		entries = make([]entry[reflect.Value], len(fields))
		for i := range fields {
			entries[i] = entry[reflect.Value]{key: fields[i].Name, value: reflect.ValueOf(&fields[i].Value).Elem()}
		}
	case reflect.Map:
		entries = make([]entry[reflect.Value], 0, rv.Len())
		for iter := rv.MapRange(); iter.Next(); {
			key := iter.Key().Interface()
			entries = append(entries, entry[reflect.Value]{wire.SortKey(key), key, iter.Value()})
		}
		sortEntries(entries)
	default:
		return nil, errors.New("properties must be a map or structure")
	}
	result := make(map[any]any, len(entries))
	for _, en := range entries {
		property, err := e.property(en.value)
		if err != nil {
			return nil, fmt.Errorf("property %v: %w", en.key, err)
		}
		result[en.key] = property
	}
	return result, nil
}

func (e *Encoder) property(rv reflect.Value) (any, error) {
	for rv.Kind() == reflect.Interface || rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return nil, errors.New("nil property")
		}
		rv = rv.Elem()
	}
	e.NextID++
	id := e.NextID
	var kind PropertyType
	var value any
	switch rv.Kind() {
	case reflect.Bool:
		kind, value = Boolean, rv.Bool()
	case reflect.String:
		kind, value = String, rv.String()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		kind, value = Integer, rv.Int()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		kind, value = Integer, rv.Uint()
	case reflect.Float32, reflect.Float64:
		kind, value = Float, rv.Float()
	case reflect.Map, reflect.Struct:
		props, err := e.Properties(rv.Interface())
		if err != nil {
			return nil, err
		}
		kind, value = Dict, props
	case reflect.Array, reflect.Slice:
		props := make([]any, rv.Len())
		for i := range props {
			var err error
			if props[i], err = e.property(rv.Index(i)); err != nil {
				return nil, err
			}
		}
		kind, value = List, props
	default:
		return nil, fmt.Errorf("unsupported property type %s", rv.Type())
	}
	cells := make([]any, 5)
	metadata, property := cells[3:5:5], cells[:3:3]
	metadata[0], metadata[1] = id, kind
	if kind == List && rv.Type().Implements(wholeReplaced) {
		metadata[1] = wholeList(kind)
	}
	property[0], property[1], property[2] = metadata, 0, value
	return property, nil
}

func Create(id []byte, class string, properties any) ([]byte, error) {
	var encoder Encoder
	props, err := encoder.Properties(properties)
	if err != nil {
		return nil, err
	}
	return create(id, class, props)
}

func create(id []byte, class string, props map[any]any) ([]byte, error) {
	data, err := wire.Pack(props)
	if err != nil {
		return nil, err
	}
	fields := []any{wire.StringBytes(id), 0, 0, 0, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, class, false, nil, wire.StringBytes(data)}
	inner, err := wire.Append(make([]byte, 0, len(id)+len(class)+len(data)+32), []any{fields})
	if err != nil {
		return nil, err
	}
	return Envelope(43, inner)
}

func Envelope(kind int, payload []byte) ([]byte, error) {
	return wire.Append(make([]byte, 0, len(payload)+16), []any{[]any{0, kind, wire.StringBytes(payload), ""}})
}

func RPC(method string, args ...any) ([]byte, error) {
	inner, err := wire.Pack([]any{[]any{0, method, args, nil}})
	if err != nil {
		return nil, err
	}
	return Envelope(1, inner)
}
