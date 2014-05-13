package config

import (
	"encoding/json"
	"log"
	"reflect"
)

type userAppConfig struct {
	source                          string
	httpAppConfig                   HttpAppConfig
	commonAppConfig                 CommonAppConfig
	storageAppConfig                StorageAppConfig
	imageMagickRenderAgentAppConfig ImageMagickRenderAgentAppConfig
	documentRenderAgentAppConfig    DocumentRenderAgentAppConfig
	assetApiAppConfig               AssetApiAppConfig
	simpleApiAppConfig              SimpleApiAppConfig
	uploaderAppConfig               UploaderAppConfig
	downloaderAppConfig             DownloaderAppConfig
}

type userCommonAppConfig struct {
	placeholderBasePath   string
	placeholderGroups     map[string][]string
	nodeId                string
	localAssetStoragePath string
}

type userHttpAppConfig struct {
	listen string
}

type userStorageAppConfig struct {
	engine            string
	cassandraNodes    []string
	cassandraKeyspace string
}

type userImageMagickRenderAgentAppConfig struct {
	enabled            bool
	count              int
	supportedFileTypes map[string]int64
}

type userDocumentRenderAgentAppConfig struct {
	enabled  bool
	count    int
	basePath string
}

type userSimpleApiAppConfig struct {
	enabled     bool
	edgeBaseUrl string
}

type userAssetApiAppConfig struct {
	enabled bool
}

type userUploaderAppConfig struct {
	engine    string
	s3Key     string
	s3Secret  string
	s3Host    string
	s3Buckets []string
}

type userDownloaderAppConfig struct {
	basePath    string
	tramEnabled bool
	tramHosts   []string
}

func NewUserAppConfig(content []byte) (AppConfig, error) {

	var f interface{}
	err := json.Unmarshal(content, &f)
	if err != nil {
		return nil, err
	}

	m := f.(map[string]interface{})
	appConfig := new(userAppConfig)
	appConfig.source = string(content)

	appConfig.commonAppConfig, err = newUserCommonAppConfig(m)
	if err != nil {
		return nil, err
	}

	appConfig.httpAppConfig, err = newUserHttpAppConfig(m)
	if err != nil {
		return nil, err
	}

	appConfig.storageAppConfig, err = newUserStorageAppConfig(m)
	if err != nil {
		return nil, err
	}

	appConfig.imageMagickRenderAgentAppConfig, err = newUserImageMagickRenderAgentAppConfig(m)
	if err != nil {
		return nil, err
	}

	appConfig.documentRenderAgentAppConfig, err = newUserDocumentRenderAgentAppConfig(m)
	if err != nil {
		return nil, err
	}

	appConfig.simpleApiAppConfig, err = newUserSimpleApiAppConfig(m)
	if err != nil {
		return nil, err
	}

	appConfig.assetApiAppConfig, err = newUserAssetApiAppConfig(m)
	if err != nil {
		return nil, err
	}

	appConfig.uploaderAppConfig, err = newUserUploaderAppConfig(m)
	if err != nil {
		return nil, err
	}

	appConfig.downloaderAppConfig, err = newUserDownloaderAppConfig(m)
	if err != nil {
		return nil, err
	}

	return appConfig, nil
}

func newUserHttpAppConfig(m map[string]interface{}) (HttpAppConfig, error) {
	data, err := parseConfigGroup("http", m)
	if err != nil {
		return nil, err
	}

	config := new(userHttpAppConfig)

	config.listen, err = parseString("http", "listen", data)
	if err != nil {
		return nil, err
	}

	return config, nil
}

func newUserCommonAppConfig(m map[string]interface{}) (CommonAppConfig, error) {
	data, err := parseConfigGroup("common", m)
	if err != nil {
		return nil, err
	}

	config := new(userCommonAppConfig)

	config.localAssetStoragePath, err = parseString("common", "localAssetStoragePath", data)
	if err != nil {
		return nil, err
	}

	config.placeholderBasePath, err = parseString("common", "placeholderBasePath", data)
	if err != nil {
		return nil, err
	}

	config.nodeId, err = parseString("common", "nodeId", data)
	if err != nil {
		return nil, err
	}

	placeholderGroupsData, hasKey := data["placeholderGroups"]
	if !hasKey {
		return nil, appConfigError{"Invalid common config: placeholderGroups attribute missing"}
	}
	placeholderGroups, ok := placeholderGroupsData.(map[string]interface{})
	if !ok {
		log.Println(reflect.TypeOf(placeholderGroups).Kind())
		return nil, appConfigError{"Invalid common config: placeholderGroups attribute not a map of strings to string arrays"}
	}
	config.placeholderGroups = make(map[string][]string)
	for label, groupValuesData := range placeholderGroups {
		groupValues, err := getStringArray(groupValuesData)
		if err == nil {
			config.placeholderGroups[label] = groupValues
		} else {
			config.placeholderGroups[label] = make([]string, 0, 0)
		}
	}

	return config, nil
}

