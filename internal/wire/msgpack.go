package wire

import (
	"encoding/binary"
	"errors"
	"fmt"
	"maps"
	"math"
	"reflect"
	"slices"
	"strconv"
	"strings"
)

const maxDepth = 128

var (
	errDepth     = errors.New("message nesting limit exceeded")
	errTruncated = errors.New("truncated MessagePack value")
	errSize      = errors.New("invalid MessagePack size")

	valueType  = reflect.TypeFor[Value]()
	boolType   = reflect.TypeFor[bool]()
	stringType = reflect.TypeFor[string]()
	bytesType  = reflect.TypeFor[[]byte]()
	f32Type    = reflect.TypeFor[float32]()
	f64Type    = reflect.TypeFor[float64]()
)

type Value interface{ WireValue() any }

type StringBytes []byte

func Pack(value any) ([]byte, error) {
	return appendValue(nil, value, 0)
}

func Append(dst []byte, value any) ([]byte, error) {
	out, err := appendValue(dst, value, 0)
	if err != nil {
		return dst, err
	}
	return out, nil
}

func SortKey(key any) string {
	switch k := key.(type) {
	case nil:
		return "<nil>"
	case string:
		return k
	case int:
		return strconv.Itoa(k)
	case int64:
		return strconv.FormatInt(k, 10)
	case uint64:
		return strconv.FormatUint(k, 10)
	case bool:
		return strconv.FormatBool(k)
	}
	return fmt.Sprint(key)
}

func sortKey(key reflect.Value) string {
	if key.Type().NumMethod() == 0 {
		switch key.Kind() {
		case reflect.String:
			return key.String()
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return strconv.FormatInt(key.Int(), 10)
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
			return strconv.FormatUint(key.Uint(), 10)
		case reflect.Interface:
			return SortKey(key.Interface())
		}
	}
	return fmt.Sprint(key)
}

func appendValue(dst []byte, value any, depth int) ([]byte, error) {
	if depth > maxDepth {
		return nil, errDepth
	}
	switch v := value.(type) {
	case nil:
		return append(dst, 0xc0), nil
	case bool:
		return appendBool(dst, v), nil
	case string:
		return appendString(dst, v), nil
	case StringBytes:
		return appendString(dst, v), nil
	case []byte:
		return appendBinary(dst, v), nil
	case float64:
		return appendFloat64(dst, v), nil
	case float32:
		return appendFloat32(dst, v), nil
	case int:
		return appendInt(dst, int64(v)), nil
	case int64:
		return appendInt(dst, v), nil
	case uint64:
		return appendUint(dst, v), nil
	case []any:
		dst = appendArrayHeader(dst, len(v))
		for _, item := range v {
			var err error
			if dst, err = appendValue(dst, item, depth+1); err != nil {
				return nil, err
			}
		}
		return dst, nil
	case map[string]any:
		dst = appendMapHeader(dst, len(v))
		if len(v) > 0 && depth+1 > maxDepth {
			return nil, errDepth
		}
		for _, key := range slices.Sorted(maps.Keys(v)) {
			var err error
			if dst, err = appendValue(appendString(dst, key), v[key], depth+1); err != nil {
				return nil, err
			}
		}
		return dst, nil
	case map[any]any:
		return appendAnyMap(dst, v, depth)
	case Value:
		if rv := reflect.ValueOf(v); rv.Kind() == reflect.Pointer && rv.IsNil() {
			return append(dst, 0xc0), nil
		}
		return appendValue(dst, v.WireValue(), depth+1)
	}
	return appendReflect(dst, reflect.ValueOf(value), depth)
}

type anyEntry struct {
	order string
	key   any
	value any
}

func appendAnyMap(dst []byte, m map[any]any, depth int) ([]byte, error) {
	dst = appendMapHeader(dst, len(m))
	entries := make([]anyEntry, 0, len(m))
	for key, value := range m {
		entries = append(entries, anyEntry{SortKey(key), key, value})
	}
	slices.SortFunc(entries, func(a, b anyEntry) int { return strings.Compare(a.order, b.order) })
	for _, e := range entries {
		var err error
		if dst, err = appendValue(dst, e.key, depth+1); err != nil {
			return nil, err
		}
		if dst, err = appendValue(dst, e.value, depth+1); err != nil {
			return nil, err
		}
	}
	return dst, nil
}

