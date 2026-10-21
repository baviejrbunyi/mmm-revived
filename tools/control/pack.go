package main

import (
	"archive/zip"
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"mmm-revived/internal/client"
	"mmm-revived/internal/patchserver"
)

const contentFolder = "content"

func findContent(directory string) (string, error) {
	if regular(filepath.Join(directory, patchserver.ContentManifest)) {
		return directory, nil
	}
	archives, err := filepath.Glob(filepath.Join(directory, "*.zip"))
	if err != nil || len(archives) == 0 {
		return "", err
	}
	if len(archives) == 1 {
		return archives[0], nil
	}
	var matching []string
	for _, archive := range archives {
		if strings.Contains(filepath.Base(archive), client.Supported.ContentVersion) {
			matching = append(matching, archive)
		}
	}
	if len(matching) == 1 {
		return matching[0], nil
	}
	return "", fmt.Errorf("%s has several content zips; keep only the one for game content %s", directory, client.Supported.ContentVersion)
}

func (w workflow) packContent(ctx context.Context, folder string) error {
	directory, err := filepath.Abs(folder)
	if err != nil {
		return err
	}
	data, err := os.ReadFile(filepath.Join(directory, patchserver.ContentManifest))
	if errors.Is(err, os.ErrNotExist) {
		return errors.New("game content is missing; run just mirror first")
	}
	if err != nil {
		return err
	}
	_, files, err := patchserver.ParseContent(data, client.Supported)
	if err != nil {
		return err
	}
	name := "MMM-" + client.Supported.ContentVersion + ".zip"
	target := filepath.Join(w.root, name)
	temporary := target + ".tmp"
	output, err := os.Create(temporary)
	if err != nil {
		return err
	}
	defer os.Remove(temporary)
	archive := zip.NewWriter(output)
	if err = packEntry(archive, path.Join(contentFolder, patchserver.ContentManifest), filepath.Join(directory, patchserver.ContentManifest), ""); err != nil {
		output.Close()
		return err
	}
	fmt.Fprintf(w.output, "Packing %d content files into %s...\n", len(files), target)
	for index, file := range files {
		if err = ctx.Err(); err != nil {
			break
		}
		source := filepath.Join(directory, "patch_files", file.Name)
		if err = packEntry(archive, path.Join(contentFolder, "patch_files", file.Name), source, file.Info.MD5); err != nil {
			break
		}
		if (index+1)%25 == 0 || index+1 == len(files) {
			fmt.Fprintf(w.output, "%d/%d files\n", index+1, len(files))
		}
	}
	if err == nil {
		err = archive.Close()
	}
	if closeErr := output.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	if err = os.Rename(temporary, target); err != nil {
		return err
	}
	fmt.Fprintf(w.output, "Content archive is ready: %s\nPut it in the content folder of any mmm-revived copy, then run just configure \"<client directory>\".\n", target)
	return nil
}

func packEntry(archive *zip.Writer, name, source, expected string) error {
	input, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("content is incomplete; run just mirror again: %w", err)
	}
	defer input.Close()
	stat, err := input.Stat()
	if err != nil {
		return err
	}
	header, err := zip.FileInfoHeader(stat)
	if err != nil {
		return err
	}
	header.Name, header.Method = name, zip.Store
	entry, err := archive.CreateHeader(header)
	if err != nil {
		return err
	}
	digest := md5.New()
	if _, err = io.Copy(io.MultiWriter(entry, digest), input); err != nil {
		return err
	}
	if expected != "" && hex.EncodeToString(digest.Sum(nil)) != expected {
		return fmt.Errorf("content file %s is damaged; run just mirror again", filepath.Base(source))
	}
	return nil
}
