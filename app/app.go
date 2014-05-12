package app

import (
	"github.com/codegangsta/martini"
	"github.com/ngerakines/preview/api"
	"github.com/ngerakines/preview/common"
	"github.com/ngerakines/preview/config"
	"github.com/ngerakines/preview/render"
	"log"
	"net/http"
)

type AppContext struct {
	appConfig                    config.AppConfig
	agentManager                 *render.RenderAgentManager
	sourceAssetStorageManager    common.SourceAssetStorageManager
	generatedAssetStorageManager common.GeneratedAssetStorageManager
	templateManager              common.TemplateManager
	downloader                   common.Downloader
	uploader                     common.Uploader
	temporaryFileManager         common.TemporaryFileManager
	placeholderManager           common.PlaceholderManager
	simpleBlueprint              api.Blueprint
	assetBlueprint               api.Blueprint
	staticBlueprint              api.Blueprint
	adminBlueprint               api.Blueprint
	martiniClassic               *martini.ClassicMartini
	cassandraManager             *common.CassandraManager
}

func NewApp(appConfig config.AppConfig) (*AppContext, error) {
	log.Println("Creating application with config", appConfig)
	app := new(AppContext)
	app.appConfig = appConfig

	err := app.initTrams()
	if err != nil {
		return nil, err
	}
	err = app.initStorage()
	if err != nil {
		return nil, err
	}
	err = app.initRenderers()
	if err != nil {
		return nil, err
	}
	err = app.initApis()
	if err != nil {
		return nil, err
	}
	return app, nil
}

func (app *AppContext) Start() {
	log.Fatal(http.ListenAndServe(app.appConfig.Http().Listen(), app.martiniClassic))
}

func (app *AppContext) initTrams() error {
	app.placeholderManager = common.NewPlaceholderManager(app.appConfig)
	app.temporaryFileManager = common.NewTemporaryFileManager()
	app.downloader = common.NewDownloader(app.appConfig.Downloader().BasePath(), app.appConfig.Common().LocalAssetStoragePath(), app.temporaryFileManager)

	switch app.appConfig.Uploader().Engine() {
	case "s3":
		{
			buckets, err := app.appConfig.Uploader().S3Buckets()
			if err != nil {
				return err
			}
			awsKey, err := app.appConfig.Uploader().S3Key()
			if err != nil {
				return err
			}
			awsSecret, err := app.appConfig.Uploader().S3Secret()
			if err != nil {
				return err
			}
			awsHost, err := app.appConfig.Uploader().S3Host()
			if err != nil {
				return err
			}
			log.Println("Creating s3 client with host", awsHost, "key", awsKey, "and secret", awsSecret)
			s3Client := common.NewAmazonS3Client(common.NewBasicS3Config(awsKey, awsSecret, awsHost))
			app.uploader = common.NewUploader(buckets, s3Client)
		}
	case "local":
		{
			app.uploader = common.NewLocalUploader(app.appConfig.Common().LocalAssetStoragePath())
		}
	}

	return nil
}

func (app *AppContext) initStorage() error {
	// NKG: This is where local (in-memory) or cassandra backed storage is
	// configured and the SourceAssetStorageManager,
	// GeneratedAssetStorageManager and TemplateManager objects are created
	// and placed into the app context.

	app.templateManager = common.NewTemplateManager()

	switch app.appConfig.Storage().Engine() {
	case "memory":
		{
			app.sourceAssetStorageManager = common.NewSourceAssetStorageManager()
			app.generatedAssetStorageManager = common.NewGeneratedAssetStorageManager(app.templateManager)
			return nil
		}
	case "cassandra":
		{
			log.Println("Using cassandra!")
			cassandraNodes, err := app.appConfig.Storage().CassandraNodes()
			if err != nil {
				return err
			}
			keyspace, err := app.appConfig.Storage().CassandraKeyspace()
			if err != nil {
				return err
			}
			cm, err := common.NewCassandraManager(cassandraNodes, keyspace)
			if err != nil {
				return err
			}
			app.cassandraManager = cm
			app.sourceAssetStorageManager, err = common.NewCassandraSourceAssetStorageManager(cm, app.appConfig.Common().NodeId(), keyspace)
			if err != nil {
				return err
			}
			app.generatedAssetStorageManager, err = common.NewCassandraGeneratedAssetStorageManager(cm, app.templateManager, app.appConfig.Common().NodeId(), keyspace)
			if err != nil {
				return err
			}
			return nil
		}
	}
	return common.ErrorNotImplemented
}

func (app *AppContext) initRenderers() error {
	// NKG: This is where the RendererManager is constructed and renderers
	// are configured and enabled through it.
	app.agentManager = render.NewRenderAgentManager(app.sourceAssetStorageManager, app.generatedAssetStorageManager, app.templateManager, app.temporaryFileManager)
	if app.appConfig.ImageMagickRenderAgent().Enabled() {
		for i := 0; i < app.appConfig.ImageMagickRenderAgent().Count(); i++ {
			app.agentManager.AddImageMagickRenderAgent(app.downloader, app.uploader, 5)
		}
	}
	return nil
}

func (app *AppContext) initApis() error {
	// NKG: This is where different APIs are configured and enabled.

	app.martiniClassic = martini.Classic()

	allSupportedFileTypes := make(map[string]int64)
	for fileType, maxFileSize := range app.appConfig.ImageMagickRenderAgent().SupportedFileTypes() {
		allSupportedFileTypes[fileType] = maxFileSize
	}

	var err error

	if app.appConfig.SimpleApi().Enabled() {
		app.simpleBlueprint, err = api.NewSimpleBlueprint(app.appConfig, app.sourceAssetStorageManager, app.generatedAssetStorageManager, app.templateManager, app.placeholderManager, allSupportedFileTypes)
		if err != nil {
			return err
		}
		app.simpleBlueprint.ConfigureMartini(app.martiniClassic)
	}

	app.assetBlueprint = api.NewAssetBlueprint(app.appConfig.Common().LocalAssetStoragePath(), app.sourceAssetStorageManager, app.generatedAssetStorageManager, app.templateManager, app.placeholderManager)
	app.assetBlueprint.ConfigureMartini(app.martiniClassic)

	app.staticBlueprint = api.NewStaticBlueprint(app.placeholderManager)
	app.staticBlueprint.ConfigureMartini(app.martiniClassic)

	app.adminBlueprint = api.NewAdminBlueprint(app.appConfig, app.placeholderManager, app.temporaryFileManager)
	app.adminBlueprint.ConfigureMartini(app.martiniClassic)

	return nil
}

func (app *AppContext) Stop() {
	app.agentManager.Stop()
	app.cassandraManager.Stop()
}
