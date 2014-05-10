package main

import (
	"github.com/docopt/docopt-go"
	"github.com/ngerakines/preview/app"
	"github.com/ngerakines/preview/common"
	"github.com/ngerakines/preview/config"
	"log"
	_ "os"
	_ "os/signal"
)

func main() {
	usage := `Preview

Usage: preview [--help]
               [--version]
               [--config <file>]

Options:
  -h --help                 Show this screen.
  --version                 Show version.
  --dump-config             Output the default config as JSON.
  -c <FILE> --config <FILE> The configuration file to use. Unless a config
                            file is specified, the following paths will be
                            loaded:
                                ./preview.conf
                                ~/.preview.conf
                                /etc/preview.conf`

	arguments, _ := docopt.Parse(usage, nil, true, "v0.1.0", false)

	configPath := getConfig(arguments)
	appConfig, err := config.LoadAppConfig(configPath)
	if err != nil {
		log.Fatal(err.Error())
		return
	}
	previewApp, err := app.NewApp(appConfig)
	if err != nil {
		log.Fatal(err.Error())
		return
	}
	log.Println("Config", appConfig)

	/* c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)
	go func() {
		select {
		case <-c:
			{
				previewApp.Stop()
			}
		}
	}() */
	common.DumpErrors()
	previewApp.Start()
}

func getConfig(arguments map[string]interface{}) string {
	configPath, hasConfigPath := arguments["--config"]
	if hasConfigPath {
		value, ok := configPath.(string)
		if ok {
			return value
		}
	}
	return ""
}
