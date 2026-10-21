package wire

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

const MaxFrameSize = 10 << 20

type Packet struct {
	Method uint16
	Data   []byte
}

type Field struct {
	Number int
	Wire   int
	Value  uint64
	Data   []byte
}

func ReadPacket(r io.Reader) (Packet, error) {
	var header [6]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return Packet{}, err
	}
	n := binary.LittleEndian.Uint32(header[:4])
	if n < 2 || n > MaxFrameSize {
		return Packet{}, fmt.Errorf("invalid frame size %d", n)
	}
	p := Packet{Method: binary.LittleEndian.Uint16(header[4:]), Data: make([]byte, n-2)}
	_, err := io.ReadFull(r, p.Data)
	return p, err
}

func WritePacket(w io.Writer, p Packet) error {
	if len(p.Data) > MaxFrameSize-2 {
		return errors.New("frame too large")
	}
	data := make([]byte, 6, 6+len(p.Data))
	putPacketHeader(data, p)
	_, err := w.Write(append(data, p.Data...))
	return err
}

func WriteBufferedPacket(w io.Writer, p Packet) error {
	if len(p.Data) > MaxFrameSize-2 {
		return errors.New("frame too large")
	}
	var header [6]byte
	putPacketHeader(header[:], p)
	if _, err := w.Write(header[:]); err != nil {
		return err
	}
	_, err := w.Write(p.Data)
	return err
}

func putPacketHeader(dst []byte, p Packet) {
	binary.LittleEndian.PutUint32(dst, uint32(len(p.Data)+2))
	binary.LittleEndian.PutUint16(dst[4:], p.Method)
}

func UintField(dst []byte, number int, value uint64) []byte {
	dst = binary.AppendUvarint(dst, uint64(number<<3))
	return binary.AppendUvarint(dst, value)
}

func BytesField(dst []byte, number int, value []byte) []byte {
	dst = binary.AppendUvarint(dst, uint64(number<<3|2))
	dst = binary.AppendUvarint(dst, uint64(len(value)))
	return append(dst, value...)
}

func ParseFields(data []byte) ([]Field, error) {
	var fields []Field
	for len(data) > 0 {
		tag, n := binary.Uvarint(data)
		if n <= 0 || tag>>3 == 0 {
			return nil, errors.New("invalid protobuf tag")
		}
		data = data[n:]
		f := Field{Number: int(tag >> 3), Wire: int(tag & 7)}
		switch f.Wire {
		case 0:
			if f.Value, n = binary.Uvarint(data); n <= 0 {
				return nil, errors.New("invalid protobuf varint")
			}
		case 1, 5:
			n = 8
			if f.Wire == 5 {
				n = 4
			}
			if n > len(data) {
				return nil, io.ErrUnexpectedEOF
			}
			f.Data = data[:n]
		case 2:
			size, prefix := binary.Uvarint(data)
			if prefix <= 0 || size > uint64(len(data)-prefix) {
				return nil, errors.New("invalid protobuf byte length")
			}
			n = prefix + int(size)
			f.Data = data[prefix:n]
		default:
			return nil, fmt.Errorf("unsupported protobuf wire type %d", f.Wire)
		}
		data = data[n:]
		fields = append(fields, f)
	}
	return fields, nil
}

func FindField(fields []Field, number int) Field {
	for _, f := range fields {
		if f.Number == number {
			return f
		}
	}
	return Field{}
}
