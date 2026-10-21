package server

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"

	"mmm-revived/internal/wire"
)

func EncodePayload(info any) (string, error) {
	data, err := wire.Pack(info)
	if err != nil {
		return "", err
	}
	var buffer bytes.Buffer
	writer := zlib.NewWriter(&buffer)
	if _, err = writer.Write(data); err != nil {
		return "", err
	}
	if err = writer.Close(); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(buffer.Bytes()), nil
}
