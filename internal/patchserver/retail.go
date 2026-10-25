package patchserver

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"mmm-revived/internal/client"
	"mmm-revived/internal/json"
	"mmm-revived/internal/patches"
)

const retailMaxFileSize int64 = 2 << 30
const retailMaxFiles = 4096

func (h *NativeHandler) ConfigureRetail(origin string, publicKey []byte) error {
	base, err := url.Parse(origin)
	if err != nil || base.Host == "" || base.User != nil || base.RawQuery != "" || base.Fragment != "" || (base.Scheme != "http" && base.Scheme != "https") || base.Path != "" {
		return errors.New("invalid retail hosting origin")
	}
	if h.root == nil {
		return errors.New("retail patches are not loaded")
	}
	path := "pcengine_files/" + h.list.PCEngineVersion + "/xlua.dll"
	entry, found := h.files[path]
	if !found || entry.info.Size > 32<<20 {
		return errors.New("retail public key patch is missing")
	}
	file, err := h.root.Open(path)
	if err != nil {
		return err
	}
	data, err := io.ReadAll(io.LimitReader(file, entry.info.Size+1))
	file.Close()
	if err != nil || nativeHash(data) != entry.info.MD5 || len(publicKey) < 64 || !bytes.Contains(data, publicKey) {
		return errors.New("retail engine patch does not match server public key; publish a new engine version")
	}
	base.Path = "/native"
	for _, versions := range h.list.Files {
		h.list.Files = map[string]map[string]map[string]NativeFileInfo{base.String(): versions}
		break
	}
	h.list.PCEngineURL = base.String()
	h.body, err = json.MarshalIndent(h.list, "", "  ")
	return err
}

type NativeEngineInfo struct {
	MD5  string
	Size int64
}

func (f NativeEngineInfo) MarshalJSON() ([]byte, error) {
	return json.Marshal([2]any{f.MD5, f.Size})
}

func (f *NativeEngineInfo) UnmarshalJSON(data []byte) error {
	var fields []json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil || len(fields) != 2 {
		return errors.New("native engine info requires MD5 and size tuple")
	}
	var value NativeEngineInfo
	if err := json.Unmarshal(fields[0], &value.MD5); err != nil {
		return err
	}
	if err := json.Unmarshal(fields[1], &value.Size); err != nil {
		return err
	}
	*f = value
	return nil
}

func validRetailInfo(info NativeFileInfo) bool {
	digest, err := hex.DecodeString(info.MD5)
	return err == nil && len(digest) == md5.Size && strings.ToLower(info.MD5) == info.MD5 && info.Size > 0 && info.Size <= retailMaxFileSize
}

func retailFiles(list NativePatchList) (map[string]NativeFileInfo, error) {
	for _, version := range []string{list.Version, list.BaseVersion, list.PCEngineVersion} {
		if _, err := nativeVersion(version); err != nil {
			return nil, err
		}
	}
	if len(list.Files) != 1 || len(list.PCEngineFiles) == 0 {
		return nil, errors.New("retail patches require one resource origin and a complete engine manifest")
	}
	if compareNativeVersion(list.BaseVersion, list.Version) > 0 {
		return nil, errors.New("retail base version exceeds current version")
	}
	files := map[string]NativeFileInfo{}
	for origin, versions := range list.Files {
		if len(versions[list.Version]) == 0 {
			return nil, errors.New("retail current version has no resource files")
		}
		if origin != list.PCEngineURL {
			return nil, errors.New("resource and engine origins differ")
		}
		base, err := url.Parse(origin)
		if err != nil || base.Host == "" || base.User != nil || base.RawQuery != "" || base.Fragment != "" || (base.Scheme != "http" && base.Scheme != "https") || base.Path != "/native" {
			return nil, errors.New("invalid local retail patch origin")
		}
		for version, entries := range versions {
			if _, err := nativeVersion(version); err != nil || compareNativeVersion(version, list.Version) > 0 {
				return nil, errors.New("invalid retail resource version")
			}
			for name, info := range entries {
				if !patches.ValidPath(name) || strings.Contains(name, "/") || !validRetailInfo(info) {
					return nil, fmt.Errorf("invalid retail patch entry %q", name)
				}
				key := "patch_files/" + name
				if previous, found := files[key]; found && previous != info {
					return nil, errors.New("conflicting retail patch entries")
				}
				files[key] = info
			}
		}
	}
	for name, info := range list.PCEngineFiles {
		entry := NativeFileInfo{Size: info.Size, MD5: info.MD5}
		if !patches.ValidPath(name) || !validRetailInfo(entry) {
			return nil, fmt.Errorf("invalid retail engine entry %q", name)
		}
		files["pcengine_files/"+list.PCEngineVersion+"/"+name] = entry
	}
	if len(files) > retailMaxFiles {
		return nil, errors.New("too many retail patch files")
	}
	return files, nil
}