func appendReflect(dst []byte, rv reflect.Value, depth int) ([]byte, error) {
	if depth > maxDepth {
		return nil, errDepth
	}
	if rv.Kind() == reflect.Interface {
		return appendValue(dst, rv.Interface(), depth)
	}
	t := rv.Type()
	if t.Implements(valueType) {
		if rv.Kind() == reflect.Pointer && rv.IsNil() {
			return append(dst, 0xc0), nil
		}
		return appendValue(dst, rv.Interface().(Value).WireValue(), depth+1)
	}
	switch rv.Kind() {
	case reflect.Bool:
		if t == boolType {
			return appendBool(dst, rv.Bool()), nil
		}
	case reflect.Float64:
		if t == f64Type {
			return appendFloat64(dst, rv.Float()), nil
		}
	case reflect.Float32:
		if t == f32Type {
			return appendFloat32(dst, float32(rv.Float())), nil
		}
	case reflect.String:
		if t != stringType && depth+1 > maxDepth {
			return nil, errDepth
		}
		return appendString(dst, rv.String()), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return appendInt(dst, rv.Int()), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return appendUint(dst, rv.Uint()), nil
	case reflect.Pointer:
		if rv.IsNil() {
			return append(dst, 0xc0), nil
		}
		return appendReflect(dst, rv.Elem(), depth+1)
	case reflect.Struct:
		return appendStruct(dst, rv, depth)
	case reflect.Slice, reflect.Array:
		if t == bytesType {
			return appendBinary(dst, rv.Bytes()), nil
		}
		n := rv.Len()
		dst = appendArrayHeader(dst, n)
		for i := range n {
			var err error
			if dst, err = appendReflect(dst, rv.Index(i), depth+1); err != nil {
				return nil, err
			}
		}
		return dst, nil
	case reflect.Map:
		return appendMap(dst, rv, depth)
	}
	return nil, fmt.Errorf("unsupported MessagePack type %s", t)
}

type reflectEntry struct {
	order string
	index int
}

func appendMap(dst []byte, rv reflect.Value, depth int) ([]byte, error) {
	n := rv.Len()
	dst = appendMapHeader(dst, n)
	if n == 0 {
		return dst, nil
	}
	t := rv.Type()
	keys := reflect.MakeSlice(reflect.SliceOf(t.Key()), n, n)
	values := reflect.MakeSlice(reflect.SliceOf(t.Elem()), n, n)
	entries := make([]reflectEntry, n)
	iter := rv.MapRange()
	for i := 0; iter.Next(); i++ {
		key := keys.Index(i)
		key.SetIterKey(iter)
		values.Index(i).SetIterValue(iter)
		entries[i] = reflectEntry{sortKey(key), i}
	}
	slices.SortFunc(entries, func(a, b reflectEntry) int { return strings.Compare(a.order, b.order) })
	for _, e := range entries {
		var err error
		if dst, err = appendReflect(dst, keys.Index(e.index), depth+1); err != nil {
			return nil, err
		}
		if dst, err = appendReflect(dst, values.Index(e.index), depth+1); err != nil {
			return nil, err
		}
	}
	return dst, nil
}

func appendStruct(dst []byte, rv reflect.Value, depth int) ([]byte, error) {
	info := structInfoOf(rv.Type())
	if err := info.checkDuplicates(rv); err != nil {
		return nil, err
	}
	if depth+1 > maxDepth {
		return nil, errDepth
	}
	var buffer [32]int
	included := buffer[:0]
	for i, f := range info.sorted {
		if f.included(rv) {
			included = append(included, i)
		}
	}
	dst = appendMapHeader(dst, len(included))
	if len(included) > 0 && depth+2 > maxDepth {
		return nil, errDepth
	}
	for _, i := range included {
		f := info.sorted[i]
		var err error
		if dst, err = appendValue(appendString(dst, f.name), f.value(rv), depth+2); err != nil {
			return nil, err
		}
	}
	return dst, nil
}

