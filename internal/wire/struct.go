package wire

import (
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"sync"

	"mmm-revived/internal/calendar"
)

type FieldValue struct {
	Name  string
	Value any
}

type structField struct {
	name      string
	index     int
	omitEmpty bool
	shift     bool
}

func (f structField) included(rv reflect.Value) bool {
	return !f.omitEmpty || !rv.Field(f.index).IsZero()
}

func (f structField) value(rv reflect.Value) any {
	value := rv.Field(f.index).Interface()
	if f.shift {
		return calendar.Shift(value)
	}
	return value
}

type structInfo struct {
	fields    []structField
	sorted    []structField
	duplicate bool
}

var structCache sync.Map

func structInfoOf(t reflect.Type) *structInfo {
	if info, ok := structCache.Load(t); ok {
		return info.(*structInfo)
	}
	info := &structInfo{}
	names := make(map[string]bool)
	for field := range t.Fields() {
		tag := field.Tag.Get("wire")
		name, _, _ := strings.Cut(tag, ",")
		if name == "" || name == "-" || !field.IsExported() {
			continue
		}
		info.duplicate = info.duplicate || names[name]
		names[name] = true
		info.fields = append(info.fields, structField{name, field.Index[0], strings.Contains(","+tag+",", ",omitempty,"), calendar.IsTimeField(name)})
	}
	info.sorted = slices.SortedStableFunc(slices.Values(info.fields), func(a, b structField) int { return strings.Compare(a.name, b.name) })
	actual, _ := structCache.LoadOrStore(t, info)
	return actual.(*structInfo)
}

func (info *structInfo) checkDuplicates(rv reflect.Value) error {
	if !info.duplicate {
		return nil
	}
	seen := make(map[string]bool, len(info.fields))
	for _, f := range info.fields {
		if !f.included(rv) {
			continue
		}
		if seen[f.name] {
			return fmt.Errorf("duplicate wire field %s", f.name)
		}
		seen[f.name] = true
	}
	return nil
}

func structValue(value any) (reflect.Value, *structInfo, error) {
	rv := reflect.ValueOf(value)
	for rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return rv, nil, errors.New("nil structure")
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return rv, nil, fmt.Errorf("expected structure, got %T", value)
	}
	info := structInfoOf(rv.Type())
	return rv, info, info.checkDuplicates(rv)
}

func StructFields(value any) (map[string]any, error) {
	rv, info, err := structValue(value)
	if err != nil {
		return nil, err
	}
	result := make(map[string]any, len(info.fields))
	for _, f := range info.fields {
		if f.included(rv) {
			result[f.name] = f.value(rv)
		}
	}
	return result, nil
}

func SortedStructFields(value any) ([]FieldValue, error) {
	rv, info, err := structValue(value)
	if err != nil {
		return nil, err
	}
	result := make([]FieldValue, 0, len(info.sorted))
	for _, f := range info.sorted {
		if f.included(rv) {
			result = append(result, FieldValue{f.name, f.value(rv)})
		}
	}
	return result, nil
}
