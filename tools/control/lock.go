package main

import (
	"errors"
	"fmt"
	"os"
	"sync"
)

func acquireSetupLock(path string) (func(), error) {
	if info, err := os.Lstat(path); err == nil {
		if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("setup lock must be a regular file: %s", path)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		return nil, err
	}
	if err = setupLockFile(file); err != nil {
		file.Close()
		return nil, fmt.Errorf("another setup command may be running; cannot acquire %s: %w", path, err)
	}
	if err = file.Truncate(0); err == nil {
		_, err = fmt.Fprintln(file, os.Getpid())
	}
	if err != nil {
		setupUnlockFile(file)
		file.Close()
		return nil, err
	}
	return sync.OnceFunc(func() {
		setupUnlockFile(file)
		file.Close()
	}), nil
}
