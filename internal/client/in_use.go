package client

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"mmm-revived/internal/patches"
)

func EnsureClientClosed(build string) error {
	if build == "" {
		return nil
	}
	for _, name := range []string{patches.Executable, "GameAssembly.dll", "xlua.dll"} {
		file, err := os.OpenFile(filepath.Join(build, name), os.O_WRONLY, 0)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return fmt.Errorf("%s is in use or not writable; close the game before changing its configuration: %w", name, err)
		}
		if err = file.Close(); err != nil {
			return err
		}
	}
	return nil
}
