package client

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"mmm-revived/internal/json"
	"mmm-revived/internal/patches"
)

const nativeEnginePath = "patch/version.json"
const nativeEngineOriginal = "native-engine-original.bin"

func safeNativeDirectory(directory string) error {
	absolute, err := filepath.Abs(directory)
	if err != nil {
		return err
	}
	for {
		info, err := os.Lstat(absolute)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		if err == nil && (info.Mode()&os.ModeSymlink != 0 || !info.IsDir()) {
			return fmt.Errorf("native configuration directory is not a regular directory: %s", absolute)
		}
		parent := filepath.Dir(absolute)
		if parent == absolute {
			return nil
		}
		absolute = parent
	}
}

type nativeEngineFile struct {
	MD5  string
	Size int64
}

func (f nativeEngineFile) MarshalJSON() ([]byte, error) { return json.Marshal([2]any{f.MD5, f.Size}) }

func (f *nativeEngineFile) UnmarshalJSON(data []byte) error {
	var fields []json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil || len(fields) != 2 {
		return errors.New("engine file requires MD5 and size")
	}
	if err := json.Unmarshal(fields[0], &f.MD5); err != nil {
		return err
	}
	return json.Unmarshal(fields[1], &f.Size)
}

type nativeEngineCatalog struct {
	Version       string                      `json:"version"`
	BaseVersion   string                      `json:"base_version"`
	EngineVersion string                      `json:"pcengine_version"`
	Files         map[string]nativeEngineFile `json:"pcengine_files"`
}

type nativeEngineInstallation struct {
	Existed        bool   `json:"existed"`
	OriginalSHA256 string `json:"original_sha256,omitempty"`
	SHA256         string `json:"sha256"`
}

type nativeEnginePlan struct {
	before  []byte
	after   []byte
	existed bool
}

func engineVersion(value string) ([3]uint64, error) {
	var result [3]uint64
	parts := strings.Split(value, ".")
	if len(parts) != 3 {
		return result, errors.New("invalid engine catalog version")
	}
	for i, part := range parts {
		n, err := strconv.ParseUint(part, 10, 32)
		if err != nil || strconv.FormatUint(n, 10) != part {
			return result, errors.New("invalid engine catalog version")
		}
		result[i] = n
	}
	return result, nil
}

func readEngineCatalog(path, clientDir string) (nativeEngineCatalog, error) {
	var catalog nativeEngineCatalog
	if clientDir == "" {
		return catalog, errors.New("client directory is required for native engine metadata")
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return catalog, err
	}
	if err = safeNativeDirectory(filepath.Dir(absolute)); err != nil {
		return catalog, err
	}
	if err = safeNativeDirectory(clientDir); err != nil {
		return catalog, err
	}
	root, err := os.OpenRoot(filepath.Dir(absolute))
	if err != nil {
		return catalog, err
	}
	defer root.Close()
	data, err := readConfigFile(root, filepath.Base(absolute), nativeConfigLimit)
	if err != nil {
		return catalog, err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	if err = decoder.Decode(&catalog); err != nil {
		return catalog, err
	}
	if err = decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		return catalog, errors.New("trailing native engine catalog content")
	}
	base, err := engineVersion(catalog.BaseVersion)
	if err != nil || catalog.BaseVersion != Supported.BaseVersion {
		return catalog, errors.New("engine catalog does not match the installed client base version")
	}
	for _, version := range []string{catalog.Version, catalog.EngineVersion} {
		value, err := engineVersion(version)
		if err != nil || value[0] < base[0] || value[0] == base[0] && value[1] < base[1] || value[0] == base[0] && value[1] == base[1] && value[2] <= base[2] {
			return catalog, errors.New("engine catalog version must exceed the installed client base version")
		}
	}
	if len(catalog.Files) < 3 || len(catalog.Files) > 4096 {
		return catalog, errors.New("native engine catalog must contain a complete file inventory")
	}
	seen := map[string]bool{}
	for name, entry := range catalog.Files {
		decoded, err := hex.DecodeString(entry.MD5)
		folded := strings.ToLower(name)
		if !patches.ValidPath(name) || seen[folded] || err != nil || len(decoded) != 16 || strings.ToLower(entry.MD5) != entry.MD5 || entry.Size < 0 || entry.Size > 2<<30 {
			return catalog, fmt.Errorf("invalid native engine file %q", name)
		}
		seen[folded] = true
	}
	for _, name := range []string{patches.Executable, "GameAssembly.dll", "xlua.dll"} {
		if entry, known := catalog.Files[name]; !known || entry.Size <= 0 {
			return catalog, fmt.Errorf("native engine catalog is missing %s", name)
		}
	}
	clientRoot, err := os.OpenRoot(clientDir)
	if err != nil {
		return catalog, err
	}
	defer clientRoot.Close()
	err = fs.WalkDir(clientRoot.FS(), ".", func(name string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if name == "." {
			return nil
		}
		lower := strings.ToLower(entry.Name())
		if entry.IsDir() && (lower == "streamingassets" || lower == "backup_files") {
			return fs.SkipDir
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("symbolic link in client engine path: %s", name)
		}
		if entry.IsDir() || lower == "launcher.exe" {
			return nil
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("invalid client engine file: %s", name)
		}
		if seen[strings.ToLower(name)] {
			if _, known := catalog.Files[name]; !known {
				return fmt.Errorf("native engine filename casing differs from installed file: %s", name)
			}
		}
		if !seen[strings.ToLower(name)] {
			if err := configSafePath(clientRoot, name, false, false); err != nil {
				return err
			}
			file, err := clientRoot.Open(name)
			if err != nil {
				return err
			}
			digest := md5.New()
			size, err := io.Copy(digest, io.LimitReader(file, (2<<30)+1))
			file.Close()
			if err != nil {
				return err
			}
			if size > 2<<30 || len(catalog.Files) >= 4096 {
				return errors.New("installed engine inventory exceeds native limits")
			}
			catalog.Files[name] = nativeEngineFile{MD5: hex.EncodeToString(digest.Sum(nil)), Size: size}
			seen[strings.ToLower(name)] = true
		}
		return nil
	})
	return catalog, err
}