func openRetail(directory string, list NativePatchList) (*NativeHandler, error) {
	files, err := retailFiles(list)
	if err != nil {
		return nil, err
	}
	root, err := os.OpenRoot(directory)
	if err != nil {
		return nil, err
	}
	handler := &NativeHandler{list: list, root: root, files: make(map[string]nativeServedFile, len(files))}
	for name, info := range files {
		if err = retailSafePath(root, name, false); err != nil {
			root.Close()
			return nil, err
		}
		stat, err := root.Stat(name)
		if err != nil || !stat.Mode().IsRegular() || stat.Size() != info.Size {
			root.Close()
			return nil, fmt.Errorf("retail patch missing or wrong size: %s", name)
		}
		handler.files[name] = nativeServedFile{path: name, info: info}
	}
	handler.body, err = json.MarshalIndent(list, "", "  ")
	if err != nil {
		root.Close()
		return nil, err
	}
	return handler, nil
}

func verifyRetailFiles(directory string, list NativePatchList) error {
	files, err := retailFiles(list)
	if err != nil {
		return err
	}
	root, err := os.OpenRoot(directory)
	if err != nil {
		return err
	}
	defer root.Close()
	for name, info := range files {
		if err = retailSafePath(root, name, false); err != nil {
			return err
		}
		file, err := root.Open(name)
		if err != nil {
			return fmt.Errorf("open %s: %w", name, err)
		}
		digest := md5.New()
		size, err := io.Copy(digest, io.LimitReader(file, info.Size+1))
		file.Close()
		if err != nil || size != info.Size || hex.EncodeToString(digest.Sum(nil)) != info.MD5 {
			return fmt.Errorf("retail patch checksum mismatch: %s", name)
		}
	}
	return nil
}

type RetailPrepareOptions struct {
	Client        string
	Directory     string
	Origin        string
	Scripts       string
	PublicKey     string
	Modules       string
	Version       string
	EngineVersion string
	Content       string
}

