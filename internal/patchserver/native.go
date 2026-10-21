package patchserver

import (
	"archive/zip"
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"mmm-revived/internal/json"
	"mmm-revived/internal/patches"
)

const nativeManifestName = "patchlist_win32.json"

var (
	nativePrefixes           = [...]string{"packc", "patch"}
	errNativePackageMetadata = errors.New("native resource package metadata mismatch")
)

type NativeFileInfo struct {
	Size int64
	MD5  string
}

func (f NativeFileInfo) MarshalJSON() ([]byte, error) {
	return json.Marshal([2]any{f.Size, f.MD5})
}

func (f *NativeFileInfo) UnmarshalJSON(data []byte) error {
	var fields []json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil || len(fields) != 2 {
		return errors.New("native patch file info requires size and MD5 tuple")
	}
	var value NativeFileInfo
	if err := json.Unmarshal(fields[0], &value.Size); err != nil {
		return err
	}
	if err := json.Unmarshal(fields[1], &value.MD5); err != nil {
		return err
	}
	*f = value
	return nil
}

type NativePatchList struct {
	Version         string                                          `json:"version"`
	RequireVersion  string                                          `json:"require_ver,omitempty"`
	BaseVersion     string                                          `json:"base_version,omitempty"`
	ExtraFirstPack  []string                                        `json:"extra_first_pack,omitempty"`
	ExtraPackMap    map[string]string                               `json:"extra_pack_map,omitempty"`
	PCEngineFiles   map[string]NativeEngineInfo                     `json:"pcengine_files,omitempty"`
	PCEngineURL     string                                          `json:"pcengine_url,omitempty"`
	PCEngineVersion string                                          `json:"pcengine_version,omitempty"`
	SVNRevision     int64                                           `json:"svn_revision,omitempty"`
	RestartVersions map[string]bool                                 `json:"-"`
	Files           map[string]map[string]map[string]NativeFileInfo `json:"files"`
}

type NativeResource struct {
	Path string
	Data []byte
}

type NativePackage struct {
	Version      string
	DownloadType string
	Data         []byte
	Info         NativeFileInfo
}

type NativeHandler struct {
	list  NativePatchList
	body  []byte
	files map[string]nativeServedFile
	owned []*os.File
	root  *os.Root
}

type nativeServedFile struct {
	file *os.File
	data []byte
	info NativeFileInfo
	path string
}

func (h *NativeHandler) Close() error {
	var errs []error
	if h.root != nil {
		errs = append(errs, h.root.Close())
		h.root = nil
	}
	for _, file := range h.owned {
		errs = append(errs, file.Close())
	}
	h.owned = nil
	return errors.Join(errs...)
}

func nativeHash(data []byte) string {
	digest := md5.Sum(data)
	return hex.EncodeToString(digest[:])
}

func nativeVersion(value string) ([3]uint64, error) {
	var result [3]uint64
	parts := strings.Split(value, ".")
	if len(parts) != len(result) {
		return result, errors.New("native resource version requires three numeric parts")
	}
	for index, part := range parts {
		number, err := strconv.ParseUint(part, 10, 32)
		if err != nil || strconv.FormatUint(number, 10) != part {
			return result, errors.New("invalid native resource version")
		}
		result[index] = number
	}
	return result, nil
}

func compareNativeVersion(left, right string) int {
	a, _ := nativeVersion(left)
	b, _ := nativeVersion(right)
	return slices.Compare(a[:], b[:])
}

func nativeDownloadType(value string) bool {
	return value != "" && len(value) <= 32 && !strings.ContainsFunc(value, func(character rune) bool {
		return (character < 'a' || character > 'z') && (character < '0' || character > '9')
	})
}

func nativeFilename(prefix string, pack NativePackage) string {
	return prefix + "_" + pack.DownloadType + "_" + pack.Version + "_win32_" + pack.Info.MD5 + ".zip"
}

func nativeBlobPath(directory string, info NativeFileInfo) string {
	return filepath.Join(directory, "files", info.MD5+".zip")
}

func validateNativeArchive(data io.ReaderAt, size int64) error {
	if size == 0 || size > patches.MaxFileSize {
		return errors.New("invalid native resource ZIP size")
	}
	reader, err := zip.NewReader(data, size)
	if err != nil {
		return err
	}
	if len(reader.File) > patches.MaxFiles {
		return errors.New("too many native resource entries")
	}
	seen := make(map[string]bool, len(reader.File))
	var total uint64
	for _, file := range reader.File {
		key := strings.ToLower(file.Name)
		if !patches.ValidPath(file.Name) || file.FileInfo().IsDir() || file.Mode()&os.ModeSymlink != 0 || seen[key] || file.UncompressedSize64 > uint64(patches.MaxFileSize)-total {
			return fmt.Errorf("invalid native ZIP entry %q", file.Name)
		}
		seen[key] = true
		total += file.UncompressedSize64
		stream, err := file.Open()
		if err != nil {
			return err
		}
		count, err := io.Copy(io.Discard, io.LimitReader(stream, int64(file.UncompressedSize64)+1))
		stream.Close()
		if err != nil || uint64(count) != file.UncompressedSize64 {
			return errors.New("native ZIP entry checksum or size mismatch")
		}
	}
	return nil
}

