package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		if exit, ok := errors.AsType[*exec.ExitError](err); ok && exit.ExitCode() > 0 {
			os.Exit(exit.ExitCode())
		}
		os.Exit(1)
	}
}

func run() error {
	if len(os.Args) < 2 {
		return errors.New("use just mirror [\"<folder>\"], just pack-content [\"<folder>\"], just configure \"<client directory>\" [\"<folder>\"], just rotate-keys \"<client directory>\", just join \"<client directory>\" <server address>, just run or just reset")
	}
	root, err := os.Getwd()
	if err != nil {
		return err
	}
	for {
		if _, err = os.Stat(filepath.Join(root, "go.mod")); err == nil {
			break
		}
		parent := filepath.Dir(root)
		if parent == root {
			return errors.New("run these commands from the mmm-revived repository")
		}
		root = parent
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	w := workflow{root: root, output: os.Stdout, execute: execute}
	switch os.Args[1] {
	case "configure", "rotate-keys":
		path, content := os.Getenv("client"), os.Getenv("content")
		if len(os.Args) >= 3 {
			path = os.Args[2]
		}
		if len(os.Args) == 4 && os.Args[1] == "configure" {
			content = os.Args[3]
		}
		if path == "" || len(os.Args) > 4 || len(os.Args) == 4 && os.Args[1] != "configure" {
			return fmt.Errorf("use just %s \"<client directory>\"", os.Args[1])
		}
		if os.Args[1] == "rotate-keys" {
			content = ""
		} else if content == "" {
			if content, err = findContent(filepath.Join(root, contentFolder)); err != nil {
				return err
			}
		}
		return w.configure(ctx, path, content, os.Args[1] == "rotate-keys")
	case "mirror":
		folder := os.Getenv("folder")
		if len(os.Args) == 3 {
			folder = os.Args[2]
		}
		if len(os.Args) > 3 {
			return errors.New("use just mirror [\"<folder>\"]")
		}
		if folder == "" {
			folder = filepath.Join(root, contentFolder)
		}
		return w.mirror(ctx, folder)
	case "pack-content":
		folder := os.Getenv("folder")
		if len(os.Args) == 3 {
			folder = os.Args[2]
		}
		if len(os.Args) > 3 {
			return errors.New("use just pack-content [\"<folder>\"]")
		}
		if folder == "" {
			folder = filepath.Join(root, contentFolder)
		}
		return w.packContent(ctx, folder)
	case "join":
		path, server := os.Getenv("client"), os.Getenv("server")
		if len(os.Args) == 4 {
			path, server = os.Args[2], os.Args[3]
		}
		if path == "" || server == "" {
			return errors.New("use just join \"<client directory>\" http://<host>:18080")
		}
		return w.join(ctx, path, server)
	case "run":
		if len(os.Args) != 2 {
			return errors.New("use just run")
		}
		return w.run(ctx)
	case "reset":
		if len(os.Args) != 2 {
			return errors.New("use just reset")
		}
		return w.reset()
	default:
		return errors.New("use just mirror [\"<folder>\"], just pack-content [\"<folder>\"], just configure \"<client directory>\" [\"<folder>\"], just rotate-keys \"<client directory>\", just join \"<client directory>\" <server address>, just run or just reset")
	}
}

func execute(ctx context.Context, directory, program string, args ...string) error {
	command := exec.CommandContext(ctx, program, args...)
	command.Dir = directory
	command.Stdin, command.Stdout, command.Stderr = os.Stdin, os.Stdout, os.Stderr
	command.Env = append(os.Environ(), "CGO_ENABLED=0")
	return command.Run()
}
