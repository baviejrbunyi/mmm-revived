package patchserver

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"

	"mmm-revived/internal/patches"
)

func openSeed(path string) (*os.File, int64, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, 0, err
	}
	if !info.Mode().IsRegular() || info.Size() < 1 || info.Size() > patches.MaxFileSize {
		return nil, 0, fmt.Errorf("invalid patch seed %q", path)
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, 0, err
	}
	return file, info.Size(), nil
}

func readSeed(path string) ([]byte, error) {
	file, size, err := openSeed(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var data bytes.Buffer
	data.Grow(int(size) + bytes.MinRead)
	if _, err = data.ReadFrom(io.LimitReader(file, patches.MaxFileSize+1)); err != nil {
		return nil, err
	}
	if int64(data.Len()) != size {
		return nil, fmt.Errorf("patch seed changed while reading %q", path)
	}
	return data.Bytes(), nil
}

func validateUI(data []byte) error {
	signature := []byte("UnityFS\x00")
	if len(data) < 1024 || !bytes.HasPrefix(data, signature) {
		return errors.New("invalid UnityFS UI bundle")
	}
	header := data
	if offset := bytes.Index(data[8:min(128, len(data))], signature); offset >= 0 && offset+8 <= 64 {
		header = data[offset+8:]
	}
	version := binary.BigEndian.Uint32(header[8:12])
	if version < 6 || version > 9 {
		return errors.New("unsupported UnityFS UI bundle version")
	}
	position := 12
	for range 2 {
		end := bytes.IndexByte(header[position:min(position+128, len(header))], 0)
		if end < 1 {
			return errors.New("invalid UnityFS UI bundle version strings")
		}
		position += end + 1
	}
	if len(header)-position < 20 {
		return errors.New("truncated UnityFS UI bundle header")
	}
	size := binary.BigEndian.Uint64(header[position : position+8])
	compressed := binary.BigEndian.Uint32(header[position+8 : position+12])
	uncompressed := binary.BigEndian.Uint32(header[position+12 : position+16])
	if size < uint64(position+20) || size > uint64(len(header)) || compressed == 0 || uint64(compressed) > size || uncompressed == 0 || uncompressed > uint32(patches.MaxFileSize) {
		return errors.New("invalid UnityFS UI bundle metadata sizes")
	}
	return nil
}
