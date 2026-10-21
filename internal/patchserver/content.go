package patchserver

import (
	"archive/zip"
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"mmm-revived/internal/client"
	"mmm-revived/internal/json"
	"mmm-revived/internal/patches"
)

const (
	ContentManifest   = "patchlist.json"
	contentDirectory  = "source/content"
	contentListPath   = contentDirectory + "/" + ContentManifest
	contentFirstPack  = "first"
	contentListLimit  = 8 << 20
	contentEntryLimit = 256 << 20
)

type ContentFile struct {
	Name    string
	Pack    string
	Version string
	Origin  string
	Info    NativeFileInfo
}

func ParseContent(data []byte, profile client.BuildProfile) (NativePatchList, []ContentFile, error) {
	var list NativePatchList
	if len(data) > contentListLimit {
		return list, nil, errors.New("upstream patch list is too large")
	}
	if err := json.Unmarshal(data, &list); err != nil {
		return list, nil, fmt.Errorf("decode upstream patch list: %w", err)
	}
	if list.Version != profile.ContentVersion {
		return list, nil, fmt.Errorf("upstream content version is %s; this server supports %s", list.Version, profile.ContentVersion)
	}
	if len(list.Files) != 1 {
		return list, nil, errors.New("upstream patch list must have one origin")
	}
	var files []ContentFile
	seen := map[string]bool{}
	for origin, versions := range list.Files {
		base, err := url.Parse(origin)
		if err != nil || base.Scheme != "https" && base.Scheme != "http" || base.Host == "" || base.User != nil || base.RawQuery != "" || base.Fragment != "" {
			return list, nil, errors.New("invalid upstream patch origin")
		}
		for version, entries := range versions {
			if _, err := nativeVersion(version); err != nil || compareNativeVersion(version, list.Version) > 0 {
				return list, nil, fmt.Errorf("invalid upstream patch version %q", version)
			}
			if compareNativeVersion(version, profile.BaseVersion) < 0 {
				continue
			}
			for name, info := range entries {
				pack, ok := contentPack(name, profile.Branch, version)
				if !ok || !patches.ValidPath(name) || strings.Contains(name, "/") || !validRetailInfo(info) || info.Size > retailMaxFileSize || seen[name] {
					return list, nil, fmt.Errorf("invalid upstream patch entry %q", name)
				}
				seen[name] = true
				files = append(files, ContentFile{Name: name, Pack: pack, Version: version, Origin: origin, Info: info})
			}
		}
	}
	if len(files) == 0 || len(files) > retailMaxFiles {
		return list, nil, errors.New("upstream patch list has an invalid number of files")
	}
	if _, err := contentFirst(files, profile); err != nil {
		return list, nil, err
	}
	return list, files, nil
}

func contentPack(name, branch, version string) (string, bool) {
	for _, prefix := range nativePrefixes {
		rest, found := strings.CutPrefix(name, prefix+"_")
		if !found {
			continue
		}
		pack, found := strings.CutSuffix(rest, "_"+branch+"_win32_"+version)
		if found && pack != "" && !strings.ContainsAny(pack, "_/\\.") {
			return pack, true
		}
	}
	return "", false
}

func contentFirst(files []ContentFile, profile client.BuildProfile) (ContentFile, error) {
	for _, file := range files {
		if file.Pack == contentFirstPack && file.Version == profile.ContentVersion && strings.HasPrefix(file.Name, "patch_") {
			return file, nil
		}
	}
	return ContentFile{}, errors.New("upstream patch list lacks the current first package")
}

func contentHash(file *os.File, size int64) (string, error) {
	digest := md5.New()
	read, err := io.Copy(digest, io.LimitReader(file, size+1))
	if err != nil {
		return "", err
	}
	if read != size {
		return "", errors.New("size mismatch")
	}
	return hex.EncodeToString(digest.Sum(nil)), nil
}

