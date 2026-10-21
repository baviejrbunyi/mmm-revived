package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"mmm-revived/internal/config"
)

func (w workflow) reset() error {
	unlock, err := w.lock()
	if err != nil {
		return err
	}
	defer unlock()
	path := filepath.Join(w.root, "build", config.FileName)
	if !config.Exists(path) {
		return fmt.Errorf("read server configuration: %s is missing", path)
	}
	c, err := config.Load(path)
	if err != nil {
		return err
	}
	if err = available(c.Address(), c.HTTPAddress()); err != nil {
		return fmt.Errorf("stop the server with Ctrl+C before running just reset: %w", err)
	}
	directory := w.root
	for _, part := range []string{"build", config.DataDir, "accounts"} {
		directory = filepath.Join(directory, part)
		info, err := os.Lstat(directory)
		if errors.Is(err, os.ErrNotExist) {
			fmt.Fprintln(w.output, "Database is already empty. Run just run to start the server.")
			return nil
		}
		if err != nil {
			return err
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("database directory must be a real directory: %s", directory)
		}
	}
	root, err := os.OpenRoot(directory)
	if err != nil {
		return err
	}
	defer root.Close()
	files := []string{"mmm-revived.db", "mmm-revived.db-wal", "mmm-revived.db-shm", "mmm-revived.db-journal"}
	for _, name := range files {
		info, err := root.Lstat(name)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("database file must be a regular file: %s", name)
		}
	}
	for _, name := range files {
		if err := root.Remove(name); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("remove %s: %w", name, err)
		}
	}
	fmt.Fprintln(w.output, "Database reset. Accounts, passwords and progress cleared. Run just run to start the server.")
	return nil
}
