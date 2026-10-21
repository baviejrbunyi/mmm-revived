package protocol

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"errors"
	"io"

	"mmm-revived/internal/wire"
)

func DecodeBatch[T any](c Call, resolve func(Call) (T, error)) ([]T, error) {
	if len(c.Args) != 1 {
		return nil, errors.New("batchRpc requires an operation list")
	}
	values, ok := c.Args[0].([]any)
	if !ok || len(values) == 0 || len(values) > 64 {
		return nil, errors.New("invalid batch operation count")
	}
	result := make([]T, 0, len(values))
	total := 0
	var reader io.ReadCloser
	for _, value := range values {
		encoded, ok := value.(string)
		if !ok || len(encoded) > 256*1024 {
			return nil, errors.New("invalid encoded batch operation")
		}
		compressed, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			return nil, err
		}
		if reader == nil {
			reader, err = zlib.NewReader(bytes.NewReader(compressed))
		} else {
			err = reader.(zlib.Resetter).Reset(bytes.NewReader(compressed), nil)
		}
		if err != nil {
			return nil, err
		}
		data, err := io.ReadAll(io.LimitReader(reader, 1024*1024+1))
		closeError := reader.Close()
		if err != nil {
			return nil, err
		}
		if closeError != nil {
			return nil, closeError
		}
		total += len(data)
		if len(data) > 1024*1024 || total > 4*1024*1024 {
			return nil, errors.New("batch expansion limit exceeded")
		}
		decoded, err := wire.Unpack(data)
		if err != nil {
			return nil, err
		}
		tuple, ok := decoded.([]any)
		if !ok || len(tuple) != 2 {
			return nil, errors.New("invalid batch tuple")
		}
		name, ok := tuple[0].(string)
		if !ok {
			return nil, errors.New("invalid batch method")
		}
		args, ok := tuple[1].([]any)
		if !ok {
			return nil, errors.New("invalid batch arguments")
		}
		operation, err := resolve(Call{Name: name, Args: args})
		if err != nil {
			return nil, err
		}
		result = append(result, operation)
	}
	return result, nil
}
