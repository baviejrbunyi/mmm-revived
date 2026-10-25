package patchserver

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

const legacyCatalogPath = "resources/legacy_assets.json"

type legacyCatalog struct {
	Packages map[string]legacyPackage `json:"packages"`
	Assets   map[string]string        `json:"assets"`
}

type legacyPackage struct {
	Head string   `json:"head"`
	Deps []string `json:"deps"`
}

type catalogNode struct {
	name     string
	pkg      uint16
	children []uint32
}

func applyLegacyCatalog(modules string, first map[string][]byte) error {
	data, err := os.ReadFile(filepath.Join(modules, filepath.FromSlash(legacyCatalogPath)))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	var catalog legacyCatalog
	if err = json.Unmarshal(data, &catalog); err != nil {
		return err
	}
	if len(catalog.Packages) == 0 {
		return nil
	}
	packages, err := inflate(first["packages.bytes"])
	if err != nil {
		return err
	}
	indices, packages, err := appendLegacyPackages(packages, catalog.Packages)
	if err != nil {
		return err
	}
	assets, err := inflate(first["assets.bytes"])
	if err != nil {
		return err
	}
	if assets, err = appendLegacyAssets(assets, catalog.Assets, indices); err != nil {
		return err
	}
	if first["packages.bytes"], err = deflate(packages); err != nil {
		return err
	}
	first["assets.bytes"], err = deflate(assets)
	return err
}

func inflate(data []byte) ([]byte, error) {
	reader, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return io.ReadAll(io.LimitReader(reader, contentEntryLimit))
}

func deflate(data []byte) ([]byte, error) {
	var output bytes.Buffer
	writer := zlib.NewWriter(&output)
	if _, err := writer.Write(data); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

type catalogReader struct {
	data []byte
	pos  int
	err  error
}

func (r *catalogReader) take(size int) []byte {
	if r.err != nil || size < 0 || r.pos+size > len(r.data) {
		r.err = errors.New("truncated resource catalog")
		return make([]byte, size)
	}
	value := r.data[r.pos : r.pos+size]
	r.pos += size
	return value
}

func (r *catalogReader) u16() uint16 { return binary.LittleEndian.Uint16(r.take(2)) }

func (r *catalogReader) u32() uint32 { return binary.LittleEndian.Uint32(r.take(4)) }

func (r *catalogReader) name() string { return string(r.take(int(r.take(1)[0]))) }

func appendLegacyPackages(data []byte, additions map[string]legacyPackage) (map[string]uint16, []byte, error) {
	reader := &catalogReader{data: data}
	count := reader.u32()
	indices := map[string]uint16{}
	for index := range count {
		name := reader.name()
		reader.take(8)
		reader.take(2 * int(reader.u16()))
		for range reader.u32() {
			reader.name()
			reader.take(4)
		}
		indices[name] = uint16(min(index+1, math.MaxUint16))
	}
	if reader.err != nil || reader.pos != len(data) {
		return nil, nil, errors.New("unsupported packages catalog")
	}
	names := slices.Sorted(maps.Keys(additions))
	for _, name := range names {
		if _, exists := indices[name]; exists {
			return nil, nil, fmt.Errorf("legacy package %s already exists", name)
		}
		if int(count)+1 > math.MaxUint16 {
			return nil, nil, errors.New("packages catalog is full")
		}
		count++
		indices[name] = uint16(count)
	}
	output := bytes.NewBuffer(slices.Clone(data))
	for _, name := range names {
		rule := additions[name]
		head, err := hex.DecodeString(rule.Head)
		if err != nil || len(head) != 8 || len(name) == 0 || len(name) > math.MaxUint8 {
			return nil, nil, fmt.Errorf("invalid legacy package %s", name)
		}
		output.WriteByte(byte(len(name)))
		output.WriteString(name)
		output.Write(head)
		binary.Write(output, binary.LittleEndian, uint16(len(rule.Deps)))
		for _, dep := range rule.Deps {
			index, known := indices[dep]
			if !known {
				return nil, nil, fmt.Errorf("legacy package %s depends on unknown %s", name, dep)
			}
			binary.Write(output, binary.LittleEndian, index)
		}
		binary.Write(output, binary.LittleEndian, uint32(0))
	}
	result := output.Bytes()
	binary.LittleEndian.PutUint32(result, count)
	return indices, result, nil
}

func appendLegacyAssets(data []byte, assets map[string]string, indices map[string]uint16) ([]byte, error) {
	reader := &catalogReader{data: data}
	nodes := make([]catalogNode, reader.u32())
	for index := range nodes {
		node := catalogNode{name: reader.name(), pkg: reader.u16()}
		if node.pkg == 0 {
			node.children = make([]uint32, reader.u32())
			for child := range node.children {
				node.children[child] = reader.u32()
			}
		}
		nodes[index] = node
	}
	if reader.err != nil || reader.pos != len(data) || len(nodes) == 0 {
		return nil, errors.New("unsupported assets catalog")
	}
	paths := slices.Sorted(maps.Keys(assets))
	for _, path := range paths {
		index, known := indices[assets[path]]
		if !known {
			return nil, fmt.Errorf("legacy asset %s uses unknown package %s", path, assets[path])
		}
		parts := strings.Split(path, "/")
		current := uint32(0)
		for position, part := range parts {
			if len(part) == 0 || len(part) > math.MaxUint8 {
				return nil, fmt.Errorf("invalid legacy asset path %s", path)
			}
			next, found := uint32(0), false
			for _, child := range nodes[current].children {
				if nodes[child].name == part {
					next, found = child, true
					break
				}
			}
			last := position == len(parts)-1
			if found {
				if last || nodes[next].pkg != 0 {
					return nil, fmt.Errorf("legacy asset path %s collides with the catalog", path)
				}
				current = next
				continue
			}
			node := catalogNode{name: part}
			if last {
				node.pkg = index
			} else {
				node.children = []uint32{}
			}
			nodes = append(nodes, node)
			created := uint32(len(nodes) - 1)
			nodes[current].children = append(nodes[current].children, created)
			current = created
		}
	}
	var output bytes.Buffer
	binary.Write(&output, binary.LittleEndian, uint32(len(nodes)))
	for _, node := range nodes {
		output.WriteByte(byte(len(node.name)))
		output.WriteString(node.name)
		binary.Write(&output, binary.LittleEndian, node.pkg)
		if node.pkg == 0 {
			binary.Write(&output, binary.LittleEndian, uint32(len(node.children)))
			for _, child := range node.children {
				binary.Write(&output, binary.LittleEndian, child)
			}
		}
	}
	return output.Bytes(), nil
}
