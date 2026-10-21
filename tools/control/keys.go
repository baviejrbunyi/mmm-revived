package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"mmm-revived/internal/config"
	"mmm-revived/internal/gate"
)

func (w workflow) rotateKeys(c config.Config, backup string, publish func(config.Config) error) error {
	private, err := os.ReadFile(c.PrivateKey())
	if err != nil {
		return err
	}
	public, err := os.ReadFile(c.PublicKey())
	if err != nil {
		return err
	}
	manifestPath := filepath.Join(c.Patches(), "patchlist_win32.json")
	manifest, err := os.ReadFile(manifestPath)
	existed := err == nil
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	for name, data := range map[string][]byte{"server.pem": private, "client-public.pem": public} {
		if err = writeAtomic(filepath.Join(backup, "keys", name), data); err != nil {
			return err
		}
	}
	if existed {
		if err = writeAtomic(filepath.Join(backup, "patchlist_win32.json"), manifest); err != nil {
			return err
		}
	}
	staged := c
	staged.Keys = config.Keys{
		Private: filepath.Join(backup, "new-keys", "server.pem"),
		Public:  filepath.Join(backup, "new-keys", "client-public.pem"),
	}
	defer func() {
		os.Remove(staged.PrivateKey())
		os.Remove(staged.PublicKey())
		os.Remove(filepath.Dir(staged.PrivateKey()))
	}()
	if _, err = gate.LoadKey(staged.PrivateKey(), staged.PublicKey()); err != nil {
		return err
	}
	restore := func(cause error) error {
		var manifestErr error
		if existed {
			manifestErr = writeAtomic(manifestPath, manifest)
		} else {
			manifestErr = os.Remove(manifestPath)
			if errors.Is(manifestErr, os.ErrNotExist) {
				manifestErr = nil
			}
		}
		return errors.Join(cause, manifestErr, writeAtomic(c.PrivateKey(), private), writeAtomic(c.PublicKey(), public))
	}
	if err = publish(staged); err != nil {
		return restore(err)
	}
	for _, pair := range [][2]string{{staged.PrivateKey(), c.PrivateKey()}, {staged.PublicKey(), c.PublicKey()}} {
		data, err := os.ReadFile(pair[0])
		if err != nil {
			return restore(err)
		}
		if err = writeAtomic(pair[1], data); err != nil {
			return restore(err)
		}
	}
	if _, err = gate.LoadKey(c.PrivateKey(), c.PublicKey()); err != nil {
		return restore(err)
	}
	return nil
}

func unusedVersions(directory, resource, engine string) (string, string, error) {
	entries, err := os.ReadDir(filepath.Join(directory, "pcengine_files"))
	if errors.Is(err, os.ErrNotExist) {
		return resource, engine, nil
	}
	if err != nil {
		return "", "", err
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		candidate, err := nextVersion(entry.Name())
		if err != nil {
			return "", "", fmt.Errorf("invalid engine release directory %s: %w", entry.Name(), err)
		}
		if compareVersions(candidate, resource) > 0 {
			resource = candidate
		}
		if compareVersions(candidate, engine) > 0 {
			engine = candidate
		}
	}
	return resource, engine, nil
}

func compareVersions(left, right string) int {
	parse := func(value string) []uint64 {
		var result []uint64
		for part := range strings.SplitSeq(value, ".") {
			number, _ := strconv.ParseUint(part, 10, 32)
			result = append(result, number)
		}
		return result
	}
	return slices.Compare(parse(left), parse(right))
}
