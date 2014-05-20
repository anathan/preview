package common

import (
	"log"
	"os"
	"sync"
	"time"
)

type TemporaryFile interface {
	Path() string
	Release()
}

type TemporaryFileManager interface {
	Notify(path string)
	Create(path string) TemporaryFile
	List() map[string]int
}

type defaultTemporaryFile struct {
	tfm  TemporaryFileManager
	path string
}

type defaultTemporaryFileManager struct {
	files map[string]int
	mu    sync.Mutex
}

func NewTemporaryFileManager() TemporaryFileManager {
	tfm := new(defaultTemporaryFileManager)
	tfm.files = make(map[string]int)
	return tfm
}

func (tf *defaultTemporaryFile) Path() string {
	return tf.path
}

func (tf *defaultTemporaryFile) Release() {
	go func() {
		time.Sleep(1 * time.Minute)
		tf.tfm.Notify(tf.path)
	}()
}

func (tfm *defaultTemporaryFileManager) Notify(path string) {
	tfm.mu.Lock()
	defer tfm.mu.Unlock()
	count, hasCount := tfm.files[path]
	if hasCount {
		count = count - 1
		if count > 0 {
			tfm.files[path] = count
			return
		}
		delete(tfm.files, path)
		err := os.Remove(path)
		if err != nil {
			log.Println(err)
		}
	}
}

func (tfm *defaultTemporaryFileManager) Create(path string) TemporaryFile {
	tfm.mu.Lock()
	defer tfm.mu.Unlock()
	count, hasCount := tfm.files[path]
	if hasCount {
		tfm.files[path] = count + 1
		return &defaultTemporaryFile{tfm, path}
	}
	tfm.files[path] = 1
	return &defaultTemporaryFile{tfm, path}
}

func (tfm *defaultTemporaryFileManager) List() map[string]int {
	tfm.mu.Lock()
	defer tfm.mu.Unlock()
	results := make(map[string]int)
	for path, count := range tfm.files {
		results[path] = count
	}
	return results
}
