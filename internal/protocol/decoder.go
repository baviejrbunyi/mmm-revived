package protocol

import (
	"errors"
	"fmt"

	"mmm-revived/internal/wire"
)

const (
	rpcMessage       = 25
	methodCacheLimit = 4096
)

type Decoder struct {
	methods map[int64]string
}

func (d *Decoder) Decode(payload []byte) ([]Call, error) {
	var calls []Call
	for len(payload) > 0 {
		value, rest, err := wire.UnpackOne(payload)
		if err != nil {
			return nil, err
		}
		payload = rest
		outer, ok := value.([]any)
		if !ok || len(outer) < 3 {
			return nil, errors.New("invalid client envelope")
		}
		if kind, err := Integer(outer[1]); err != nil || kind != rpcMessage {
			return nil, errors.New("unsupported client message type")
		}
		call, err := d.call(outer[2])
		if err != nil {
			return nil, err
		}
		calls = append(calls, call)
	}
	return calls, nil
}

func (d *Decoder) call(envelope any) (Call, error) {
	packed, err := binaryValue(envelope)
	if err != nil {
		return Call{}, err
	}
	value, err := wire.Unpack(packed)
	if err != nil {
		return Call{}, err
	}
	wrapper, ok := value.([]any)
	if !ok || len(wrapper) != 1 {
		return Call{}, errors.New("invalid RPC wrapper")
	}
	inner, ok := wrapper[0].([]any)
	if !ok || len(inner) < 3 {
		return Call{}, errors.New("invalid RPC tuple")
	}
	encoding, err := Integer(inner[0])
	if err != nil {
		return Call{}, err
	}
	var name string
	var raw any
	switch encoding {
	case 0:
		name, ok = inner[1].(string)
		raw = inner[2]
	case 1:
		if len(inner) < 4 {
			return Call{}, errors.New("invalid method registration")
		}
		name, ok = inner[1].(string)
		index, err := Integer(inner[2])
		if err != nil || index < 0 || index > 65535 {
			return Call{}, errors.New("invalid method index")
		}
		if d.methods == nil {
			d.methods = map[int64]string{}
		}
		if len(d.methods) >= methodCacheLimit {
			return Call{}, errors.New("method cache limit exceeded")
		}
		d.methods[index] = name
		raw = inner[3]
	case 2:
		index, err := Integer(inner[1])
		if err != nil {
			return Call{}, err
		}
		name, ok = d.methods[index]
		raw = inner[2]
	default:
		return Call{}, fmt.Errorf("unsupported RPC encoding %d", encoding)
	}
	if !ok || name == "" || len(name) > 128 {
		return Call{}, errors.New("invalid RPC name")
	}
	args, err := decodeArguments(raw)
	return Call{Name: name, Args: args}, err
}

func decodeArguments(raw any) ([]any, error) {
	packed, err := binaryValue(raw)
	if err != nil || len(packed) == 0 {
		return []any{}, err
	}
	value, err := wire.Unpack(packed)
	if err != nil {
		return nil, err
	}
	switch decoded := value.(type) {
	case []any:
		return decoded, nil
	case map[any]any:
		if len(decoded) != 0 {
			return nil, errors.New("RPC arguments are not an array")
		}
	case nil:
	default:
		return nil, errors.New("RPC arguments are not an array")
	}
	return []any{}, nil
}
