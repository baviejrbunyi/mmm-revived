package main

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"mmm-revived/internal/config"
)

type fingerprintInput struct {
	Name string
	Path string
}

func (w workflow) fingerprint(c config.Config) (string, error) {
	inputs := []fingerprintInput{
		{Name: "go.mod", Path: filepath.Join(w.root, "go.mod")},
		{Name: "go.sum", Path: filepath.Join(w.root, "go.sum")},
		{Name: "justfile", Path: filepath.Join(w.root, "justfile")},
		{Name: "@patch-manifest", Path: filepath.Join(c.Patches(), "patchlist_win32.json")},
		{Name: "@public-key", Path: c.PublicKey()},
	}
	for _, directory := range []string{"cmd", "internal", "tools", "assets"} {
		absolute := filepath.Join(w.root, directory)
		info, err := os.Lstat(absolute)
		if err != nil {
			return "", fmt.Errorf("fingerprint %s: %w", directory, err)
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("fingerprint requires a real directory: %s", directory)
		}
		err = filepath.WalkDir(absolute, func(name string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.Type()&os.ModeSymlink != 0 {
				return fmt.Errorf("fingerprint input contains a symbolic link: %s", name)
			}
			if entry.IsDir() {
				return nil
			}
			if directory != "assets" && (!strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go")) {
				return nil
			}
			if !entry.Type().IsRegular() {
				return fmt.Errorf("fingerprint input is not a regular file: %s", name)
			}
			relative, err := filepath.Rel(w.root, name)
			if err != nil {
				return err
			}
			inputs = append(inputs, fingerprintInput{Name: filepath.ToSlash(relative), Path: name})
			return nil
		})
		if err != nil {
			return "", fmt.Errorf("fingerprint %s: %w", directory, err)
		}
	}
	slices.SortFunc(inputs, func(first, second fingerprintInput) int { return strings.Compare(first.Name, second.Name) })
	digest := sha256.New()
	fingerprintFrame(digest, "@schema", "mmm-revived-build-1")
	fingerprintFrame(digest, "@go-version", runtime.Version())
	fingerprintFrame(digest, "@goos", runtime.GOOS)
	fingerprintFrame(digest, "@goarch", runtime.GOARCH)
	for _, input := range inputs {
		info, err := os.Lstat(input.Path)
		if err != nil {
			return "", fmt.Errorf("fingerprint %s: %w", input.Name, err)
		}
		if !info.Mode().IsRegular() {
			return "", fmt.Errorf("fingerprint input is not a regular file: %s", input.Name)
		}
		file, err := os.Open(input.Path)
		if err != nil {
			return "", fmt.Errorf("fingerprint %s: %w", input.Name, err)
		}
		fileDigest := sha256.New()
		size, readErr := io.Copy(fileDigest, io.LimitReader(file, info.Size()+1))
		after, statErr := file.Stat()
		closeErr := file.Close()
		if readErr != nil {
			return "", fmt.Errorf("fingerprint %s: %w", input.Name, readErr)
		}
		if statErr != nil {
			return "", fmt.Errorf("fingerprint %s: %w", input.Name, statErr)
		}
		if closeErr != nil {
			return "", fmt.Errorf("fingerprint %s: %w", input.Name, closeErr)
		}
		if size != info.Size() || !info.ModTime().Equal(after.ModTime()) {
			return "", fmt.Errorf("fingerprint input changed while reading: %s", input.Name)
		}
		fingerprintFrame(digest, input.Name, hex.EncodeToString(fileDigest.Sum(nil)))
	}
	return hex.EncodeToString(digest.Sum(nil)), nil
}

func fingerprintFrame(destination io.Writer, name, value string) {
	var size [8]byte
	binary.BigEndian.PutUint64(size[:], uint64(len(name)))
	destination.Write(size[:])
	io.WriteString(destination, name)
	binary.BigEndian.PutUint64(size[:], uint64(len(value)))
	destination.Write(size[:])
	io.WriteString(destination, value)
}
