package common

import (
	"fmt"
	"github.com/ngerakines/preview/config"
	"image/png"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type PlaceholderManager interface {
	Url(fileType, placeholderSize string) *Placeholder
	AllFileTypes() []string
}

type Placeholder struct {
	Url      string
	Path     string
	Height   int
	Width    int
	FileSize int64
}

type defaultPlaceholderManager struct {
	basePath     string
	groups       map[string]string
	placeholders map[string]placeholderSet
}

type placeholderSet map[string]*placeholderInfo

type placeholderInfo struct {
	size     string
	path     string
	fileSize int64
	height   int
	width    int
}

var (
	PlaceholderSizeJumbo  = "jumbo"
	PlaceholderSizeLarge  = "large"
	PlaceholderSizeMedium = "medium"
	PlaceholderSizeSmall  = "small"

	DefaultPlaceholderSizes = []string{PlaceholderSizeJumbo, PlaceholderSizeLarge, PlaceholderSizeMedium, PlaceholderSizeSmall}

	DefaultPlaceholderType = "unknown"
)

func NewPlaceholderManager(appConfig config.AppConfig) PlaceholderManager {
	pm := new(defaultPlaceholderManager)
	pm.basePath = appConfig.Common().PlaceholderBasePath()
	pm.groups = make(map[string]string)

	for group, fileTypes := range appConfig.Common().PlaceholderGroups() {
		for _, fileType := range fileTypes {
			pm.groups[fileType] = group
		}
	}

	pm.placeholders = make(map[string]placeholderSet)

	go pm.loadPlaceholders()

	return pm
}

func (pm *defaultPlaceholderManager) AllFileTypes() []string {
	results := make([]string, 0, 0)
	for fileType := range pm.placeholders {
		results = append(results, fileType)
	}
	return results
}

func (pm *defaultPlaceholderManager) Url(fileType, placeholderSize string) *Placeholder {

	fileTypePlaceholder, hasFileTypePlaceholder := pm.placeholders[fileType]
	if hasFileTypePlaceholder {
		fileTypeSize, hasfileTypeSize := fileTypePlaceholder[placeholderSize]
		if hasfileTypeSize {
			return &Placeholder{"/" + fileType + "/" + placeholderSize + ".png", fileTypeSize.path, fileTypeSize.height, fileTypeSize.width, fileTypeSize.fileSize}
		}
	}

	fileTypeGroup, hasFileTypeGroup := pm.groups[fileType]
	if hasFileTypeGroup {

		fileTypeGroupPlaceholder, hasFileTypeGroupPlaceholder := pm.placeholders[fileTypeGroup]
		if hasFileTypeGroupPlaceholder {
			fileTypeGroupSize, hasfileTypeGroupSize := fileTypeGroupPlaceholder[placeholderSize]
			if hasfileTypeGroupSize {
				return &Placeholder{"/" + fileTypeGroup + "/" + placeholderSize + ".png", fileTypeGroupSize.path, fileTypeGroupSize.height, fileTypeGroupSize.width, fileTypeGroupSize.fileSize}
			}
		}
	}

	unknownPlaceholder, hasUnknownPlaceholder := pm.placeholders[DefaultPlaceholderType]
	if hasUnknownPlaceholder {
		jumboUnknownPlaceholder, hasJumboUnknownPlaceholder := unknownPlaceholder[PlaceholderSizeJumbo]
		if hasJumboUnknownPlaceholder {
			return &Placeholder{"/" + DefaultPlaceholderType + "/" + PlaceholderSizeJumbo + ".png", jumboUnknownPlaceholder.path, jumboUnknownPlaceholder.height, jumboUnknownPlaceholder.width, jumboUnknownPlaceholder.fileSize}
		}
	}

	return &Placeholder{"/unknown/jumbo.png", "", 0, 0, 0}
}

func (pm *defaultPlaceholderManager) loadPlaceholders() {
	files, err := ioutil.ReadDir(pm.basePath)
	if err != nil {
		log.Println("Error reading files in placeholder base directory:", err)
		return
	}
	for _, file := range files {
		if file.IsDir() {
			subdirFiles, err := ioutil.ReadDir(filepath.Join(pm.basePath, file.Name()))
			if err == nil {
				for _, subdirFile := range subdirFiles {
					if !subdirFile.IsDir() {
						if strings.HasSuffix(subdirFile.Name(), ".png") {
							fullPath := filepath.Join(pm.basePath, file.Name(), subdirFile.Name())
							pm.loadPlaceholder(file.Name(), subdirFile.Name(), fullPath)
						}
					}
				}
			}
		}
	}
}

func (pm *defaultPlaceholderManager) loadPlaceholder(fileType, fileName, path string) {
	dotIndex := strings.Index(fileName, ".")
	placeholderSize := fileName[:dotIndex]

	pset, hasPset := pm.placeholders[fileType]
	if !hasPset {
		pset = make(placeholderSet)
	}

	height, width, err := pm.getBounds(path)
	if err != nil {
		return
	}
	fileSize, err := pm.getFileSize(path)
	if err != nil {
		return
	}

	pset[placeholderSize] = &placeholderInfo{placeholderSize, path, fileSize, height, width}

	pm.placeholders[fileType] = pset
}

func (pi *placeholderInfo) String() string {
	return fmt.Sprintf("placeholderInfo{size=%s fileSize=%d height=%d width=%d path=%s}", pi.size, pi.fileSize, pi.height, pi.width, pi.path)
}

func (pm *defaultPlaceholderManager) getBounds(path string) (int, int, error) {
	reader, err := os.Open(path)
	if err != nil {
		return 0, 0, err
	}
	defer reader.Close()
	image, err := png.Decode(reader)
	if err != nil {
		return 0, 0, err
	}
	bounds := image.Bounds()
	return bounds.Max.X, bounds.Max.Y, nil
}

func (pm *defaultPlaceholderManager) getFileSize(path string) (int64, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return fi.Size(), nil
}
