# go-runner

go-runner watches for changes in \*.go, go.mod and go.sum files and recompiles/restarts the go program in the current directory.

go-runner requires at least go1.11 and a runnable main function in the working directory. If you use modules, dependencies will be downloaded automatically, otherwise you need to download them manually.

**Caveats**: A binary that creates own child processes, will not work well with go-runner. This is because at the moment go-runner can only kill the main process. Furthermore if delve is used, the main program starts after a debugger client connects to the delve server.

```bash
Usage for go-runner | Version 0.2.0:

go-runner [Options] [-- arguments (ex. -c config.json --http=:8080 1234)]

Options:
  -a, --address ip             Listen address for delve (default 0.0.0.0)
  -v, --api-version int        API version to use for delve server (default 2)
  -e, --entry string           The directory with the main.go file (default "./")
  -x, --exclude-dirs strings   Don't listen to changes in these directories
  -h, --help                   Show help
  -p, --port uint16            Listen port for delve (default 2345)
  -s, --skip-tests             Don't run any tests
  -r, --test-non-recursive     Don't run tests recursively
  -t, --tests strings          Test directories in which the go test command will be executed (default [./])
  -d, --use-dlv                Use delve to run the program
  -w, --watch-dirs strings     Directories to listen recursively for file changes (*.go, go.mod, go.sum) (default [./])

```
