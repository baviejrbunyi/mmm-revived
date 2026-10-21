package patchserver

import (
	"crypto/md5"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"mmm-revived/internal/client"
	"mmm-revived/internal/json"
	"mmm-revived/internal/patches"
)

const retailSourceManifest = "source/engine/manifest.json"
const retailSourceFiles = "source/engine/files"
const retailSourceResources = "source/engine/resources"

type retailSourceFile struct {
	Size   int64  `json:"size"`
	MD5    string `json:"md5"`
	SHA256 string `json:"sha256"`
}

type retailSourceProfile struct {
	DLLSHA256      string `json:"dll_sha256"`
	AssemblySHA256 string `json:"assembly_sha256"`
	ScriptsSHA256  string `json:"scripts_sha256"`
}

type retailEngineSource struct {
	Version     int                         `json:"version"`
	BaseVersion string                      `json:"base_version"`
	Branch      string                      `json:"branch"`
	Profile     retailSourceProfile         `json:"profile"`
	Files       map[string]retailSourceFile `json:"files"`
	Resources   map[string]retailSourceFile `json:"resources"`
}

func retailSafePath(root *os.Root, name string, missing bool) error {
	if !patches.ValidPath(name) {
		return fmt.Errorf("unsafe retail path %q", name)
	}
	parts := strings.Split(name, "/")
	for index := range parts {
		current := strings.Join(parts[:index+1], "/")
		info, err := root.Lstat(current)
		if errors.Is(err, os.ErrNotExist) && missing {
			return nil
		}
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("retail path contains a symbolic link: %s", current)
		}
		if index < len(parts)-1 && !info.IsDir() {
			return fmt.Errorf("retail parent is not a directory: %s", current)
		}
	}
	return nil
}

func retailEnsureDirectory(root *os.Root, name string) error {
	if name == "." {
		return nil
	}
	if err := retailSafePath(root, name, true); err != nil {
		return err
	}
	if err := root.MkdirAll(name, 0755); err != nil {
		return err
	}
	if err := retailSafePath(root, name, false); err != nil {
		return err
	}
	info, err := root.Lstat(name)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("retail output is not a directory: %s", name)
	}
	return nil
}

func retailHashFile(root *os.Root, name string) (retailSourceFile, error) {
	if err := retailSafePath(root, name, false); err != nil {
		return retailSourceFile{}, err
	}
	info, err := root.Lstat(name)
	if err != nil {
		return retailSourceFile{}, err
	}
	if !info.Mode().IsRegular() || info.Size() < 1 || info.Size() > retailMaxFileSize {
		return retailSourceFile{}, fmt.Errorf("invalid retail source file: %s", name)
	}
	file, err := root.Open(name)
	if err != nil {
		return retailSourceFile{}, err
	}
	defer file.Close()
	md5Hash, shaHash := md5.New(), sha256.New()
	size, err := io.Copy(io.MultiWriter(md5Hash, shaHash), io.LimitReader(file, info.Size()+1))
	if err != nil {
		return retailSourceFile{}, err
	}
	if size != info.Size() {
		return retailSourceFile{}, fmt.Errorf("retail source changed while reading: %s", name)
	}
	return retailSourceFile{Size: size, MD5: hex.EncodeToString(md5Hash.Sum(nil)), SHA256: hex.EncodeToString(shaHash.Sum(nil))}, nil
}

func retailExcludedSource(name string, directory bool) bool {
	lower := strings.ToLower(name)
	if lower == "marvel mm cn_data/streamingassets" || strings.HasPrefix(lower, "marvel mm cn_data/streamingassets/") {
		return true
	}
	parts := strings.Split(lower, "/")
	for index, part := range parts {
		if index == len(parts)-1 && !directory {
			break
		}
		switch part {
		case ".git", "_offline_analysis", "private_server", "editor", "lua", "backup_files", "logs", "launcher_logs", "cache", "gpucache", "dawncache", "shadercache", "temp", "crashreports":
			return true
		}
		if strings.HasPrefix(part, ".mmm-revived") {
			return true
		}
	}
	base := path.Base(lower)
	if strings.HasPrefix(base, ".mmm-revived") {
		return true
	}
	if directory {
		return false
	}
	if base == "unisdkntdetectcache" || base == "mmm revived launcher.exe" {
		return true
	}
	for _, suffix := range []string{".log", ".tmp", ".dmp", ".cache", ".lock"} {
		if strings.HasSuffix(base, suffix) {
			return true
		}
	}
	return false
}

