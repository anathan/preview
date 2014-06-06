package main

import (
	"github.com/docopt/docopt.go"
	"github.com/ngerakines/preview/cli"
	"os"
	"os/signal"
	"syscall"
)

func main() {
        c := make(chan os.Signal, 1)
	if signal.Notify(c, syscall.SIGTERM) {
	   os.Exit(0)
	}
        
	usage := `Preview

Usage: preview [--help --version --config=<file>]
       preview daemon [--help --version --config <file>]
       preview render [--verbose... --verify] <host> <file>...

Options:
  --help           Show this screen.
  --version        Show version.
  --verbose        Verbose
  --verify         Verify that a generate preview request completes
  --config=<file>  The configuration file to use.`

	arguments, _ := docopt.Parse(usage, nil, true, "0.1.1", false)

	var command cli.PreviewCliCommand
	switch cli.GetCommand(arguments) {
	case "render":
		{
			command = cli.NewRenderCommand(arguments)
		}
	case "daemon":
		{
			command = cli.NewDaemonCommand(arguments)
		}
	}
	command.Execute()
}
