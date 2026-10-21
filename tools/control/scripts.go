package main

import (
	"archive/zip"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"strings"

	"mmm-revived/internal/client"
	"mmm-revived/internal/patches"
	"mmm-revived/internal/patchserver"
)

const patchScriptsLimit int64 = 128 << 20

func patchScriptsMatch(directory string, list patchserver.NativePatchList, assets string, resources ...string) bool {
	if len(list.Files) != 1 {
		return false
	}
	expected, size, ok := patchScriptsDigest(filepath.Join(assets, "scripts", "scripts.pck"))
	if !ok {
		return false
	}
	type resourceDigest struct {
		hash string
		size int64
	}
	expectedFiles := map[string]resourceDigest{"scripts.pck": {hash: expected, size: size}}
	for _, path := range resources {
		digest, size, ok := patchScriptsDigest(filepath.Join(assets, filepath.FromSlash(path)))
		if !ok || !strings.HasPrefix(path, "resources/") {
			return false
		}
		expectedFiles[strings.TrimPrefix(path, "resources/")] = resourceDigest{hash: digest, size: size}
	}
	var name string
	var info patchserver.NativeFileInfo
	for _, versions := range list.Files {
		for _, prefix := range []string{"packc", "patch"} {
			candidate := prefix + "_first_" + client.Supported.Branch + "_win32_" + list.Version
			if value, exists := versions[list.Version][candidate]; exists {
				name, info = candidate, value
				break
			}
		}
	}
	if !patches.ValidPath(name) || strings.Contains(name, "/") || info.Size < 1 || info.Size > patchScriptsLimit {
		return false
	}
	root, err := os.OpenRoot(directory)
	if err != nil {
		return false
	}
	defer root.Close()
	parent, err := root.Lstat("patch_files")
	if err != nil || !parent.IsDir() || parent.Mode()&os.ModeSymlink != 0 {
		return false
	}
	stat, err := root.Lstat("patch_files/" + name)
	if err != nil || !stat.Mode().IsRegular() || stat.Size() != info.Size {
		return false
	}
	file, err := root.Open("patch_files/" + name)
	if err != nil {
		return false
	}
	defer file.Close()
	digest := md5.New()
	read, err := io.Copy(digest, io.LimitReader(file, info.Size+1))
	if err != nil || read != info.Size || hex.EncodeToString(digest.Sum(nil)) != info.MD5 {
		return false
	}
	reader, err := zip.NewReader(file, info.Size)
	if err != nil || len(reader.File) > patches.MaxFiles {
		return false
	}
	seen := make(map[string]bool, len(reader.File))
	selected := make(map[string]*zip.File, len(expectedFiles))
	var total uint64
	for _, entry := range reader.File {
		folded := strings.ToLower(entry.Name)
		if !patches.ValidPath(entry.Name) || entry.FileInfo().IsDir() || !entry.Mode().IsRegular() || seen[folded] || entry.UncompressedSize64 > uint64(patchScriptsLimit)-total {
			return false
		}
		seen[folded] = true
		total += entry.UncompressedSize64
		if _, exists := expectedFiles[entry.Name]; exists {
			selected[entry.Name] = entry
		}
	}
	if len(selected) != len(expectedFiles) {
		return false
	}
	for name, entry := range selected {
		expected := expectedFiles[name]
		if entry.UncompressedSize64 != uint64(expected.size) {
			return false
		}
		input, err := entry.Open()
		if err != nil {
			return false
		}
		digestSHA := sha256.New()
		read, readErr := io.Copy(digestSHA, io.LimitReader(input, expected.size+1))
		closeErr := input.Close()
		if readErr != nil || closeErr != nil || read != expected.size || hex.EncodeToString(digestSHA.Sum(nil)) != expected.hash {
			return false
		}
	}
	return true
}

func patchScriptsDigest(name string) (string, int64, bool) {
	info, err := os.Lstat(name)
	if err != nil || !info.Mode().IsRegular() || info.Size() < 1 || info.Size() > patchScriptsLimit {
		return "", 0, false
	}
	file, err := os.Open(name)
	if err != nil {
		return "", 0, false
	}
	defer file.Close()
	digest := sha256.New()
	size, err := io.Copy(digest, io.LimitReader(file, info.Size()+1))
	if err != nil || size != info.Size() {
		return "", 0, false
	}
	after, err := file.Stat()
	if err != nil || !info.ModTime().Equal(after.ModTime()) {
		return "", 0, false
	}
	return hex.EncodeToString(digest.Sum(nil)), size, true
}
