package client

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"mmm-revived/internal/json"
	"mmm-revived/internal/patches"
)

const (
	nativeConfigMagic    = "[g124 great.01813388933a8fb635618a1d86cd6d97]"
	nativeConfigManifest = "native-installation.json"
	nativeConfigRestored = "native-installation.restored.json"
	nativeConfigOriginal = "native-original.bin"
	nativeConfigName     = "mcconfig"
	nativeConfigLimit    = 1 << 20
	runtimeLockName      = ".mmm-revived-update.lock"
	runtimeResources     = "patch"
	runtimeCache         = "patch_cache"
	runtimeContentMarker = "patch/.mmm-revived-content"
	backupLockName       = ".native-update.lock"
)

type NativePatchOptions struct {
	EngineManifest string
	ClientDir      string
	Runtime        string
	ServerURL      string
	Backup         string
	Content        string
	Debug          bool
}

type nativeConfiguration struct {
	Engine         *nativeEngineInstallation `json:"engine,omitempty"`
	Version        int                       `json:"version"`
	Runtime        string                    `json:"runtime"`
	Existed        bool                      `json:"existed"`
	OriginalSHA256 string                    `json:"original_sha256,omitempty"`
	SHA256         string                    `json:"sha256"`
}

func nativeBootConfig(previous []byte, serverURL string, debug bool) ([]byte, error) {
	origin, err := url.Parse(serverURL)
	if err != nil || origin.Host == "" || origin.User != nil || origin.RawQuery != "" || origin.Fragment != "" || origin.Path != "" && origin.Path != "/" || origin.Scheme != "http" && origin.Scheme != "https" {
		return nil, errors.New("invalid bootstrap server origin")
	}
	origin.Path = ""
	object := map[string]json.RawMessage{}
	if len(previous) > 0 {
		encoded, found := bytes.CutPrefix(previous, []byte("!"))
		if !found {
			return nil, errors.New("unsupported existing mcconfig encoding")
		}
		decoded, err := base64.StdEncoding.DecodeString(string(encoded))
		body, found := bytes.CutPrefix(decoded, []byte(nativeConfigMagic))
		if err != nil || !found {
			return nil, errors.New("invalid existing mcconfig")
		}
		if err = json.Unmarshal(body, &object); err != nil || object == nil {
			return nil, errors.New("invalid existing mcconfig JSON")
		}
	}
	debugValue := 0
	if debug {
		debugValue = 1
	}
	base := origin.String()
	managed := map[string]any{"debug": debugValue, "skip_patch": 0, "debug_tool": 2, "skip_maintain_notice": 1, "serverlist": base + "/serverlist", "usePatchInPC": true, "patchlist_win32": base + PatchListPath, "dont_auto_download": 0}
	for key, value := range managed {
		if object[key], err = json.Marshal(value); err != nil {
			return nil, err
		}
	}
	data, err := json.Marshal(object)
	if err != nil {
		return nil, err
	}
	return []byte("!" + base64.StdEncoding.EncodeToString(append([]byte(nativeConfigMagic), data...))), nil
}

func nativeConfigToken() (string, error) {
	var value [16]byte
	rand.Read(value[:])
	return hex.EncodeToString(value[:]), nil
}

func replaceConfigData(root *os.Root, name string, data []byte, digest, token string) error {
	return replaceConfigFile(root, name, bytes.NewReader(data), digest, int64(len(data)), 0600, token)
}

func ConfigureNativePatches(options NativePatchOptions) error {
	if options.Runtime == "" || options.Backup == "" {
		return errors.New("runtime and backup directories are required")
	}
	if _, err := nativeBootConfig(nil, options.ServerURL, options.Debug); err != nil {
		return err
	}
	runtimePath, err := filepath.Abs(options.Runtime)
	if err != nil {
		return err
	}
	if err = safeNativeDirectory(runtimePath); err != nil {
		return err
	}
	if err = os.MkdirAll(runtimePath, 0700); err != nil {
		return err
	}
	runtimeRoot, err := openLockedRoot(runtimePath, runtimeLockName)
	if err != nil {
		return err
	}
	defer runtimeRoot.Close()
	previous, err := readConfigFile(runtimeRoot.Root, nativeConfigName, nativeConfigLimit)
	existed := err == nil
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	configured, err := nativeBootConfig(previous, options.ServerURL, options.Debug)
	if err != nil {
		return err
	}
	if options.EngineManifest != "" {
		if err = EnsureClientClosed(options.ClientDir); err != nil {
			return err
		}
	}
	if err = resetRuntimeContent(runtimeRoot.Root, options.Content); err != nil {
		return err
	}
	enginePlan, err := planNativeEngine(runtimeRoot.Root, options)
	if err != nil {
		return err
	}
	if bytes.Equal(configured, previous) && enginePlan == nil {
		return markRuntimeContent(runtimeRoot.Root, options.Content)
	}
	backupPath, err := filepath.Abs(options.Backup)
	if err != nil {
		return err
	}
	if sameConfigDirectory(backupPath, runtimePath) {
		return errors.New("backup directory must differ from runtime directory")
	}
	if err = safeNativeDirectory(backupPath); err != nil {
		return err
	}
	if err = os.MkdirAll(backupPath, 0700); err != nil {
		return err
	}
	backup, err := openLockedRoot(backupPath, backupLockName)
	if err != nil {
		return err
	}
	defer backup.Close()
	installation := nativeConfiguration{Version: 1, Runtime: runtimePath, Existed: existed, SHA256: hash(configured)}
	existing, readErr := readConfigFile(backup.Root, nativeConfigManifest, nativeConfigLimit)
	if readErr == nil {
		if !bytes.Equal(configured, previous) || json.Unmarshal(existing, &installation) != nil || installation.Version != 1 || !sameConfigDirectory(installation.Runtime, runtimePath) || installation.SHA256 != hash(previous) {
			return errors.New("native configuration backup is already in use")
		}
		if installation.Existed {
			original, err := readConfigFile(backup.Root, nativeConfigOriginal, nativeConfigLimit)
			if err != nil {
				return err
			}
			if hash(original) != installation.OriginalSHA256 {
				return errors.New("native configuration backup checksum mismatch")
			}
		}
	} else if !errors.Is(readErr, os.ErrNotExist) {
		return readErr
	} else if existed {
		installation.OriginalSHA256 = hash(previous)
		if err = writeConfigBackup(backup.Root, nativeConfigOriginal, previous); err != nil {
			return err
		}
	}
	if err = backupNativeEngine(backup.Root, enginePlan, &installation); err != nil {
		return err
	}
	token, err := nativeConfigToken()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(installation, "", "  ")
	if err != nil {
		return err
	}
	current, err := readConfigFile(runtimeRoot.Root, nativeConfigName, nativeConfigLimit)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if !bytes.Equal(current, previous) || (err == nil) != existed {
		return errors.New("mcconfig changed during native patch configuration")
	}
	if err = replaceConfigData(backup.Root, nativeConfigManifest, data, hash(data), token); err != nil {
		return err
	}
	if !bytes.Equal(configured, previous) {
		if err = replaceConfigData(runtimeRoot.Root, nativeConfigName, configured, hash(configured), token); err != nil {
			return err
		}
	}
	if err = applyNativeEngine(runtimeRoot.Root, enginePlan, token); err != nil {
		var rollback error
		if !bytes.Equal(configured, previous) {
			if existed {
				rollback = replaceConfigData(runtimeRoot.Root, nativeConfigName, previous, hash(previous), token)
			} else {
				rollback = runtimeRoot.Remove(nativeConfigName)
			}
		}
		return errors.Join(err, rollback)
	}
	return markRuntimeContent(runtimeRoot.Root, options.Content)
}