func validateNativeData(data []byte) error {
	return validateNativeArchive(bytes.NewReader(data), int64(len(data)))
}

func writeNativeArchive(writer *zip.Writer, entries []NativeResource) error {
	for _, entry := range entries {
		if !patches.ValidPath(entry.Path) || int64(len(entry.Data)) > patches.MaxFileSize {
			return errors.New("invalid native resource path or size")
		}
		size := uint64(len(entry.Data))
		header := zip.FileHeader{Name: entry.Path, Method: zip.Store, ModifiedDate: 33, CRC32: crc32.ChecksumIEEE(entry.Data), CompressedSize64: size, UncompressedSize64: size}
		header.SetMode(0644)
		file, err := writer.CreateRaw(&header)
		if err != nil {
			return err
		}
		if _, err = file.Write(entry.Data); err != nil {
			return err
		}
	}
	return nil
}

func BuildNativeResourcePackage(version, downloadType string, entries []NativeResource) (NativePackage, error) {
	pack := NativePackage{Version: version, DownloadType: downloadType}
	if _, err := nativeVersion(version); err != nil {
		return pack, err
	}
	if !nativeDownloadType(downloadType) || len(entries) > patches.MaxFiles {
		return pack, errors.New("invalid native download type or entry count")
	}
	entries = slices.SortedFunc(slices.Values(entries), func(left, right NativeResource) int { return strings.Compare(left.Path, right.Path) })
	var output bytes.Buffer
	writer := zip.NewWriter(&output)
	err := writeNativeArchive(writer, entries)
	if closeErr := writer.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return pack, err
	}
	pack.Data = output.Bytes()
	if err = validateNativeData(pack.Data); err != nil {
		return NativePackage{}, err
	}
	pack.Info = NativeFileInfo{Size: int64(len(pack.Data)), MD5: nativeHash(pack.Data)}
	return pack, nil
}

func newNativeHandler(origin, version, requireVersion string, packages []NativePackage, serve func(NativePackage) (nativeServedFile, error)) (*NativeHandler, error) {
	base, err := url.Parse(origin)
	if err != nil || base.Scheme != "http" && base.Scheme != "https" || base.Host == "" || base.User != nil || base.Path != "" && base.Path != "/" || base.RawQuery != "" || base.Fragment != "" {
		return nil, errors.New("invalid native patch origin")
	}
	if _, err = nativeVersion(version); err != nil {
		return nil, err
	}
	if _, err = nativeVersion(requireVersion); err != nil {
		return nil, err
	}
	if compareNativeVersion(requireVersion, version) > 0 || len(packages) == 0 || len(packages) > patches.MaxFiles {
		return nil, errors.New("invalid native resource release versions or package count")
	}
	base.Path = "/native"
	base.RawPath = ""
	files := make(map[string]NativeFileInfo, len(nativePrefixes)*len(packages))
	handler := &NativeHandler{files: make(map[string]nativeServedFile, len(nativePrefixes)*len(packages))}
	seen := make(map[string]bool, len(packages))
	for _, pack := range packages {
		if pack.Version != version || !nativeDownloadType(pack.DownloadType) || seen[pack.DownloadType] {
			return nil, errNativePackageMetadata
		}
		served, err := serve(pack)
		if err != nil {
			return nil, err
		}
		seen[pack.DownloadType] = true
		for _, prefix := range nativePrefixes {
			name := nativeFilename(prefix, pack)
			files[name] = pack.Info
			handler.files[name] = served
		}
	}
	handler.list = NativePatchList{Version: version, RequireVersion: requireVersion, Files: map[string]map[string]map[string]NativeFileInfo{base.String(): {version: files}}}
	if handler.body, err = json.MarshalIndent(handler.list, "", "  "); err != nil {
		return nil, err
	}
	handler.body = append(handler.body, '\n')
	return handler, nil
}

func openNativeBlob(path string, expected NativeFileInfo) (*os.File, error) {
	file, size, err := openSeed(path)
	if err != nil {
		return nil, err
	}
	if err = checkNativeBlob(file, size, path, expected); err != nil {
		file.Close()
		return nil, err
	}
	return file, nil
}

func checkNativeBlob(file *os.File, size int64, path string, expected NativeFileInfo) error {
	digest := md5.New()
	count, err := io.Copy(digest, io.LimitReader(file, patches.MaxFileSize+1))
	if err != nil {
		return err
	}
	if count != size {
		return fmt.Errorf("patch seed changed while reading %q", path)
	}
	if count != expected.Size || hex.EncodeToString(digest.Sum(nil)) != expected.MD5 {
		return errors.New("native resource archive checksum mismatch")
	}
	return validateNativeArchive(file, count)
}

