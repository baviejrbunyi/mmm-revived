package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"mmm-revived/internal/config"
	"mmm-revived/internal/gate"
)

func main() {
	out := flag.String("out", "build", "output directory")
	targetOS := flag.String("os", runtime.GOOS, "target operating system")
	targetArch := flag.String("arch", runtime.GOARCH, "target architecture")
	prepare := flag.Bool("prepare-only", false, "prepare configuration and keys without compiling")
	flag.Parse()
	if *prepare {
		root, err := moduleRoot()
		if err == nil {
			err = prepareEnvironment(root, *out, true)
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, "setup failed:", err)
			os.Exit(1)
		}
		return
	}
	if err := build(*out, *targetOS, *targetArch); err != nil {
		fmt.Fprintln(os.Stderr, "build failed:", err)
		os.Exit(1)
	}
}

func build(out, targetOS, targetArch string) error {
	root, err := moduleRoot()
	if err != nil {
		return err
	}
	if !filepath.IsAbs(out) {
		out = filepath.Join(root, out)
	}
	if err = validateBuildDestination(root, out); err != nil {
		return err
	}
	if err = os.MkdirAll(out, 0755); err != nil {
		return err
	}
	if err = prepareEnvironment(root, out, false); err != nil {
		return err
	}
	executable := "server"
	if targetOS == "windows" {
		executable += ".exe"
	}
	compile := exec.Command("go", "build", "-trimpath", "-o", filepath.Join(out, executable), "./cmd/server")
	compile.Dir = root
	compile.Env = append(os.Environ(), "GOOS="+targetOS, "GOARCH="+targetArch, "CGO_ENABLED=0")
	compile.Stdout, compile.Stderr = os.Stdout, os.Stderr
	if err = compile.Run(); err != nil {
		return fmt.Errorf("compile server: %w", err)
	}
	if err = replaceTree(filepath.Join(root, config.AssetsDir), filepath.Join(out, config.AssetsDir)); err != nil {
		return fmt.Errorf("copy assets: %w", err)
	}
	fmt.Printf("server ready in %s\nrun %s\n", out, filepath.Join(out, executable))
	return nil
}

func prepareEnvironment(root, out string, generate bool) error {
	if !filepath.IsAbs(out) {
		out = filepath.Join(root, out)
	}
	if err := validateBuildDestination(root, out); err != nil {
		return err
	}
	c, err := config.Load(filepath.Join(out, config.FileName))
	if err != nil {
		return err
	}
	if !generate && !exists(c.PrivateKey()) {
		return errors.New("connection keys are missing; run just configure first")
	}

	_, err = gate.LoadKey(c.PrivateKey(), c.PublicKey())
	return err
}

func moduleRoot() (string, error) {
	output, err := exec.Command("go", "env", "GOMOD").Output()
	if err != nil {
		return "", fmt.Errorf("locate module: %w", err)
	}
	module := strings.TrimSpace(string(output))
	if module == "" || module == os.DevNull {
		return "", errors.New("run the build tool from inside the mmm-revived module")
	}
	return filepath.Dir(module), nil
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func replaceTree(source, target string) error {
	if err := separateTrees(source, target); err != nil {
		return err
	}
	if err := os.RemoveAll(target); err != nil {
		return err
	}
	if err := os.MkdirAll(target, 0755); err != nil {
		return err
	}
	entries, err := os.ReadDir(source)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.Name() != config.DataTablesDir {
			continue
		}
		if entry.IsDir() {
			err = os.CopyFS(filepath.Join(target, entry.Name()), os.DirFS(filepath.Join(source, entry.Name())))
		} else {
			err = copyFile(filepath.Join(source, entry.Name()), filepath.Join(target, entry.Name()))
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func copyFile(source, target string) error {
	if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
		return err
	}
	from, err := os.Open(source)
	if err != nil {
		return err
	}
	defer from.Close()
	info, err := from.Stat()
	if err != nil {
		return err
	}
	to, err := os.CreateTemp(filepath.Dir(target), ".copy-*")
	if err != nil {
		return err
	}
	defer os.Remove(to.Name())
	if err = to.Chmod(info.Mode().Perm()); err != nil {
		to.Close()
		return err
	}
	_, err = io.Copy(to, from)
	if closeErr := to.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	return os.Rename(to.Name(), target)
}
