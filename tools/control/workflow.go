package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"time"

	"mmm-revived/internal/client"
	"mmm-revived/internal/config"
	"mmm-revived/internal/gate"
	"mmm-revived/internal/json"
	"mmm-revived/internal/patchserver"
)

type workflow struct {
	root    string
	output  io.Writer
	execute func(context.Context, string, string, ...string) error
}

type setupState struct {
	Client string `json:"client"`
}

type buildState struct {
	Fingerprint string `json:"fingerprint"`
}

func (w workflow) configure(ctx context.Context, directory, content string, rotate bool) error {
	absolute, err := filepath.Abs(directory)
	if err != nil {
		return err
	}
	if content != "" {
		if content, err = filepath.Abs(content); err != nil {
			return err
		}
	}
	if err = client.EnsureClientClosed(absolute); err != nil {
		return err
	}
	if _, err = client.DetectBuild(absolute); err != nil {
		return err
	}
	unlock, err := w.lock()
	if err != nil {
		return err
	}
	defer unlock()
	c, err := w.environment(ctx, true)
	if err != nil {
		if c, err = config.Load(filepath.Join(w.root, "build", config.FileName)); err != nil {
			return err
		}
		fmt.Fprintln(w.output, "The existing connection keys are missing or invalid.")
		rotate = true
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	runtimePath, err := client.RuntimeDirectory(home, absolute)
	if err != nil {
		return err
	}
	backup := filepath.Join(w.root, "build", "backups", filepath.Base(runtimePath), time.Now().UTC().Format("20060102T150405.000000000"))
	publish := func(staged config.Config) error {
		if err := w.ensurePatches(ctx, staged, absolute, content); err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		return client.ConfigureNativePatches(client.NativePatchOptions{
			Runtime: runtimePath, ClientDir: absolute,
			Backup: backup, Debug: false, ServerURL: "http://" + c.HTTPAddress(),
			EngineManifest: filepath.Join(c.Patches(), "patchlist_win32.json"), Content: client.Supported.ContentMarker(),
		})
	}
	if rotate {
		fmt.Fprintln(w.output, "Generating a new connection key pair...")
		err = w.rotateKeys(c, backup, publish)
	} else {
		fmt.Fprintln(w.output, "Reusing the existing connection key pair...")
		err = publish(c)
	}
	if err != nil {
		if regular(filepath.Join(backup, "native-installation.json")) {
			err = errors.Join(err, client.RestoreNativePatches(backup))
		}
		return err
	}
	if err = writeJSON(filepath.Join(w.root, "build", "setup.json"), setupState{Client: absolute}); err != nil {
		return err
	}
	if err = patchserver.PruneRetail(c.Patches()); err != nil {
		return fmt.Errorf("configured successfully, but obsolete patch cleanup failed: %w", err)
	}
	fmt.Fprintf(w.output, "Configured %s.\nStop any previously running server, then run just run.\nOpen the game and accept any download and restart prompts.\n", absolute)
	fmt.Fprintln(w.output, "Configuration backup:", backup)
	return nil
}

func (w workflow) run(ctx context.Context) error {
	unlock, err := w.lock()
	if err != nil {
		return err
	}
	locked := true
	defer func() {
		if locked {
			unlock()
		}
	}()
	c, err := w.environment(ctx, false)
	if err != nil {
		return err
	}
	if err = available(c.Address(), c.HTTPAddress()); err != nil {
		return fmt.Errorf("server ports are already in use; stop the existing server before running just run: %w", err)
	}
	if !w.patchesReady(c.Patches(), c) {
		return errors.New("client patches are missing or outdated; close the game and run just configure \"<client directory>\"")
	}
	if err = patchserver.PruneRetail(c.Patches()); err != nil {
		return err
	}
	fingerprint, err := w.fingerprint(c)
	if err != nil {
		return err
	}
	var state buildState
	_ = readJSON(filepath.Join(w.root, "build", ".setup.json"), &state)
	executable := filepath.Join(w.root, "build", "server")
	if runtime.GOOS == "windows" {
		executable += ".exe"
	}
	ready := state.Fingerprint == fingerprint && regular(executable) && regular(c.Catalog())
	if !ready {
		fmt.Fprintln(w.output, "Building server and preparing build/...")
		if err = w.execute(ctx, w.root, "go", "run", "./tools/build"); err != nil {
			return fmt.Errorf("build server: %w", err)
		}
		if !regular(executable) || !regular(c.Catalog()) || !w.patchesReady(c.Patches(), c) {
			return errors.New("build output is incomplete")
		}
		if err = writeJSON(filepath.Join(w.root, "build", ".setup.json"), buildState{Fingerprint: fingerprint}); err != nil {
			return err
		}
	} else {
		fmt.Fprintln(w.output, "Using existing build.")
	}
	unlock()
	locked = false
	fmt.Fprintln(w.output, "Starting server. Press Ctrl+C to stop.")
	return w.execute(ctx, filepath.Join(w.root, "build"), executable, "-config", filepath.Join(w.root, "build", config.FileName))
}

func (w workflow) environment(ctx context.Context, create bool) (config.Config, error) {
	path := filepath.Join(w.root, "build", config.FileName)
	if config.Exists(path) {
		c, err := config.Load(path)
		if err != nil {
			return c, err
		}
		if regular(c.PrivateKey()) {
			_, err = gate.LoadKey(c.PrivateKey(), c.PublicKey())
			return c, err
		}
	}
	if !create {
		return config.Config{}, errors.New("connection keys are missing; run just configure \"<client directory>\" first")
	}
	fmt.Fprintln(w.output, "Preparing configuration and connection keys...")
	if err := w.execute(ctx, w.root, "go", "run", "./tools/build", "-prepare-only"); err != nil {
		return config.Config{}, err
	}
	c, err := config.Load(path)
	if err != nil {
		return c, err
	}
	_, err = gate.LoadKey(c.PrivateKey(), c.PublicKey())
	return c, err
}

func (w workflow) patchesReady(directory string, c config.Config) bool {
	catalog, err := patchserver.OpenNativeCatalog(directory)
	if err != nil {
		return false
	}
	defer catalog.Close()
	list := catalog.PatchList()
	if list.BaseVersion != client.Supported.BaseVersion || !list.RestartVersions[list.Version] || len(list.ExtraFirstPack) == 0 {
		return false
	}
	for _, versions := range list.Files {
		if len(versions[client.Supported.ContentVersion]) == 0 {
			return false
		}
	}
	key, err := os.ReadFile(c.PublicKey())
	if err != nil || catalog.ConfigureRetail("http://"+c.HTTPAddress(), key) != nil {
		return false
	}
	dll, err := os.ReadFile(filepath.Join(directory, "pcengine_files", list.PCEngineVersion, "xlua.dll"))
	if err != nil {
		return false
	}
	profile := client.Supported
	for _, file := range profile.Modules() {
		module, err := os.ReadFile(filepath.Join(w.root, "assets", filepath.FromSlash(file.Path)))
		if err != nil || len(module) == 0 || !bytes.Contains(dll, module) {
			return false
		}
	}
	var resources []string
	for _, file := range profile.Resources() {
		resources = append(resources, file.Path)
	}
	return patchScriptsMatch(directory, list, filepath.Join(w.root, "assets"), resources...)
}

func (w workflow) ensurePatches(ctx context.Context, c config.Config, directory, content string) error {
	patches := c.Patches()
	if w.patchesReady(patches, c) {
		return nil
	}
	var saved setupState
	if directory == "" {
		if err := readJSON(filepath.Join(w.root, "build", "setup.json"), &saved); err == nil {
			directory = saved.Client
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	if !regular(filepath.Join(patches, "source", "engine", "manifest.json")) && directory == "" {
		return errors.New("client setup is missing; run just configure \"<client directory>\" first")
	}
	assets := filepath.Join(w.root, "assets")
	seed, err := os.ReadFile(filepath.Join(assets, "scripts", "scripts.pck"))
	if err != nil {
		return fmt.Errorf("read packaged client patch: %w", err)
	}
	if client.Digest(seed) != client.Supported.PatchedScriptsSHA256 {
		return errors.New("packaged scripts bundle does not match the approved client patch")
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	resourceVersion, err := nextVersion(client.Supported.ContentVersion)
	if err != nil {
		return err
	}
	engineVersion := resourceVersion
	var previous patchserver.NativePatchList
	if err = readJSON(filepath.Join(patches, "patchlist_win32.json"), &previous); err == nil {
		if resourceVersion, err = nextVersion(laterVersion(previous.Version, client.Supported.ContentVersion)); err != nil {
			return err
		}
		if engineVersion, err = nextVersion(previous.PCEngineVersion); err != nil {
			return err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("read existing patch release: %w", err)
	}
	resourceVersion, engineVersion, err = unusedVersions(patches, resourceVersion, engineVersion)
	if err != nil {
		return err
	}
	if regular(filepath.Join(patches, "source", "engine", "manifest.json")) {
		directory = ""
	}
	fmt.Fprintln(w.output, "Publishing the native client update...")
	_, err = patchserver.PrepareRetail(patchserver.RetailPrepareOptions{
		Client: directory, Directory: patches, Origin: "http://" + c.HTTPAddress(),
		Scripts: filepath.Join(assets, "scripts", "scripts.pck"), PublicKey: c.PublicKey(), Modules: assets,
		Version: resourceVersion, EngineVersion: engineVersion, Content: content,
	})
	return err
}

func parseVersion(value string) ([3]uint64, error) {
	var numbers [3]uint64
	parts := strings.Split(value, ".")
	if len(parts) != 3 {
		return numbers, errors.New("invalid patch version")
	}
	for index, part := range parts {
		number, err := strconv.ParseUint(part, 10, 32)
		if err != nil || strconv.FormatUint(number, 10) != part {
			return numbers, errors.New("invalid patch version")
		}
		numbers[index] = number
	}
	return numbers, nil
}

func laterVersion(left, right string) string {
	first, err := parseVersion(left)
	if err != nil {
		return right
	}
	second, err := parseVersion(right)
	if err != nil || slices.Compare(first[:], second[:]) >= 0 {
		return left
	}
	return right
}

func nextVersion(value string) (string, error) {
	numbers, err := parseVersion(value)
	if err != nil {
		return "", err
	}
	base, err := parseVersion(client.Supported.BaseVersion)
	if err != nil {
		return "", err
	}
	if slices.Compare(numbers[:], base[:]) < 0 {
		numbers = base
	}
	if numbers[2] == 1<<32-1 {
		return "", errors.New("patch revision is exhausted")
	}
	return fmt.Sprintf("%d.%d.%d", numbers[0], numbers[1], numbers[2]+1), nil
}

func available(addresses ...string) error {
	var listeners []net.Listener
	defer func() {
		for _, listener := range listeners {
			listener.Close()
		}
	}()
	for _, address := range addresses {
		listener, err := net.Listen("tcp", address)
		if err != nil {
			return err
		}
		listeners = append(listeners, listener)
	}
	return nil
}

func (w workflow) lock() (func(), error) {
	path := filepath.Join(w.root, "build", ".configure.lock")
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, err
	}
	return acquireSetupLock(path)
}

func regular(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}

func readJSON(path string, target any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}

func writeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return writeAtomic(path, append(data, '\n'))
}

func writeAtomic(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".setup-*")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	_, err = file.Write(data)
	if closeErr := file.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	return os.Rename(file.Name(), path)
}
