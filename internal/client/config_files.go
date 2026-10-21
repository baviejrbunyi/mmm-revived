package client

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
)

func configSafePath(root *os.Root, name string, missing, directory bool) error {
	parts := strings.Split(name, "/")
	end := -1
	for index, part := range parts {
		end += len(part) + 1
		info, err := root.Lstat(name[:end])
		if errors.Is(err, os.ErrNotExist) && missing {
			return nil
		}
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("symbolic links are not allowed in update paths: %s", name)
		}
		if index < len(parts)-1 || directory {
			if !info.IsDir() {
				return fmt.Errorf("update parent is not a directory: %s", name)
			}
		} else if !info.Mode().IsRegular() {
			return fmt.Errorf("update target is not a regular file: %s", name)
		}
	}
	return nil
}

func readConfigFile(root *os.Root, name string, limit int64) ([]byte, error) {
	if err := configSafePath(root, name, true, false); err != nil {
		return nil, err
	}
	file, err := root.Open(name)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, errors.New("update metadata exceeds size limit")
	}
	return data, nil
}

func writeConfigBackup(root *os.Root, name string, data []byte) error {
	parent := path.Dir(name)
	if err := configSafePath(root, parent, true, true); err != nil {
		return err
	}
	if err := root.MkdirAll(parent, 0700); err != nil {
		return err
	}
	if err := configSafePath(root, name, true, false); err != nil {
		return err
	}
	file, err := root.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	defer file.Close()
	if _, err = file.Write(data); err != nil {
		return err
	}
	return file.Sync()
}

func replaceConfigFile(root *os.Root, name string, input io.Reader, digest string, size int64, mode os.FileMode, token string) error {
	if err := configSafePath(root, name, true, false); err != nil {
		return err
	}
	if err := root.MkdirAll(path.Dir(name), 0700); err != nil {
		return err
	}
	temporary := configTemporaryPath(name, token)
	if err := configSafePath(root, temporary, true, false); err != nil {
		return err
	}
	file, err := root.OpenFile(temporary, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	defer root.Remove(temporary)
	checksum := sha256.New()
	written, copyErr := io.Copy(io.MultiWriter(file, checksum), io.LimitReader(input, size+1))
	if copyErr == nil && (written != size || hex.EncodeToString(checksum.Sum(nil)) != digest) {
		copyErr = errors.New("staged update checksum mismatch")
	}
	if copyErr == nil {
		copyErr = file.Sync()
	}
	closeErr := file.Close()
	if copyErr != nil {
		return copyErr
	}
	if closeErr != nil {
		return closeErr
	}
	if err = configSafePath(root, name, true, false); err != nil {
		return err
	}
	return root.Rename(temporary, name)
}

func configTemporaryPath(name, token string) string {
	return path.Join(path.Dir(name), "."+path.Base(name)+".mmm-"+token+".tmp")
}

func sameConfigDirectory(first, second string) bool {
	firstInfo, firstErr := os.Stat(first)
	secondInfo, secondErr := os.Stat(second)
	if firstErr != nil || secondErr != nil {
		return filepath.Clean(first) == filepath.Clean(second)
	}
	return os.SameFile(firstInfo, secondInfo)
}
