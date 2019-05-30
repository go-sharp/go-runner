package main

import (
	"fmt"
	"os"
	"os/signal"

	"github.com/go-sharp/go-runner/log"
	"github.com/go-sharp/go-runner/runner"
	flag "github.com/spf13/pflag"
)

const Version = "0.1.3"

func main() {
	cd := flag.StringP("entry", "e", "./", "The directory with the main.go file (default: current working directory)")
	testdirs := flag.StringSliceP("tests", "t", []string{"./"}, "Test directories in which the go test command will be executed (default: current working directory)")
	skipTests := flag.BoolP("skip-tests", "s", false, "Don't run any tests (default: false)")
	recursiveTests := flag.BoolP("test-non-recursive", "r", false, "Don't run tests recursively (default: false)")
	watchDirs := flag.StringSliceP("watch-dirs", "w", []string{"./"}, "Directories to listen recursively for file changes (*.go, go.mod, go.sum) (default: entry directory)")
	excludeDirs := flag.StringSliceP("exclude-dirs", "x", []string{}, "Don't listen to changes in these directories (default: none)")
	cmdArgs := flag.StringSliceP("args", "a", []string{}, "Arguments to pass to the program (default: none)")
	help := flag.BoolP("help", "h", false, "Show help")

	flag.Parse()

	if *help {
		fmt.Printf("Usage for %v | Version %v:\n\n", os.Args[0], Version)
		fmt.Printf("%v [Options] arguments...\n\nOptions:\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(0)
	}

	r := runner.NewRunner(runner.WorkingDirectory(*cd),
		runner.TestWorkingDirectories(*testdirs...),
		runner.RunTests(!*skipTests),
		runner.RecursiveTests(!*recursiveTests),
		runner.WatchDirs(*watchDirs...),
		runner.ExcludeDirs(*excludeDirs...),
		runner.CommandArgs(*cmdArgs...))

	if err := r.Watch(); err != nil {
		log.Errorln(err)
		os.Exit(1)
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c
	log.Infoln("Shutting down go-runner...")
	r.Stop()
}