func planNativeEngine(root *os.Root, options NativePatchOptions) (*nativeEnginePlan, error) {
	if options.EngineManifest == "" {
		return nil, nil
	}
	catalog, err := readEngineCatalog(options.EngineManifest, options.ClientDir)
	if err != nil {
		return nil, err
	}
	before, err := readConfigFile(root, nativeEnginePath, nativeConfigLimit)
	existed := err == nil
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	object := map[string]json.RawMessage{}
	if existed {
		if err = json.Unmarshal(before, &object); err != nil || object == nil {
			return nil, errors.New("invalid native patch version metadata")
		}
	}
	var current map[string]nativeEngineFile
	var branch, appVersion string
	json.Unmarshal(object["branch"], &branch)
	json.Unmarshal(object["app_version"], &appVersion)
	identified := branch == Supported.Branch && appVersion == Supported.BaseVersion
	if identified && json.Unmarshal(object["pcengine_files"], &current) == nil && reflect.DeepEqual(current, catalog.Files) {
		return nil, nil
	}
	object["pcengine_files"], err = json.Marshal(catalog.Files)
	if err != nil {
		return nil, err
	}
	if !identified {
		if object["branch"], err = json.Marshal(Supported.Branch); err != nil {
			return nil, err
		}
		if object["app_version"], err = json.Marshal(Supported.BaseVersion); err != nil {
			return nil, err
		}
	}
	after, err := json.Marshal(object)
	if err != nil {
		return nil, err
	}
	if len(after) > nativeConfigLimit {
		return nil, errors.New("native engine metadata exceeds size limit")
	}
	return &nativeEnginePlan{before: before, after: after, existed: existed}, nil
}

func backupNativeEngine(backup *os.Root, plan *nativeEnginePlan, installation *nativeConfiguration) error {
	if plan == nil {
		return nil
	}
	if installation.Engine != nil {
		return errors.New("native engine metadata is already configured; restore it before selecting a different engine catalog")
	}
	state := &nativeEngineInstallation{Existed: plan.existed, SHA256: hash(plan.after)}
	if plan.existed {
		state.OriginalSHA256 = hash(plan.before)
		if err := writeConfigBackup(backup, nativeEngineOriginal, plan.before); err != nil {
			return err
		}
	}
	installation.Engine = state
	return nil
}

func planNativeEngineRestore(runtime, backup *os.Root, installation *nativeEngineInstallation) (*nativeEnginePlan, error) {
	if installation == nil {
		return nil, nil
	}
	if !patches.ValidHash(installation.SHA256) || installation.Existed && !patches.ValidHash(installation.OriginalSHA256) {
		return nil, errors.New("invalid native engine backup metadata")
	}
	current, err := readConfigFile(runtime, nativeEnginePath, nativeConfigLimit)
	exists := err == nil
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	if exists && hash(current) != installation.SHA256 && (!installation.Existed || hash(current) != installation.OriginalSHA256) || !exists && installation.Existed {
		return nil, errors.New("native engine metadata changed after configuration; refusing to overwrite patch progress")
	}
	var original []byte
	if installation.Existed {
		original, err = readConfigFile(backup, nativeEngineOriginal, nativeConfigLimit)
		if err != nil {
			return nil, err
		}
		if hash(original) != installation.OriginalSHA256 {
			return nil, errors.New("native engine metadata backup checksum mismatch")
		}
	}
	return &nativeEnginePlan{before: current, after: original, existed: exists}, nil
}

func applyNativeEngine(root *os.Root, plan *nativeEnginePlan, token string) error {
	if plan == nil || bytes.Equal(plan.before, plan.after) {
		return nil
	}
	current, err := readConfigFile(root, nativeEnginePath, nativeConfigLimit)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if !bytes.Equal(current, plan.before) || (err == nil) != plan.existed {
		return errors.New("native engine metadata changed during configuration")
	}
	if plan.after == nil {
		if !plan.existed {
			return nil
		}
		return root.Remove(nativeEnginePath)
	}
	return replaceConfigData(root, nativeEnginePath, plan.after, hash(plan.after), token)
}
