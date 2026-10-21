package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"mmm-revived/internal/config"
)

func resolvedPath(path string) (string, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	var missing []string
	for {
		resolved, err := filepath.EvalSymlinks(absolute)
		if err == nil {
			for _, m := range slices.Backward(missing) {
				resolved = filepath.Join(resolved, m)
			}
			if runtime.GOOS == "windows" {
				resolved = strings.ToLower(resolved)
			}
			return resolved, nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
		parent := filepath.Dir(absolute)
		if parent == absolute {
			return "", err
		}
		missing = append(missing, filepath.Base(absolute))
		absolute = parent
	}
}

func containsPath(parent, child string) bool {
	relative, err := filepath.Rel(parent, child)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) && !filepath.IsAbs(relative)
}

func separateTrees(source, target string) error {
	left, err := resolvedPath(source)
	if err != nil {
		return err
	}
	right, err := resolvedPath(target)
	if err != nil {
		return err
	}
	if containsPath(left, right) || containsPath(right, left) {
		return fmt.Errorf("source and destination trees overlap: %s and %s", source, target)
	}
	return nil
}

func validateBuildDestination(root, out string) error {
	for _, directory := range []string{config.AssetsDir, config.PatchesDir} {
		if err := separateTrees(filepath.Join(root, directory), out); err != nil {
			return err
		}
	}
	return nil
}