func appendBool(dst []byte, v bool) []byte {
	if v {
		return append(dst, 0xc3)
	}
	return append(dst, 0xc2)
}

func appendFloat64(dst []byte, v float64) []byte {
	return binary.BigEndian.AppendUint64(append(dst, 0xcb), math.Float64bits(v))
}

func appendFloat32(dst []byte, v float32) []byte {
	return binary.BigEndian.AppendUint32(append(dst, 0xca), math.Float32bits(v))
}

func appendString[T string | []byte | StringBytes](dst []byte, s T) []byte {
	switch n := len(s); {
	case n < 32:
		dst = append(dst, 0xa0|byte(n))
	case n <= math.MaxUint8:
		dst = append(dst, 0xd9, byte(n))
	case n <= math.MaxUint16:
		dst = binary.BigEndian.AppendUint16(append(dst, 0xda), uint16(n))
	default:
		dst = binary.BigEndian.AppendUint32(append(dst, 0xdb), uint32(n))
	}
	return append(dst, s...)
}

func appendBinary(dst, b []byte) []byte {
	switch n := len(b); {
	case n <= math.MaxUint8:
		dst = append(dst, 0xc4, byte(n))
	case n <= math.MaxUint16:
		dst = binary.BigEndian.AppendUint16(append(dst, 0xc5), uint16(n))
	default:
		dst = binary.BigEndian.AppendUint32(append(dst, 0xc6), uint32(n))
	}
	return append(dst, b...)
}

func appendArrayHeader(dst []byte, n int) []byte {
	return appendCollectionHeader(dst, n, 0x90, 0xdc)
}

func appendMapHeader(dst []byte, n int) []byte {
	return appendCollectionHeader(dst, n, 0x80, 0xde)
}

func appendCollectionHeader(dst []byte, n int, fix, tag byte) []byte {
	switch {
	case n < 16:
		return append(dst, fix|byte(n))
	case n <= math.MaxUint16:
		return binary.BigEndian.AppendUint16(append(dst, tag), uint16(n))
	default:
		return binary.BigEndian.AppendUint32(append(dst, tag+1), uint32(n))
	}
}

func appendInt(dst []byte, v int64) []byte {
	switch {
	case v >= 0:
		return appendUint(dst, uint64(v))
	case v >= -32:
		return append(dst, byte(v))
	case v >= math.MinInt8:
		return append(dst, 0xd0, byte(v))
	case v >= math.MinInt16:
		return binary.BigEndian.AppendUint16(append(dst, 0xd1), uint16(v))
	case v >= math.MinInt32:
		return binary.BigEndian.AppendUint32(append(dst, 0xd2), uint32(v))
	default:
		return binary.BigEndian.AppendUint64(append(dst, 0xd3), uint64(v))
	}
}

func appendUint(dst []byte, v uint64) []byte {
	switch {
	case v < 128:
		return append(dst, byte(v))
	case v <= math.MaxUint8:
		return append(dst, 0xcc, byte(v))
	case v <= math.MaxUint16:
		return binary.BigEndian.AppendUint16(append(dst, 0xcd), uint16(v))
	case v <= math.MaxUint32:
		return binary.BigEndian.AppendUint32(append(dst, 0xce), uint32(v))
	default:
		return binary.BigEndian.AppendUint64(append(dst, 0xcf), v)
	}
}

func Unpack(data []byte) (any, error) {
	v, rest, err := UnpackOne(data)
	if err == nil && len(rest) != 0 {
		return nil, errors.New("trailing MessagePack data")
	}
	return v, err
}

func UnpackOne(data []byte) (any, []byte, error) {
	d := decoder{data}
	v, err := d.value(0)
	return v, d.data, err
}

type decoder struct {
	data []byte
}

func (d *decoder) take(n uint64) ([]byte, error) {
	if n > uint64(len(d.data)) {
		return nil, errTruncated
	}
	v := d.data[:n]
	d.data = d.data[n:]
	return v, nil
}

