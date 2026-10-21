package client

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"time"
)

const staleLockAge = 10 * time.Minute

type lockedRoot struct {
	*os.Root
	lock string
}

func openLockedRoot(directory, lockName string) (*lockedRoot, error) {
	root, err := os.OpenRoot(directory)
	if err != nil {
		return nil, err
	}
	if err = acquireLock(root, lockName); err != nil {
		root.Close()
		return nil, err
	}
	return &lockedRoot{Root: root, lock: lockName}, nil
}

func acquireLock(root *os.Root, name string) error {
	if err := configSafePath(root, name, true, false); err != nil {
		return err
	}
	for attempt := range 2 {
		file, err := root.OpenFile(name, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err == nil {
			_, err = fmt.Fprintf(file, "%d %d\n", os.Getpid(), time.Now().Unix())
			return errors.Join(err, file.Close())
		}
		if !errors.Is(err, fs.ErrExist) {
			return err
		}
		info, statErr := root.Lstat(name)
		if attempt > 0 || statErr != nil || time.Since(info.ModTime()) < staleLockAge {
			return fmt.Errorf("another client configuration may be running; remove %s if it is not", name)
		}
		if err = root.Remove(name); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
	}
	return nil
}

func (r *lockedRoot) Close() {
	r.Root.Remove(r.lock)
	r.Root.Close()
}