func resetRuntimeContent(root *os.Root, content string) error {
	if content == "" {
		return nil
	}
	current, err := readConfigFile(root, runtimeContentMarker, nativeConfigLimit)
	if err == nil && string(current) == content {
		return nil
	}
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	for _, name := range []string{runtimeResources, runtimeCache} {
		info, err := root.Lstat(name)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return err
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("client runtime %s is not a directory", name)
		}
		if err = root.RemoveAll(name); err != nil {
			return fmt.Errorf("reset client resources: %w", err)
		}
	}
	return nil
}

func markRuntimeContent(root *os.Root, content string) error {
	if content == "" {
		return nil
	}
	if current, err := readConfigFile(root, runtimeContentMarker, nativeConfigLimit); err == nil && string(current) == content {
		return nil
	}
	if err := root.MkdirAll(runtimeResources, 0700); err != nil {
		return err
	}
	token, err := nativeConfigToken()
	if err != nil {
		return err
	}
	return replaceConfigData(root, runtimeContentMarker, []byte(content), hash([]byte(content)), token)
}

func RestoreNativePatches(backupPath string) error {
	if err := safeNativeDirectory(backupPath); err != nil {
		return err
	}
	backup, err := openLockedRoot(backupPath, backupLockName)
	if err != nil {
		return err
	}
	defer backup.Close()
	data, err := readConfigFile(backup.Root, nativeConfigManifest, nativeConfigLimit)
	if err != nil {
		return err
	}
	var installation nativeConfiguration
	if json.Unmarshal(data, &installation) != nil || installation.Version != 1 || !filepath.IsAbs(installation.Runtime) || !patches.ValidHash(installation.SHA256) || installation.Existed && !patches.ValidHash(installation.OriginalSHA256) {
		return errors.New("invalid native configuration backup")
	}
	if err = safeNativeDirectory(installation.Runtime); err != nil {
		return err
	}
	runtimeRoot, err := openLockedRoot(installation.Runtime, runtimeLockName)
	if err != nil {
		return err
	}
	defer runtimeRoot.Close()
	current, err := readConfigFile(runtimeRoot.Root, nativeConfigName, nativeConfigLimit)
	currentExists := err == nil
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if currentExists {
		if digest := hash(current); digest != installation.SHA256 && (!installation.Existed || digest != installation.OriginalSHA256) {
			return errors.New("mcconfig changed after native configuration")
		}
	} else if installation.Existed {
		return errors.New("native mcconfig is missing")
	}
	enginePlan, err := planNativeEngineRestore(runtimeRoot.Root, backup.Root, installation.Engine)
	if err != nil {
		return err
	}
	token, err := nativeConfigToken()
	if err != nil {
		return err
	}
	var originalConfig []byte
	if installation.Existed {
		originalConfig, err = readConfigFile(backup.Root, nativeConfigOriginal, nativeConfigLimit)
		if err != nil {
			return err
		}
		if hash(originalConfig) != installation.OriginalSHA256 {
			return errors.New("native configuration backup checksum mismatch")
		}
	}
	if err = applyNativeEngine(runtimeRoot.Root, enginePlan, token); err != nil {
		return err
	}
	if installation.Existed {
		original := originalConfig
		if !bytes.Equal(current, original) {
			if err = replaceConfigData(runtimeRoot.Root, nativeConfigName, original, installation.OriginalSHA256, token); err != nil {
				return err
			}
		}
	} else if currentExists {
		if err = runtimeRoot.Remove(nativeConfigName); err != nil {
			return err
		}
	}
	if err = backup.Rename(nativeConfigManifest, nativeConfigRestored); err != nil {
		return fmt.Errorf("record native configuration restoration: %w", err)
	}
	return nil
}
