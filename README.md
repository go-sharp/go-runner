# go-runner

go-runner watches for changes in \*.go, go.mod and go.sum files and recompiles/restarts the go program in the current directory.

go-runner requires at least go1.11 and a runnable main function in the working directory. If you use modules, dependencies will be downloaded automatically, otherwise you need to download them manually.

**Caveats**: A binary that creates own child processes, will not work well with go-runner. This is because at the moment go-runner can only kill the main process.

```bash
Usage for go-runner | Version 0.1.9:

go-runner [Options] [-- arguments (ex. -c config.json --http=:8080 1234)]

Options:
  -e, --entry string           The directory with the main.go file (default "./")
  -x, --exclude-dirs strings   Don't listen to changes in these directories
  -h, --help                   Show help
  -s, --skip-tests             Don't run any tests
  -r, --test-non-recursive     Don't run tests recursively
  -t, --tests strings          Test directories in which the go test command will be executed (default [./])
  -w, --watch-dirs strings     Directories to listen recursively for file changes (*.go, go.mod, go.sum) (default [./])
```
