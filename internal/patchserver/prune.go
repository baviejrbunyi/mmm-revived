package patchserver

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

func PruneRetail(directory string) error {
	info, err := os.Lstat(directory)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return errors.New("patch directory must be a real directory")
	}
	root, err := os.OpenRoot(directory)
	if err != nil {
		return err
	}
	defer root.Close()
	var list NativePatchList
	if err = retailReadJSON(root, nativeManifestName, &list); err != nil {
		return err
	}
	keep, err := retailFiles(list)
	if err != nil {
		return err
	}
	for name, expected := range keep {
		if err = retailSafePath(root, name, false); err != nil {
			return err
		}
		info, err := root.Stat(name)
		if err != nil || !info.Mode().IsRegular() || info.Size() != expected.Size {
			return fmt.Errorf("current patch is missing or incomplete: %s", name)
		}
	}
	var remove, directories []string
	for _, base := range []string{"patch_files", "pcengine_files"} {
		if err = fs.WalkDir(root.FS(), base, func(name string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.Type()&os.ModeSymlink != 0 {
				return fmt.Errorf("patch cleanup refuses a symbolic link: %s", name)
			}
			if entry.IsDir() {
				directories = append(directories, name)
			} else if _, found := keep[name]; !found {
				remove = append(remove, name)
			}
			return nil
		}); err != nil {
			return err
		}
	}
	for _, name := range []string{"upstream.json", "upstream-verified.json", "upstream.verified.json", "source/scripts.pck"} {
		if err = retailSafePath(root, name, true); err != nil {
			return err
		}
		info, err := root.Lstat(name)
		if err == nil {
			if !info.Mode().IsRegular() {
				return fmt.Errorf("obsolete patch metadata is not a regular file: %s", name)
			}
			remove = append(remove, name)
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	for _, name := range remove {
		if err = root.Remove(name); err != nil {
			return fmt.Errorf("remove obsolete patch %s: %w", name, err)
		}
	}
	slices.SortFunc(directories, func(left, right string) int {
		return strings.Count(right, "/") - strings.Count(left, "/")
	})
	for _, name := range directories {
		entries, err := fs.ReadDir(root.FS(), name)
		if err != nil {
			return err
		}
		if len(entries) == 0 {
			if err = root.Remove(filepath.ToSlash(name)); err != nil {
				return err
			}
		}
	}
	return nil
}
