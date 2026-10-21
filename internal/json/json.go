package json

import (
	standard "encoding/json"
	"io"

	"github.com/bytedance/sonic"
)

type RawMessage = standard.RawMessage
type Decoder = sonic.Decoder

func Marshal(value any) ([]byte, error) {
	return sonic.ConfigStd.Marshal(value)
}

func MarshalIndent(value any, prefix, indent string) ([]byte, error) {
	return sonic.ConfigStd.MarshalIndent(value, prefix, indent)
}

func Unmarshal(data []byte, value any) error {
	return sonic.ConfigStd.Unmarshal(data, value)
}

func NewDecoder(reader io.Reader) Decoder {
	return sonic.ConfigStd.NewDecoder(reader)
}