func PrepareRetail(options RetailPrepareOptions) (NativePatchList, error) {
	var empty NativePatchList
	if _, err := nativeVersion(options.Version); err != nil || compareNativeVersion(options.Version, client.Supported.BaseVersion) <= 0 || compareNativeVersion(options.Version, client.Supported.ContentVersion) <= 0 {
		return empty, errors.New("resource version must exceed the client base version")
	}
	if _, err := nativeVersion(options.EngineVersion); err != nil || compareNativeVersion(options.EngineVersion, client.Supported.BaseVersion) <= 0 {
		return empty, errors.New("engine version must exceed the client base version")
	}
	base, err := url.Parse(options.Origin)
	if err != nil || base.Host == "" || base.User != nil || base.RawQuery != "" || base.Fragment != "" || (base.Scheme != "http" && base.Scheme != "https") || (base.Path != "" && base.Path != "/") {
		return empty, errors.New("invalid retail server origin")
	}
	base.Path = "/native"
	if err = os.MkdirAll(options.Directory, 0755); err != nil {
		return empty, err
	}
	rootInfo, err := os.Lstat(options.Directory)
	if err != nil {
		return empty, err
	}
	if !rootInfo.IsDir() || rootInfo.Mode()&os.ModeSymlink != 0 {
		return empty, errors.New("retail publication directory must be a real directory")
	}
	root, err := os.OpenRoot(options.Directory)
	if err != nil {
		return empty, err
	}
	defer root.Close()
	var previous NativePatchList
	hasPrevious := false
	if err = retailReadJSON(root, nativeManifestName, &previous); err == nil {
		hasPrevious = true
		for _, version := range []string{previous.Version, previous.PCEngineVersion} {
			if _, err = nativeVersion(version); err != nil {
				return empty, errors.New("invalid previous retail patch version")
			}
		}
		if compareNativeVersion(options.Version, previous.Version) < 0 || compareNativeVersion(options.EngineVersion, previous.PCEngineVersion) < 0 {
			return empty, errors.New("retail patches cannot downgrade")
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return empty, err
	}
	source, profile, err := loadRetailSource(root, options.Client)
	if err != nil {
		return empty, err
	}
	scripts, err := readSeed(options.Scripts)
	if err != nil {
		return empty, err
	}
	if client.Digest(scripts) != profile.PatchedScriptsSHA256 {
		return empty, errors.New("unapproved CN retail scripts bundle")
	}
	if err = validateUI(scripts); err != nil {
		return empty, err
	}
	original, err := retailReadSource(root, retailSourceFiles+"/xlua.dll", source.Files["xlua.dll"])
	if err != nil {
		return empty, err
	}
	key, err := os.ReadFile(options.PublicKey)
	if err != nil {
		return empty, err
	}
	var modules [][]byte
	for _, file := range profile.Modules() {
		module, err := os.ReadFile(filepath.Join(options.Modules, filepath.FromSlash(file.Path)))
		if err != nil {
			return empty, err
		}
		modules = append(modules, module)
	}
	patched, err := client.PatchClientModules(original, key, modules)
	if err != nil {
		return empty, err
	}
	content, err := loadContent(root, options.Content, profile)
	if err != nil {
		return empty, err
	}
	if err = applyLegacyCatalog(options.Modules, content.first); err != nil {
		return empty, err
	}
	resources := []NativeResource{{Path: "scripts.pck", Data: scripts}, {Path: "pcengine_files/xlua.dll", Data: patched}}
	for _, name := range []string{"strs.pck", "assets.bytes", "packages.bytes"} {
		data, found := content.first[name]
		if !found {
			return empty, fmt.Errorf("upstream first package lacks %s", name)
		}
		resources = append(resources, NativeResource{Path: name, Data: data})
	}
	for _, file := range profile.Resources() {
		data, err := os.ReadFile(filepath.Join(options.Modules, filepath.FromSlash(file.Path)))
		if err != nil {
			return empty, err
		}
		if client.Digest(data) != file.SHA256 {
			return empty, fmt.Errorf("unapproved retail resource: %s", file.Path)
		}
		resources = append(resources, NativeResource{Path: strings.TrimPrefix(file.Path, "resources/"), Data: data})
	}
	pack, err := BuildNativeResourcePackage(options.Version, "first", resources)
	if err != nil {
		return empty, err
	}
	entries := map[string]NativeFileInfo{}
	for _, prefix := range nativePrefixes {
		name := prefix + "_first_" + client.Supported.Branch + "_win32_" + options.Version
		entries[name] = pack.Info
		if hasPrevious && previous.Version == options.Version {
			for _, oldVersions := range previous.Files {
				if oldVersions[options.Version][name] != pack.Info {
					return empty, errors.New("CN resources changed; increase the resource patch version")
				}
			}
		}
	}
	if hasPrevious && previous.PCEngineVersion == options.EngineVersion {
		if previous.PCEngineFiles["xlua.dll"].MD5 != nativeHash(patched) || len(previous.PCEngineFiles) != len(source.Files) {
			return empty, errors.New("CN engine changed; increase the engine patch version")
		}
		for name, info := range source.Files {
			if name != "xlua.dll" && previous.PCEngineFiles[name] != (NativeEngineInfo{MD5: info.MD5, Size: info.Size}) {
				return empty, errors.New("CN engine inventory changed; increase the engine patch version")
			}
		}
	}
	engine := make(map[string]NativeEngineInfo, len(source.Files))
	for name, info := range source.Files {
		target := "pcengine_files/" + options.EngineVersion + "/" + name
		if name == "xlua.dll" {
			if err = retailWriteImmutable(root, target, patched); err != nil {
				return empty, err
			}
			engine[name] = NativeEngineInfo{MD5: nativeHash(patched), Size: int64(len(patched))}
		} else {
			if err = retailCopyImmutable(root, retailSourceFiles+"/"+name, root, target, info); err != nil {
				return empty, err
			}
			engine[name] = NativeEngineInfo{MD5: info.MD5, Size: info.Size}
		}
	}
	for name := range entries {
		if err = retailWriteImmutable(root, "patch_files/"+name, pack.Data); err != nil {
			return empty, err
		}
	}
	list := retailCurrentRelease(options, base.String(), entries, engine, content)
	if err = verifyRetailFiles(options.Directory, list); err != nil {
		return list, err
	}
	handler, err := openRetail(options.Directory, list)
	if err != nil {
		return list, err
	}
	defer handler.Close()
	if err = retailSafePath(root, nativeManifestName, true); err != nil {
		return list, err
	}
	if err = retailWriteAtomic(root, nativeManifestName, handler.body); err != nil {
		return list, err
	}
	return list, nil
}

func retailReadJSON(root *os.Root, name string, target any) error {
	if err := retailSafePath(root, name, false); err != nil {
		return err
	}
	file, err := root.Open(name)
	if err != nil {
		return err
	}
	defer file.Close()
	decoder := json.NewDecoder(io.LimitReader(file, 8<<20))
	if err = decoder.Decode(target); err != nil {
		return err
	}
	if err = decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		return errors.New("trailing retail JSON content")
	}
	return nil
}

func retailReadSource(root *os.Root, name string, expected retailSourceFile) ([]byte, error) {
	actual, err := retailHashFile(root, name)
	if err != nil || actual != expected {
		return nil, fmt.Errorf("retail source checksum mismatch: %s", name)
	}
	file, err := root.Open(name)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, expected.Size+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) != expected.Size || nativeHash(data) != expected.MD5 || client.Digest(data) != expected.SHA256 {
		return nil, fmt.Errorf("retail source changed while reading: %s", name)
	}
	return data, nil
}

func retailCurrentRelease(options RetailPrepareOptions, origin string, entries map[string]NativeFileInfo, engine map[string]NativeEngineInfo, content contentState) NativePatchList {
	versions := content.versions()
	versions[options.Version] = entries
	return NativePatchList{
		Version: options.Version, BaseVersion: client.Supported.BaseVersion, RequireVersion: client.Supported.BaseVersion,
		ExtraFirstPack: content.list.ExtraFirstPack, ExtraPackMap: content.list.ExtraPackMap,
		PCEngineVersion: options.EngineVersion, PCEngineURL: origin, PCEngineFiles: engine,
		Files:           map[string]map[string]map[string]NativeFileInfo{origin: versions},
		RestartVersions: map[string]bool{options.Version: true},
	}
}