func (d *decoder) number(n uint64) (uint64, error) {
	b, err := d.take(n)
	if err != nil {
		return 0, err
	}
	switch n {
	case 1:
		return uint64(b[0]), nil
	case 2:
		return uint64(binary.BigEndian.Uint16(b)), nil
	case 4:
		return uint64(binary.BigEndian.Uint32(b)), nil
	}
	return binary.BigEndian.Uint64(b), nil
}

func (d *decoder) check(size uint64) error {
	if size > MaxFrameSize || size > uint64(len(d.data)) {
		return errSize
	}
	return nil
}

func (d *decoder) value(depth int) (any, error) {
	if depth > maxDepth {
		return nil, errDepth
	}
	if len(d.data) == 0 {
		return nil, errTruncated
	}
	tag := d.data[0]
	d.data = d.data[1:]
	switch {
	case tag < 0x80:
		return int64(tag), nil
	case tag >= 0xe0:
		return int64(int8(tag)), nil
	case tag&0xe0 == 0xa0:
		return d.string(uint64(tag & 0x1f))
	case tag&0xf0 == 0x90:
		return d.array(uint64(tag&0x0f), depth)
	case tag&0xf0 == 0x80:
		return d.mapping(uint64(tag&0x0f), depth)
	}
	switch tag {
	case 0xc0:
		return nil, nil
	case 0xc2, 0xc3:
		return tag == 0xc3, nil
	case 0xcc, 0xcd, 0xce, 0xcf:
		v, err := d.number(1 << (tag - 0xcc))
		if v <= math.MaxInt64 {
			return int64(v), err
		}
		return v, err
	case 0xd0, 0xd1, 0xd2, 0xd3:
		n := uint64(1) << (tag - 0xd0)
		v, err := d.number(n)
		shift := 64 - n*8
		return int64(v<<shift) >> shift, err
	case 0xca:
		v, err := d.number(4)
		return float64(math.Float32frombits(uint32(v))), err
	case 0xcb:
		v, err := d.number(8)
		return math.Float64frombits(v), err
	case 0xd9, 0xda, 0xdb:
		size, err := d.number(1 << (tag - 0xd9))
		if err != nil {
			return nil, err
		}
		return d.string(size)
	case 0xc4, 0xc5, 0xc6:
		size, err := d.number(1 << (tag - 0xc4))
		if err != nil {
			return nil, err
		}
		return d.binary(size)
	case 0xdc, 0xdd:
		size, err := d.number(2 << (tag - 0xdc))
		if err != nil {
			return nil, err
		}
		return d.array(size, depth)
	case 0xde, 0xdf:
		size, err := d.number(2 << (tag - 0xde))
		if err != nil {
			return nil, err
		}
		return d.mapping(size, depth)
	}
	return nil, fmt.Errorf("unsupported MessagePack tag %#x", tag)
}

func (d *decoder) string(size uint64) (any, error) {
	if err := d.check(size); err != nil {
		return nil, err
	}
	v, err := d.take(size)
	return string(v), err
}

func (d *decoder) binary(size uint64) (any, error) {
	if err := d.check(size); err != nil {
		return nil, err
	}
	v, err := d.take(size)
	return v, err
}

func (d *decoder) array(size uint64, depth int) (any, error) {
	if err := d.check(size); err != nil {
		return nil, err
	}
	v := make([]any, size)
	for i := range v {
		var err error
		if v[i], err = d.value(depth + 1); err != nil {
			return nil, err
		}
	}
	return v, nil
}

func (d *decoder) mapping(size uint64, depth int) (any, error) {
	if err := d.check(size); err != nil {
		return nil, err
	}
	v := make(map[any]any, size)
	for range size {
		key, err := d.value(depth + 1)
		if err != nil {
			return nil, err
		}
		switch k := key.(type) {
		case []byte, []any, map[any]any:
			return nil, errors.New("unhashable MessagePack key")
		case float64:
			if math.IsNaN(k) {
				return nil, errors.New("NaN MessagePack map key")
			}
		}
		if v[key], err = d.value(depth + 1); err != nil {
			return nil, err
		}
	}
	return v, nil
}
