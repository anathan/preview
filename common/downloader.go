package common

import (
	"code.google.com/p/go-uuid/uuid"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// Downloader structures retreive remote files and make them available locally.
type Downloader interface {
	// Download attempts to retreive a file with a given url and store it to a temporary file that is managed by a TemporaryFileManager.
	Download(url string) (TemporaryFile, error)
}

type defaultDownloader struct {
	basePath         string
	localStoragePath string
	tfm              TemporaryFileManager
}

// NewDownloader creates, configures and returns a new defaultDownloader.
func NewDownloader(basePath, localStoragePath string, tfm TemporaryFileManager) Downloader {
	downloader := new(defaultDownloader)
	downloader.basePath = basePath
	downloader.localStoragePath = localStoragePath
	downloader.tfm = tfm
	return downloader
}

// Download attempts to retreive a file with a given url and store it to a temporary file that is managed by a TemporaryFileManager.
func (downloader *defaultDownloader) Download(url string) (TemporaryFile, error) {
	log.Println("Attempting to download", url)
	if strings.HasPrefix(url, "file://") {
		return downloader.handleFile(url)
	}
	if strings.HasPrefix(url, "local://") {
		return downloader.handleLocal(url)
	}
	if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
		return downloader.handleHttp(url)
	}
	return nil, ErrorNotImplemented
}

func (downloader *defaultDownloader) handleLocal(url string) (TemporaryFile, error) {
	log.Println("Attempting to download file", url[8:])
	path := filepath.Join(downloader.localStoragePath, url[8:])

	newPath := filepath.Join(downloader.basePath, uuid.New())

	newPathDir := filepath.Dir(newPath)
	err := os.MkdirAll(newPathDir, 0777)
	if err != nil {
		log.Println("error copying file:", err.Error())
		return nil, err
	}

	err = copyFile(path, newPath)
	if err != nil {
		log.Println("error copying file:", err.Error())
		return nil, err
	}
	log.Println("File", path, "copied to", newPath)

	return downloader.tfm.Create(newPath), nil
}

func (downloader *defaultDownloader) handleFile(url string) (TemporaryFile, error) {
	log.Println("Attempting to download file", url[7:])
	log.Println("downloading file url", url)
	path := url[7:]
	log.Println("actual path", path)

	newPath := filepath.Join(downloader.basePath, uuid.New())

	newPathDir := filepath.Dir(newPath)
	err := os.MkdirAll(newPathDir, 0777)
	if err != nil {
		log.Println("error copying file:", err.Error())
		return nil, err
	}

	err = copyFile(path, newPath)
	if err != nil {
		log.Println("error copying file:", err.Error())
		return nil, err
	}
	log.Println("File", path, "copied to", newPath)

	return downloader.tfm.Create(newPath), nil
}

func (downloader *defaultDownloader) handleHttp(url string) (TemporaryFile, error) {
	newPath := filepath.Join(downloader.basePath, uuid.New())

	newPathDir := filepath.Dir(newPath)
	os.MkdirAll(newPathDir, 0777)

	out, err := os.Create(newPath)
	if err != nil {
		return nil, err
	}
	defer out.Close()

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	n, err := io.Copy(out, resp.Body)
	if err != nil {
		return nil, err
	}
	log.Println("Downloaded", n, "bytes to file", newPath)

	return downloader.tfm.Create(newPath), nil
}

// CopyFile copies a file from src to dst. If src and dst files exist, and are
// the same, then return success. Otherise, attempt to create a hard link
// between the two files. If that fail, copy the file contents from src to dst.
func copyFile(src, dst string) (err error) {
	sfi, err := os.Stat(src)
	if err != nil {
		return
	}
	if !sfi.Mode().IsRegular() {
		// cannot copy non-regular files (e.g., directories,
		// symlinks, devices, etc.)
		return fmt.Errorf("CopyFile: non-regular source file %s (%q)", sfi.Name(), sfi.Mode().String())
	}
	dfi, err := os.Stat(dst)
	if err != nil {
		if !os.IsNotExist(err) {
			return
		}
	} else {
		if !(dfi.Mode().IsRegular()) {
			return fmt.Errorf("CopyFile: non-regular destination file %s (%q)", dfi.Name(), dfi.Mode().String())
		}
		if os.SameFile(sfi, dfi) {
			return
		}
	}
	if err = os.Link(src, dst); err == nil {
		return
	}
	err = copyFileContents(src, dst)
	return
}

// copyFileContents copies the contents of the file named src to the file named
// by dst. The file will be created if it does not already exist. If the
// destination file exists, all it's contents will be replaced by the contents
// of the source file.
func copyFileContents(src, dst string) (err error) {
	in, err := os.Open(src)
	if err != nil {
		return
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return
	}
	defer func() {
		cerr := out.Close()
		if err == nil {
			err = cerr
		}
	}()
	if _, err = io.Copy(out, in); err != nil {
		return
	}
	err = out.Sync()
	return
}
