package config

import (
	"github.com/ngerakines/preview/util"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
)

type appConfigError struct {
	message string
}

// AppConfig represents the configuration collections for the preview application.
type AppConfig interface {
	// Common returns common configuration.
	Common() CommonAppConfig
	// Http returns HTTP configuration.
	Http() HttpAppConfig
	// Storage returns storage configuration.
	Storage() StorageAppConfig
	// ImageMagickRenderer returns ImageMagick render agent configuration.
	ImageMagickRenderAgent() ImageMagickRenderAgentAppConfig
	// DocumentRenderAgent returns Document render agent configuration.
	DocumentRenderAgent() DocumentRenderAgentAppConfig
	// SimpleApi returns SimpleBlueprint configuration.
	SimpleApi() SimpleApiAppConfig
	AssetApi() AssetApiAppConfig
	Uploader() UploaderAppConfig
	Downloader() DownloaderAppConfig
	Source() string
}

type CommonAppConfig interface {
	PlaceholderBasePath() string
	PlaceholderGroups() map[string][]string
	LocalAssetStoragePath() string
	NodeId() string
}

type HttpAppConfig interface {
	Listen() string
}

type StorageAppConfig interface {
	Engine() string
	CassandraNodes() ([]string, error)
	CassandraKeyspace() (string, error)
}

type ImageMagickRenderAgentAppConfig interface {
	Enabled() bool
	Count() int
	SupportedFileTypes() map[string]int64
}

type DocumentRenderAgentAppConfig interface {
	Enabled() bool
	Count() int
	BasePath() string
}

type SimpleApiAppConfig interface {
	Enabled() bool
	EdgeBaseUrl() string
}

type AssetApiAppConfig interface {
	Enabled() bool
}

type UploaderAppConfig interface {
	Engine() string
	S3Key() (string, error)
	S3Secret() (string, error)
	S3Host() (string, error)
	S3Buckets() ([]string, error)
}

type DownloaderAppConfig interface {
	BasePath() string
}

func LoadAppConfig(givenPath string) (AppConfig, error) {
	configPath := determineConfigPath(givenPath)
	if configPath == "" {
		return NewDefaultAppConfig()
	}
	content, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, err
	}
	return NewUserAppConfig(content)
}

func (err appConfigError) Error() string {
	return err.message
}

func determineConfigPath(givenPath string) string {
	paths := []string{
		givenPath,
		filepath.Join(util.Cwd(), "preview.config"),
		filepath.Join(userHomeDir(), ".preview.config"),
		"/etc/preview.config",
	}
	for _, path := range paths {
		if util.CanLoadFile(path) {
			return path
		}
	}
	return ""
}

func userHomeDir() string {
	if runtime.GOOS == "windows" {
		home := filepath.Join(os.Getenv("HOMEDRIVE"), os.Getenv("HOMEPATH"))
		if home == "" {
			home = os.Getenv("USERPROFILE")
		}
		return home
	}
	return os.Getenv("HOME")
}