func newUserStorageAppConfig(m map[string]interface{}) (StorageAppConfig, error) {
	data, err := parseConfigGroup("storage", m)
	if err != nil {
		return nil, err
	}

	config := new(userStorageAppConfig)

	config.engine, err = parseString("storage", "engine", data)
	if err != nil {
		return nil, err
	}

	if config.engine == "cassandra" {
		config.cassandraKeyspace, err = parseString("storage", "cassandraKeyspace", data)
		if err != nil {
			return nil, err
		}

		config.cassandraNodes, err = parseStringArray("storage", "cassandraNodes", data)
		if err != nil {
			return nil, err
		}
	}

	return config, nil
}

func newUserImageMagickRenderAgentAppConfig(m map[string]interface{}) (ImageMagickRenderAgentAppConfig, error) {
	data, err := parseConfigGroup("imageMagickRenderAgent", m)
	if err != nil {
		return nil, err
	}
	config := new(userImageMagickRenderAgentAppConfig)

	config.enabled, err = parseBool("imageMagickRenderAgent", "enabled", data)
	if err != nil {
		return nil, err
	}
	config.count, err = parseInt("imageMagickRenderAgent", "count", data)
	if err != nil {
		return nil, err
	}

	supportedFileTypesValue, hasKey := data["supportedFileTypes"]
	if !hasKey {
		return nil, appConfigError{"Invalid imageMagickRenderAgent config: supportedFileTypes attribute missing"}
	}
	supportedFileTypes, ok := supportedFileTypesValue.(map[string]interface{})
	if !ok {
		log.Println(reflect.TypeOf(supportedFileTypesValue).Kind())
		return nil, appConfigError{"Invalid imageMagickRenderAgent config: supportedFileTypes attribute not a map of strings to ints"}
	}
	config.supportedFileTypes = make(map[string]int64)
	for fileType, fileSize := range supportedFileTypes {
		val, err := getFloat(fileSize)
		if err != nil {
			log.Println(err.Error())
		} else {
			config.supportedFileTypes[fileType] = int64(val)
		}
	}

	return config, nil
}

func newUserDocumentRenderAgentAppConfig(m map[string]interface{}) (DocumentRenderAgentAppConfig, error) {
	data, err := parseConfigGroup("documentRenderAgent", m)
	if err != nil {
		return nil, err
	}

	config := new(userDocumentRenderAgentAppConfig)

	config.enabled, err = parseBool("documentRenderAgent", "enabled", data)
	if err != nil {
		return nil, err
	}

	config.basePath, err = parseString("documentRenderAgent", "basePath", data)
	if err != nil {
		return nil, err
	}

	config.count, err = parseInt("documentRenderAgent", "count", data)
	if err != nil {
		return nil, err
	}

	return config, nil
}

func newUserSimpleApiAppConfig(m map[string]interface{}) (SimpleApiAppConfig, error) {
	data, err := parseConfigGroup("simpleApi", m)
	if err != nil {
		return nil, err
	}

	config := new(userSimpleApiAppConfig)

	config.enabled, err = parseBool("simpleApi", "enabled", data)
	if err != nil {
		return nil, err
	}

	config.edgeBaseUrl, err = parseString("simpleApi", "edgeBaseUrl", data)
	if err != nil {
		return nil, err
	}

	return config, nil
}

func newUserAssetApiAppConfig(m map[string]interface{}) (AssetApiAppConfig, error) {
	data, err := parseConfigGroup("assetApi", m)
	if err != nil {
		return nil, err
	}

	config := new(userAssetApiAppConfig)

	config.enabled, err = parseBool("assetApi", "enabled", data)
	if err != nil {
		return nil, err
	}

	return config, nil
}

func newUserUploaderAppConfig(m map[string]interface{}) (UploaderAppConfig, error) {
	data, err := parseConfigGroup("uploader", m)
	if err != nil {
		return nil, err
	}

	config := new(userUploaderAppConfig)

	config.engine, err = parseString("uploader", "engine", data)
	if err != nil {
		return nil, err
	}

	if config.engine == "s3" {
		config.s3Key, err = parseString("uploader", "s3Key", data)
		if err != nil {
			return nil, err
		}
		config.s3Secret, err = parseString("uploader", "s3Secret", data)
		if err != nil {
			return nil, err
		}
		config.s3Host, err = parseString("uploader", "s3Host", data)
		if err != nil {
			return nil, err
		}
		config.s3Buckets, err = parseStringArray("uploader", "s3Buckets", data)
		if err != nil {
			return nil, err
		}
	}

	return config, nil
}