func retailInventory(root *os.Root) (map[string]retailSourceFile, error) {
	files := map[string]retailSourceFile{}
	seen := map[string]bool{}
	err := fs.WalkDir(root.FS(), ".", func(name string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if name == "." {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("retail source contains a symbolic link: %s", name)
		}
		if retailExcludedSource(name, entry.IsDir()) {
			if entry.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if !patches.ValidPath(name) {
			return fmt.Errorf("invalid retail source name: %s", name)
		}
		key := strings.ToLower(name)
		if seen[key] {
			return fmt.Errorf("case-colliding retail source paths: %s", name)
		}
		seen[key] = true
		if entry.IsDir() {
			return nil
		}
		if len(files) >= retailMaxFiles {
			return errors.New("too many retail engine source files")
		}
		info, err := retailHashFile(root, name)
		if err != nil {
			return err
		}
		files[name] = info
		return nil
	})
	return files, err
}

func retailCopyImmutable(source *os.Root, sourcePath string, target *os.Root, targetPath string, expected retailSourceFile) error {
	if err := retailSafePath(target, targetPath, true); err != nil {
		return err
	}
	if actual, err := retailHashFile(target, targetPath); err == nil {
		if actual != expected {
			return fmt.Errorf("immutable retail source or published file differs: %s", targetPath)
		}
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := retailEnsureDirectory(target, path.Dir(targetPath)); err != nil {
		return err
	}
	if err := retailSafePath(source, sourcePath, false); err != nil {
		return err
	}
	input, err := source.Open(sourcePath)
	if err != nil {
		return err
	}
	defer input.Close()
	var token [16]byte
	rand.Read(token[:])
	temporary := path.Join(path.Dir(targetPath), ".source-"+hex.EncodeToString(token[:])+".tmp")
	output, err := target.OpenFile(temporary, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return err
	}
	defer target.Remove(temporary)
	md5Hash, shaHash := md5.New(), sha256.New()
	size, copyErr := io.Copy(io.MultiWriter(output, md5Hash, shaHash), io.LimitReader(input, expected.Size+1))
	if copyErr == nil && (size != expected.Size || hex.EncodeToString(md5Hash.Sum(nil)) != expected.MD5 || hex.EncodeToString(shaHash.Sum(nil)) != expected.SHA256) {
		copyErr = fmt.Errorf("retail source changed during snapshot: %s", sourcePath)
	}
	if copyErr == nil {
		copyErr = output.Sync()
	}
	if closeErr := output.Close(); copyErr == nil {
		copyErr = closeErr
	}
	if copyErr != nil {
		return copyErr
	}
	if err = retailSafePath(target, targetPath, true); err != nil {
		return err
	}
	if _, err = target.Lstat(targetPath); err == nil {
		return fmt.Errorf("retail output appeared during snapshot: %s", targetPath)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return target.Link(temporary, targetPath)
}

func retailWriteImmutable(root *os.Root, name string, data []byte) error {
	if err := retailSafePath(root, name, true); err != nil {
		return err
	}
	if file, err := root.Open(name); err == nil {
		actual, readErr := io.ReadAll(io.LimitReader(file, int64(len(data))+1))
		file.Close()
		if readErr != nil {
			return readErr
		}
		if len(actual) != len(data) || client.Digest(actual) != client.Digest(data) {
			return fmt.Errorf("immutable retail publication changed: %s", name)
		}
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := retailEnsureDirectory(root, path.Dir(name)); err != nil {
		return err
	}
	return retailWriteBytes(root, name, data, false)
}

func retailWriteAtomic(root *os.Root, name string, data []byte) error {
	return retailWriteBytes(root, name, data, true)
}

func retailWriteBytes(root *os.Root, name string, data []byte, replace bool) error {
	if err := retailSafePath(root, name, true); err != nil {
		return err
	}
	if err := retailEnsureDirectory(root, path.Dir(name)); err != nil {
		return err
	}
	var token [16]byte
	rand.Read(token[:])
	temporary := path.Join(path.Dir(name), ".publish-"+hex.EncodeToString(token[:])+".tmp")
	output, err := root.OpenFile(temporary, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return err
	}
	defer root.Remove(temporary)
	_, writeErr := output.Write(data)
	if writeErr == nil {
		writeErr = output.Sync()
	}
	if closeErr := output.Close(); writeErr == nil {
		writeErr = closeErr
	}
	if writeErr != nil {
		return writeErr
	}
	if err = retailSafePath(root, name, true); err != nil {
		return err
	}
	if replace {
		return root.Rename(temporary, name)
	}
	return root.Link(temporary, name)
}

func retailProfile(profile client.BuildProfile) retailSourceProfile {
	return retailSourceProfile{DLLSHA256: profile.DLLSHA256, AssemblySHA256: profile.AssemblySHA256, ScriptsSHA256: profile.ScriptsSHA256}
}

func validateRetailSource(source retailEngineSource, profile client.BuildProfile) error {
	if source.Version != 1 || source.BaseVersion != profile.BaseVersion || source.Branch != profile.Branch || source.Profile != retailProfile(profile) || len(source.Files) == 0 || len(source.Files) > retailMaxFiles || len(source.Resources) != 3 {
		return errors.New("unsupported retail engine source snapshot")
	}
	seen := map[string]bool{}
	for name, info := range source.Files {
		key := strings.ToLower(name)
		if !patches.ValidPath(name) || retailExcludedSource(name, false) || seen[key] || !patches.ValidHash(info.SHA256) || !validRetailInfo(NativeFileInfo{Size: info.Size, MD5: info.MD5}) {
			return fmt.Errorf("invalid retail snapshot engine entry: %s", name)
		}
		seen[key] = true
	}
	if source.Files["xlua.dll"].SHA256 != profile.DLLSHA256 || source.Files["GameAssembly.dll"].SHA256 != profile.AssemblySHA256 || source.Files[patches.Executable].Size < 1 {
		return errors.New("retail engine snapshot lacks exact build files")
	}
	for _, name := range []string{"scripts.pck", "strs.pck", "package_version.json"} {
		info, exists := source.Resources[name]
		if !exists || !patches.ValidHash(info.SHA256) || !validRetailInfo(NativeFileInfo{Size: info.Size, MD5: info.MD5}) {
			return fmt.Errorf("invalid retail snapshot resource: %s", name)
		}
	}
	if source.Resources["scripts.pck"].SHA256 != profile.ScriptsSHA256 {
		return errors.New("retail engine snapshot has another scripts base")
	}
	return nil
}

func verifyRetailSource(root *os.Root, source retailEngineSource) error {
	for name, expected := range source.Files {
		actual, err := retailHashFile(root, retailSourceFiles+"/"+name)
		if err != nil || actual != expected {
			return fmt.Errorf("retail engine snapshot checksum mismatch: %s", name)
		}
	}
	for name, expected := range source.Resources {
		actual, err := retailHashFile(root, retailSourceResources+"/"+name)
		if err != nil || actual != expected {
			return fmt.Errorf("retail resource snapshot checksum mismatch: %s", name)
		}
	}
	return nil
}

func captureRetailSource(root *os.Root, installed *os.Root, profile client.BuildProfile) (retailEngineSource, error) {
	source := retailEngineSource{Version: 1, BaseVersion: profile.BaseVersion, Branch: profile.Branch, Profile: retailProfile(profile), Resources: map[string]retailSourceFile{}}
	var err error
	if source.Files, err = retailInventory(installed); err != nil {
		return source, err
	}
	for _, name := range []string{"scripts.pck", "strs.pck", "package_version.json"} {
		info, err := retailHashFile(installed, "Marvel MM CN_Data/StreamingAssets/"+name)
		if err != nil {
			return source, err
		}
		source.Resources[name] = info
	}
	if err = validateRetailSource(source, profile); err != nil {
		return source, err
	}
	for name, info := range source.Files {
		if err = retailCopyImmutable(installed, name, root, retailSourceFiles+"/"+name, info); err != nil {
			return source, err
		}
	}
	for name, info := range source.Resources {
		if err = retailCopyImmutable(installed, "Marvel MM CN_Data/StreamingAssets/"+name, root, retailSourceResources+"/"+name, info); err != nil {
			return source, err
		}
	}
	body, err := json.MarshalIndent(source, "", "  ")
	if err != nil {
		return source, err
	}
	if err = retailWriteImmutable(root, retailSourceManifest, body); err != nil {
		return source, err
	}
	return source, nil
}

func loadRetailSource(root *os.Root, installedPath string) (retailEngineSource, client.BuildProfile, error) {
	var source retailEngineSource
	profile := client.Supported
	var installed *os.Root
	if installedPath != "" {
		var err error
		installed, err = openRetailInstalled(root.Name(), installedPath, profile)
		if err != nil {
			return source, profile, err
		}
		defer installed.Close()
	}
	if err := retailSafePath(root, retailSourceManifest, true); err != nil {
		return source, profile, err
	}
	if file, err := root.Open(retailSourceManifest); err == nil {
		defer file.Close()
		decoder := json.NewDecoder(io.LimitReader(file, patches.MaxManifestSize+1))
		if err = decoder.Decode(&source); err != nil {
			return source, profile, err
		}
		if err = decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
			return source, profile, errors.New("trailing retail source manifest content")
		}
		if err = validateRetailSource(source, profile); err != nil {
			return source, profile, err
		}
		if err = verifyRetailSource(root, source); err != nil {
			return source, profile, err
		}
		return source, profile, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return source, profile, err
	}
	if installedPath == "" {
		return source, profile, errors.New("client engine snapshot is missing; specify the original client directory")
	}
	source, err := captureRetailSource(root, installed, profile)
	return source, profile, err
}

func openRetailInstalled(outputPath, installedPath string, profile client.BuildProfile) (*os.Root, error) {
	outputAbsolute, err := filepath.Abs(outputPath)
	if err != nil {
		return nil, err
	}
	installedAbsolute, err := filepath.Abs(installedPath)
	if err != nil {
		return nil, err
	}
	relative, err := filepath.Rel(installedAbsolute, outputAbsolute)
	if err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) && !filepath.IsAbs(relative) {
		return nil, errors.New("retail publication directory must be outside the source client")
	}
	info, err := os.Lstat(installedPath)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("retail client source must be a real directory")
	}
	installed, err := os.OpenRoot(installedPath)
	if err != nil {
		return nil, err
	}
	for _, name := range []string{"xlua.dll", "GameAssembly.dll", "Marvel MM CN_Data/StreamingAssets/scripts.pck"} {
		if err = retailSafePath(installed, name, false); err != nil {
			installed.Close()
			return nil, err
		}
	}
	detected, err := client.DetectBuild(installedPath)
	if err != nil {
		installed.Close()
		return nil, err
	}
	if retailProfile(detected) != retailProfile(profile) {
		installed.Close()
		return nil, errors.New("client source does not match the supported build")
	}
	dll, err := retailHashFile(installed, "xlua.dll")
	if err != nil || dll.SHA256 != profile.DLLSHA256 {
		installed.Close()
		return nil, errors.New("client DLL is not the original binary")
	}
	return installed, nil
}