func OpenNative(directory string) (*NativeHandler, error) {
	manifest := filepath.Join(directory, nativeManifestName)
	info, err := os.Lstat(manifest)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() < 1 || info.Size() > patches.MaxManifestSize {
		return nil, errors.New("invalid native patch list size")
	}
	body, err := os.ReadFile(manifest)
	if err != nil {
		return nil, err
	}
	var list NativePatchList
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(&list); err != nil {
		return nil, err
	}
	if err = decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		return nil, errors.New("invalid trailing native patch list content")
	}
	if list.PCEngineVersion != "" {
		return openRetail(directory, list)
	}
	if len(list.Files) != 1 {
		return nil, errors.New("native resource release requires one origin")
	}
	var packages []NativePackage
	var origin string
	seen := map[string]bool{}
	blobs := map[string]*os.File{}
	release := func() {
		for _, file := range blobs {
			file.Close()
		}
	}
	for key, versions := range list.Files {
		base, parseErr := url.Parse(key)
		if parseErr != nil || base.Path != "/native" {
			return nil, errors.New("invalid native resource base URL")
		}
		base.Path = ""
		origin = base.String()
		files, ok := versions[list.Version]
		if !ok || len(versions) != 1 || len(files) > patches.MaxFiles*2 {
			return nil, errors.New("invalid native resource version entries")
		}
		for name, entry := range files {
			fields := strings.Split(name, "_")
			if len(fields) != 5 || fields[0] != "packc" && fields[0] != "patch" || fields[2] != list.Version || fields[3] != "win32" || fields[4] != entry.MD5+".zip" || len(entry.MD5) != 32 || strings.ToLower(entry.MD5) != entry.MD5 {
				return nil, errors.New("invalid native resource filename")
			}
			if _, err = hex.DecodeString(entry.MD5); err != nil {
				return nil, err
			}
			if seen[fields[1]] {
				continue
			}
			if _, opened := blobs[entry.MD5]; !opened {
				file, err := openNativeBlob(nativeBlobPath(directory, entry), entry)
				if err != nil {
					release()
					return nil, err
				}
				blobs[entry.MD5] = file
			}
			packages = append(packages, NativePackage{Version: list.Version, DownloadType: fields[1], Info: entry})
			seen[fields[1]] = true
		}
	}
	handler, err := newNativeHandler(origin, list.Version, list.RequireVersion, packages, func(pack NativePackage) (nativeServedFile, error) {
		return nativeServedFile{file: blobs[pack.Info.MD5], info: pack.Info}, nil
	})
	if err != nil {
		release()
		return nil, err
	}
	for _, file := range blobs {
		handler.owned = append(handler.owned, file)
	}
	canonical, _ := json.Marshal(list)
	expected, _ := json.Marshal(handler.list)
	if !bytes.Equal(canonical, expected) {
		handler.Close()
		return nil, errors.New("native resource aliases do not match")
	}
	return handler, nil
}

func (h *NativeHandler) PatchList() NativePatchList {
	var list NativePatchList
	json.Unmarshal(h.body, &list)
	return list
}

func (h *NativeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	header := w.Header()
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		header.Set("Allow", "GET, HEAD")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	header.Set("X-Content-Type-Options", "nosniff")
	if r.URL.Path == "/native/"+nativeManifestName {
		header.Set("Content-Type", "application/json")
		header.Set("Cache-Control", "no-cache")
		http.ServeContent(w, r, nativeManifestName, time.Time{}, bytes.NewReader(h.body))
		return
	}
	name, found := strings.CutPrefix(r.URL.Path, "/native/")
	if h.root == nil {
		name, found = strings.CutPrefix(r.URL.Path, "/native/patch_files/")
	}
	entry, listed := h.files[name]
	if !found || !listed {
		http.NotFound(w, r)
		return
	}
	header.Set("Content-Type", "application/zip")
	header.Set("Cache-Control", "public, max-age=31536000, immutable")
	header.Set("ETag", `"`+entry.info.MD5+`"`)
	if entry.path != "" {
		file, err := h.root.Open(entry.path)
		if err != nil {
			http.Error(w, "patch unavailable", http.StatusServiceUnavailable)
			return
		}
		defer file.Close()
		info, err := file.Stat()
		if err != nil || !info.Mode().IsRegular() || info.Size() != entry.info.Size {
			http.Error(w, "patch changed", http.StatusServiceUnavailable)
			return
		}
		header.Set("Content-Type", "application/octet-stream")
		http.ServeContent(w, r, name, time.Time{}, io.NewSectionReader(file, 0, entry.info.Size))
		return
	}
	if entry.file == nil {
		http.ServeContent(w, r, name, time.Time{}, bytes.NewReader(entry.data))
		return
	}
	http.ServeContent(w, r, name, time.Time{}, io.NewSectionReader(entry.file, 0, entry.info.Size))
}