func newUserDownloaderAppConfig(m map[string]interface{}) (DownloaderAppConfig, error) {
	data, err := parseConfigGroup("downloader", m)
	if err != nil {
		return nil, err
	}

	config := new(userDownloaderAppConfig)

	config.basePath, err = parseString("downloader", "basePath", data)
	if err != nil {
		return nil, err
	}

	config.tramEnabled, err = parseBool("downloader", "tramEnabled", data)
	if err != nil {
		return nil, err
	}
	if config.tramEnabled {
		config.tramHosts, err = parseStringArray("downloader", "tramHosts", data)
		if err != nil {
			return nil, err
		}
	}

	return config, nil
}

func (c *userAppConfig) Source() string {
	return c.source
}

func (c *userAppConfig) Common() CommonAppConfig {
	return c.commonAppConfig
}

func (c *userAppConfig) Http() HttpAppConfig {
	return c.httpAppConfig
}

func (c *userAppConfig) Storage() StorageAppConfig {
	return c.storageAppConfig
}

func (c *userAppConfig) ImageMagickRenderAgent() ImageMagickRenderAgentAppConfig {
	return c.imageMagickRenderAgentAppConfig
}

func (c *userAppConfig) DocumentRenderAgent() DocumentRenderAgentAppConfig {
	return c.documentRenderAgentAppConfig
}

func (c *userAppConfig) SimpleApi() SimpleApiAppConfig {
	return c.simpleApiAppConfig
}

func (c *userAppConfig) AssetApi() AssetApiAppConfig {
	return c.assetApiAppConfig
}

func (c *userAppConfig) Uploader() UploaderAppConfig {
	return c.uploaderAppConfig
}

func (c *userAppConfig) Downloader() DownloaderAppConfig {
	return c.downloaderAppConfig
}

func (c *userHttpAppConfig) Listen() string {
	return c.listen
}

func (c *userStorageAppConfig) Engine() string {
	return c.engine
}

func (c *userStorageAppConfig) CassandraNodes() ([]string, error) {
	if c.engine == "cassandra" {
		return c.cassandraNodes, nil
	}
	return nil, appConfigError{"Cassandra storage engine is not enabled."}
}

func (c *userStorageAppConfig) CassandraKeyspace() (string, error) {
	if c.engine == "cassandra" {
		return c.cassandraKeyspace, nil
	}
	return "", appConfigError{"Cassandra storage engine is not enabled."}
}

func (c *userImageMagickRenderAgentAppConfig) Enabled() bool {
	return c.enabled
}

func (c *userImageMagickRenderAgentAppConfig) Count() int {
	return c.count
}

func (c *userImageMagickRenderAgentAppConfig) SupportedFileTypes() map[string]int64 {
	return c.supportedFileTypes
}

func (c *userDocumentRenderAgentAppConfig) Enabled() bool {
	return c.enabled
}

func (c *userDocumentRenderAgentAppConfig) Count() int {
	return c.count
}

func (c *userDocumentRenderAgentAppConfig) BasePath() string {
	return c.basePath
}

func (c *userSimpleApiAppConfig) Enabled() bool {
	return c.enabled
}

func (c *userSimpleApiAppConfig) EdgeBaseUrl() string {
	return c.edgeBaseUrl
}

func (c *userAssetApiAppConfig) Enabled() bool {
	return c.enabled
}

func (c *userUploaderAppConfig) Engine() string {
	return c.engine
}

func (c *userUploaderAppConfig) S3Key() (string, error) {
	if c.engine == "s3" {
		return c.s3Key, nil
	}
	return "", appConfigError{"S3 uploader engine is not enabled."}
}

func (c *userUploaderAppConfig) S3Secret() (string, error) {
	if c.engine == "s3" {
		return c.s3Secret, nil
	}
	return "", appConfigError{"S3 uploader engine is not enabled."}
}

func (c *userUploaderAppConfig) S3Host() (string, error) {
	if c.engine == "s3" {
		return c.s3Host, nil
	}
	return "", appConfigError{"S3 uploader engine is not enabled."}
}

func (c *userUploaderAppConfig) S3Buckets() ([]string, error) {
	if c.engine == "s3" {
		return c.s3Buckets, nil
	}
	return nil, appConfigError{"S3 uploader engine is not enabled."}
}

func (c *userDownloaderAppConfig) BasePath() string {
	return c.basePath
}

func (c *userDownloaderAppConfig) TramEnabled() bool {
	return c.tramEnabled
}
func (c *userDownloaderAppConfig) TramHosts() ([]string, error) {
	if c.TramEnabled() {
		return c.tramHosts, nil
	}
	return nil, appConfigError{"Tram support is not enabled."}
}

func (c *userCommonAppConfig) NodeId() string {
	return c.nodeId
}

func (c *userCommonAppConfig) LocalAssetStoragePath() string {
	return c.localAssetStoragePath
}

func (c *userCommonAppConfig) PlaceholderBasePath() string {
	return c.placeholderBasePath
}

func (c *userCommonAppConfig) PlaceholderGroups() map[string][]string {
	return c.placeholderGroups
}