func contentImport(root *os.Root, source contentSource, name, target string, info NativeFileInfo) error {
	if err := retailSafePath(root, target, true); err != nil {
		return err
	}
	if existing, err := root.Open(target); err == nil {
		stat, statErr := existing.Stat()
		matches := statErr == nil && stat.Mode().IsRegular() && stat.Size() == info.Size
		if matches {
			digest, hashErr := contentHash(existing, info.Size)
			matches = hashErr == nil && digest == info.MD5
		}
		existing.Close()
		if matches {
			return nil
		}
		if err = root.Remove(target); err != nil {
			return err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	input, err := source.open(name)
	if err != nil {
		return fmt.Errorf("game content is missing %s: %w", name, err)
	}
	defer input.Close()
	if err = retailEnsureDirectory(root, filepath.ToSlash(filepath.Dir(filepath.FromSlash(target)))); err != nil {
		return err
	}
	temporary := target + ".import"
	if path := source.path(name); path != "" {
		if linked, err := contentLink(root, path, temporary, info); err != nil || linked {
			if err != nil {
				return err
			}
			return root.Rename(temporary, target)
		}
	}
	output, err := root.OpenFile(temporary, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer root.Remove(temporary)
	digest := md5.New()
	size, copyErr := io.Copy(io.MultiWriter(output, digest), io.LimitReader(input, info.Size+1))
	if copyErr == nil && (size != info.Size || hex.EncodeToString(digest.Sum(nil)) != info.MD5) {
		copyErr = fmt.Errorf("game content file %s does not match its checksum", name)
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
	return root.Rename(temporary, target)
}

type contentSource interface {
	manifest() ([]byte, error)
	open(name string) (io.ReadCloser, error)
	path(name string) string
	Close() error
}

type contentFolder string

func (f contentFolder) manifest() ([]byte, error) {
	return os.ReadFile(filepath.Join(string(f), ContentManifest))
}

func (f contentFolder) open(name string) (io.ReadCloser, error) {
	return os.Open(f.path(name))
}

func (f contentFolder) path(name string) string {
	return filepath.Join(string(f), "patch_files", name)
}

func (contentFolder) Close() error {
	return nil
}

type contentArchive struct {
	reader  *zip.ReadCloser
	entries map[string]*zip.File
	list    *zip.File
}

func (a contentArchive) manifest() ([]byte, error) {
	input, err := a.list.Open()
	if err != nil {
		return nil, err
	}
	defer input.Close()
	data, err := io.ReadAll(io.LimitReader(input, contentListLimit+1))
	if err == nil && len(data) > contentListLimit {
		err = errors.New("game content patch list is too large")
	}
	return data, err
}

func (a contentArchive) open(name string) (io.ReadCloser, error) {
	entry, ok := a.entries[name]
	if !ok {
		return nil, os.ErrNotExist
	}
	return entry.Open()
}

func (contentArchive) path(string) string {
	return ""
}

func (a contentArchive) Close() error {
	return a.reader.Close()
}

func openContent(location string) (contentSource, error) {
	stat, err := os.Stat(location)
	if err != nil {
		return nil, err
	}
	if stat.IsDir() {
		return contentFolder(location), nil
	}
	reader, err := zip.OpenReader(location)
	if err != nil {
		return nil, fmt.Errorf("open game content archive %s: %w", filepath.Base(location), err)
	}
	archive := contentArchive{reader: reader, entries: map[string]*zip.File{}}
	prefix, depth := "", -1
	for _, entry := range reader.File {
		if path.Base(entry.Name) != ContentManifest || entry.FileInfo().IsDir() {
			continue
		}
		if level := strings.Count(entry.Name, "/"); depth < 0 || level < depth {
			archive.list, depth = entry, level
			prefix = strings.TrimSuffix(entry.Name, ContentManifest)
		}
	}
	if archive.list == nil {
		reader.Close()
		return nil, fmt.Errorf("game content archive %s has no %s", filepath.Base(location), ContentManifest)
	}
	for _, entry := range reader.File {
		if name, ok := strings.CutPrefix(entry.Name, prefix+"patch_files/"); ok && name != "" && !strings.Contains(name, "/") {
			archive.entries[name] = entry
		}
	}
	return archive, nil
}

func contentLink(root *os.Root, source, temporary string, info NativeFileInfo) (bool, error) {
	if err := root.Remove(temporary); err != nil && !errors.Is(err, os.ErrNotExist) {
		return false, err
	}
	if os.Link(source, filepath.Join(root.Name(), filepath.FromSlash(temporary))) != nil {
		return false, nil
	}
	linked, err := root.Open(temporary)
	if err != nil {
		return false, err
	}
	digest, hashErr := contentHash(linked, info.Size)
	linked.Close()
	if hashErr != nil || digest != info.MD5 {
		root.Remove(temporary)
		return false, fmt.Errorf("game content file %s does not match its checksum", filepath.Base(source))
	}
	return true, nil
}

type contentState struct {
	list  NativePatchList
	files []ContentFile
	first map[string][]byte
}

func loadContent(root *os.Root, mirror string, profile client.BuildProfile) (contentState, error) {
	var state contentState
	var data []byte
	var err error
	var source contentSource
	if mirror != "" {
		if source, err = openContent(mirror); err != nil {
			return state, err
		}
		defer source.Close()
		data, err = source.manifest()
	} else {
		if err = retailSafePath(root, contentListPath, false); err != nil {
			return state, errors.New("game content is missing; put the content zip in the content folder of this repository, then run just configure \"<client directory>\" again")
		}
		data, err = root.ReadFile(contentListPath)
	}
	if err != nil {
		return state, err
	}
	if state.list, state.files, err = ParseContent(data, profile); err != nil {
		return state, err
	}
	first, _ := contentFirst(state.files, profile)
	firstPath := contentDirectory + "/" + first.Name
	for _, file := range state.files {
		target := "patch_files/" + file.Name
		if file.Pack == contentFirstPack {
			if file.Name != first.Name {
				continue
			}
			target = firstPath
		}
		if mirror != "" {
			if err = contentImport(root, source, file.Name, target, file.Info); err != nil {
				return state, err
			}
		}
	}
	if mirror != "" {
		if err = retailEnsureDirectory(root, contentDirectory); err != nil {
			return state, err
		}
		if err = retailWriteAtomic(root, contentListPath, data); err != nil {
			return state, err
		}
	}
	archive, err := root.ReadFile(firstPath)
	if err != nil {
		return state, fmt.Errorf("read upstream first package: %w", err)
	}
	if int64(len(archive)) != first.Info.Size || nativeHash(archive) != first.Info.MD5 {
		return state, errors.New("upstream first package checksum mismatch")
	}
	reader, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		return state, err
	}
	state.first = map[string][]byte{}
	for _, entry := range reader.File {
		if entry.FileInfo().IsDir() || !patches.ValidPath(entry.Name) || strings.HasSuffix(entry.Name, ".mdf") || entry.UncompressedSize64 > contentEntryLimit {
			return state, fmt.Errorf("unsupported upstream first package entry %q", entry.Name)
		}
		input, err := entry.Open()
		if err != nil {
			return state, err
		}
		value, err := io.ReadAll(io.LimitReader(input, contentEntryLimit+1))
		input.Close()
		if err != nil {
			return state, err
		}
		state.first[entry.Name] = value
	}
	if client.Digest(state.first["scripts.pck"]) != profile.ContentScriptsSHA256 {
		return state, errors.New("upstream scripts bundle does not match the supported content")
	}
	return state, nil
}

func (state contentState) versions() map[string]map[string]NativeFileInfo {
	versions := map[string]map[string]NativeFileInfo{}
	for _, file := range state.files {
		if file.Pack == contentFirstPack {
			continue
		}
		if versions[file.Version] == nil {
			versions[file.Version] = map[string]NativeFileInfo{}
		}
		versions[file.Version][file.Name] = file.Info
	}
	return versions
}
