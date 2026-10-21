package main

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"mmm-revived/internal/client"
	"mmm-revived/internal/patchserver"
)

const mirrorWorkers = 4

func (w workflow) mirror(ctx context.Context, directory string) error {
	absolute, err := filepath.Abs(directory)
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Join(absolute, "patch_files"), 0755); err != nil {
		return err
	}
	data, err := download(ctx, client.Supported.ContentList, 8<<20)
	if err != nil {
		return fmt.Errorf("fetch the game patch list: %w", err)
	}
	_, files, err := patchserver.ParseContent(data, client.Supported)
	if err != nil {
		return err
	}
	if err = writeAtomic(filepath.Join(absolute, patchserver.ContentManifest), data); err != nil {
		return err
	}
	slices.SortFunc(files, func(left, right patchserver.ContentFile) int {
		return int(left.Info.Size - right.Info.Size)
	})
	var total int64
	for _, file := range files {
		total += file.Info.Size
	}
	fmt.Fprintf(w.output, "Mirroring %d files (%.1f GB) of game content %s into %s...\n", len(files), float64(total)/1e9, client.Supported.ContentVersion, absolute)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	jobs := make(chan patchserver.ContentFile)
	var done, bytesDone atomic.Int64
	var failures []error
	var mutex sync.Mutex
	var group sync.WaitGroup
	for range mirrorWorkers {
		group.Go(func() {
			for file := range jobs {
				if err := mirrorFile(ctx, absolute, file); err != nil {
					mutex.Lock()
					failures = append(failures, err)
					mutex.Unlock()
					cancel()
					continue
				}
				count := done.Add(1)
				size := bytesDone.Add(file.Info.Size)
				if count%25 == 0 || count == int64(len(files)) {
					fmt.Fprintf(w.output, "%d/%d files, %.2f/%.2f GB\n", count, len(files), float64(size)/1e9, float64(total)/1e9)
				}
			}
		})
	}
	for _, file := range files {
		select {
		case jobs <- file:
		case <-ctx.Done():
		}
	}
	close(jobs)
	group.Wait()
	if len(failures) > 0 {
		return errors.Join(failures...)
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	if absolute == filepath.Join(w.root, contentFolder) {
		fmt.Fprintln(w.output, "Game content is ready. Close the game, then run just configure \"<client directory>\"")
		return nil
	}
	fmt.Fprintf(w.output, "Game content is ready. Close the game, then run just configure \"<client directory>\" \"%s\"\n", absolute)
	return nil
}

func download(ctx context.Context, address string, limit int64) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		return nil, err
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d from %s", response.StatusCode, address)
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, errors.New("response is too large")
	}
	return data, nil
}

func fileMD5(path string, size int64) (string, bool) {
	file, err := os.Open(path)
	if err != nil {
		return "", false
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() != size {
		return "", false
	}
	digest := md5.New()
	if read, err := io.Copy(digest, file); err != nil || read != size {
		return "", false
	}
	return hex.EncodeToString(digest.Sum(nil)), true
}

func mirrorFile(ctx context.Context, directory string, file patchserver.ContentFile) error {
	target := filepath.Join(directory, "patch_files", file.Name)
	if digest, ok := fileMD5(target, file.Info.Size); ok && digest == file.Info.MD5 {
		return nil
	}
	partial := target + ".partial"
	var last error
	for attempt := range 6 {
		if attempt > 0 {
			select {
			case <-time.After(time.Duration(attempt) * 2 * time.Second):
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		if last = resumeDownload(ctx, file, partial); last != nil {
			continue
		}
		if digest, ok := fileMD5(partial, file.Info.Size); ok && digest == file.Info.MD5 {
			return os.Rename(partial, target)
		}
		os.Remove(partial)
		last = fmt.Errorf("%s does not match its checksum", file.Name)
	}
	return fmt.Errorf("download %s: %w", file.Name, last)
}

func resumeDownload(ctx context.Context, file patchserver.ContentFile, partial string) error {
	var offset int64
	if info, err := os.Stat(partial); err == nil {
		offset = info.Size()
	}
	if offset > file.Info.Size {
		os.Remove(partial)
		offset = 0
	}
	if offset == file.Info.Size {
		return nil
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, file.Origin+"/patch_files/"+file.Name, nil)
	if err != nil {
		return err
	}
	if offset > 0 {
		request.Header.Set("Range", fmt.Sprintf("bytes=%d-", offset))
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	flags := os.O_WRONLY | os.O_CREATE | os.O_APPEND
	switch {
	case offset > 0 && response.StatusCode == http.StatusPartialContent:
	case response.StatusCode == http.StatusOK:
		flags = os.O_WRONLY | os.O_CREATE | os.O_TRUNC
		offset = 0
	default:
		return fmt.Errorf("HTTP %d", response.StatusCode)
	}
	output, err := os.OpenFile(partial, flags, 0644)
	if err != nil {
		return err
	}
	_, err = io.Copy(output, io.LimitReader(response.Body, file.Info.Size-offset))
	if closeErr := output.Close(); err == nil {
		err = closeErr
	}
	return err
}
