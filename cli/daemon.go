package cli

import (
	"fmt"
	"github.com/ngerakines/preview/app"
	"github.com/ngerakines/preview/config"
	"log"
	"os"
	"os/signal"
)

type DaemonCommand struct {
	config string
}

func NewDaemonCommand(arguments map[string]interface{}) PreviewCliCommand {
	command := new(DaemonCommand)
	command.config = getConfigString(arguments, "--config")
	return command
}

func (command *DaemonCommand) String() string {
	return fmt.Sprintf("DaemonCommand<config=%s>", command.config)
}

func (command *DaemonCommand) Execute() {
	appConfig, err := config.LoadAppConfig(command.config)
	if err != nil {
		log.Fatal(err.Error())
		return
	}
	previewApp, err := app.NewApp(appConfig)
	if err != nil {
		log.Fatal(err.Error())
		return
	}

	k := make(chan os.Signal, 1)
	signal.Notify(k, os.Interrupt, os.Kill)
	go func() {
		<-k
		previewApp.Stop()
	}()

	previewApp.Start()
}
