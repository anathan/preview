package app

import (
	"github.com/bmizerany/pat"
	"github.com/codegangsta/negroni"
	"github.com/etix/stoppableListener"
	"github.com/ngerakines/preview/api"
	"github.com/ngerakines/preview/common"
	"github.com/ngerakines/preview/config"
	"github.com/ngerakines/preview/render"
	"github.com/rcrowley/go-metrics"
	"log"
	"net"
	"net/http"
	"os"
	"time"
)

type AppContext struct {
	registry                     metrics.Registry
	appConfig                    config.AppConfig
	agentManager                 *render.RenderAgentManager
	sourceAssetStorageManager    common.SourceAssetStorageManager
	generatedAssetStorageManager common.GeneratedAssetStorageManager
	templateManager              common.TemplateManager
	downloader                   common.Downloader
	uploader                     common.Uploader
	temporaryFileManager         common.TemporaryFileManager
	placeholderManager           common.PlaceholderManager
	signatureManager             api.SignatureManager
	simpleBlueprint              api.Blueprint
	assetBlueprint               api.Blueprint
	adminBlueprint               api.Blueprint
	staticBlueprint              api.Blueprint
	listener                     *stoppableListener.StoppableListener
	negroni                      *negroni.Negroni
	cassandraManager             *common.CassandraManager
}

func NewApp(appConfig config.AppConfig) (*AppContext, error) {
	log.Println("Creating application with config", appConfig)
	app := new(AppContext)
	app.registry = metrics.NewRegistry()

	metrics.RegisterRuntimeMemStats(app.registry)
	go metrics.CaptureRuntimeMemStats(app.registry, 60e9)

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
	httpListener, err := net.Listen("tcp", app.appConfig.Http().Listen())
	if err != nil {
		panic(err)
	}
	app.listener = stoppableListener.Handle(httpListener)

	http.Serve(app.listener, app.negroni)

	if app.listener.Stopped {
		var alive int

		/* Wait at most 5 seconds for the clients to disconnect */
		for i := 0; i < 5; i++ {
			/* Get the number of clients still connected */
			alive = app.listener.ConnCount.Get()
			if alive == 0 {
				break
			}
			log.Printf("%d client(s) still connected…\n", alive)
			time.Sleep(1 * time.Second)
		}

		alive = app.listener.ConnCount.Get()
		if alive > 0 {
			log.Fatalf("Server stopped after 5 seconds with %d client(s) still connected.", alive)
		} else {
			log.Println("Server stopped gracefully.")
			os.Exit(0)
		}
	} else if err != nil {
		log.Fatal(err)
	}
}

func (app *AppContext) initTrams() error {
	app.placeholderManager = common.NewPlaceholderManager(app.appConfig)
	app.temporaryFileManager = common.NewTemporaryFileManager()
	if app.appConfig.Downloader().TramEnabled() {
		tramHosts, err := app.appConfig.Downloader().TramHosts()
		if err != nil {
			panic(err)
		}
		app.downloader = common.NewDownloader(app.appConfig.Downloader().BasePath(), app.appConfig.Common().LocalAssetStoragePath(), app.temporaryFileManager, true, tramHosts, app.buildS3Client())
	} else {
		app.downloader = common.NewDownloader(app.appConfig.Downloader().BasePath(), app.appConfig.Common().LocalAssetStoragePath(), app.temporaryFileManager, false, []string{}, app.buildS3Client())
	}

	switch app.appConfig.Uploader().Engine() {
	case "s3":
		{
			s3Client := app.buildS3Client()
			buckets, err := app.appConfig.Uploader().S3Buckets()
			if err != nil {
				panic(err)
			}
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
	app.agentManager = render.NewRenderAgentManager(app.registry, app.sourceAssetStorageManager, app.generatedAssetStorageManager, app.templateManager, app.temporaryFileManager, app.uploader)
	app.agentManager.SetRenderAgentInfo(common.RenderAgentImageMagick, app.appConfig.ImageMagickRenderAgent().Enabled(), app.appConfig.ImageMagickRenderAgent().Count())
	app.agentManager.SetRenderAgentInfo(common.RenderAgentDocument, app.appConfig.DocumentRenderAgent().Enabled(), app.appConfig.DocumentRenderAgent().Count())
	if app.appConfig.ImageMagickRenderAgent().Enabled() {
		for i := 0; i < app.appConfig.ImageMagickRenderAgent().Count(); i++ {
			app.agentManager.AddImageMagickRenderAgent(app.downloader, app.uploader, 5)
		}
	}
	if app.appConfig.DocumentRenderAgent().Enabled() {
		for i := 0; i < app.appConfig.DocumentRenderAgent().Count(); i++ {
			app.agentManager.AddDocumentRenderAgent(app.downloader, app.uploader, app.appConfig.DocumentRenderAgent().BasePath(), 5)
		}
	}
	return nil
}

func (app *AppContext) initApis() error {
	// NKG: This is where different APIs are configured and enabled.

	allSupportedFileTypes := make(map[string]int64)
	for fileType, maxFileSize := range app.appConfig.ImageMagickRenderAgent().SupportedFileTypes() {
		allSupportedFileTypes[fileType] = maxFileSize
	}

	app.signatureManager = api.NewSignatureManager()

	var err error

	p := pat.New()

	if app.appConfig.SimpleApi().Enabled() {
		app.simpleBlueprint, err = api.NewSimpleBlueprint(app.registry, app.appConfig.SimpleApi().BaseUrl(), app.appConfig.SimpleApi().EdgeBaseUrl(), app.agentManager, app.sourceAssetStorageManager, app.generatedAssetStorageManager, app.templateManager, app.placeholderManager, app.signatureManager, allSupportedFileTypes)
		if err != nil {
			return err
		}
		app.simpleBlueprint.AddRoutes(p)
	}

	app.assetBlueprint = api.NewAssetBlueprint(app.registry, app.appConfig.Common().LocalAssetStoragePath(), app.sourceAssetStorageManager, app.generatedAssetStorageManager, app.templateManager, app.placeholderManager, app.buildS3Client(), app.signatureManager)
	app.assetBlueprint.AddRoutes(p)

	app.adminBlueprint = api.NewAdminBlueprint(app.registry, app.appConfig, app.placeholderManager, app.temporaryFileManager, app.agentManager)
	app.adminBlueprint.AddRoutes(p)

	app.staticBlueprint = api.NewStaticBlueprint(app.placeholderManager)
	app.staticBlueprint.AddRoutes(p)

	app.negroni = negroni.Classic()
	app.negroni.UseHandler(p)

	return nil
}

func (app *AppContext) Stop() {
	app.agentManager.Stop()
	if app.cassandraManager != nil {
		app.cassandraManager.Stop()
	}
	app.listener.Stop <- true
}

func (app *AppContext) buildS3Client() common.S3Client {
	if app.appConfig.Uploader().Engine() != "s3" {
		return nil
	}
	awsKey, err := app.appConfig.Uploader().S3Key()
	if err != nil {
		panic(err)
	}
	awsSecret, err := app.appConfig.Uploader().S3Secret()
	if err != nil {
		panic(err)
	}
	awsHost, err := app.appConfig.Uploader().S3Host()
	if err != nil {
		panic(err)
	}
	log.Println("Creating s3 client with host", awsHost, "key", awsKey, "and secret", awsSecret)
	return common.NewAmazonS3Client(common.NewBasicS3Config(awsKey, awsSecret, awsHost))
}
