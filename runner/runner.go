package runner

import (
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/Masterminds/semver"
	"github.com/fsnotify/fsnotify"
	"github.com/go-sharp/go-runner/log"
)

const binName = "gr-tmp-bin"

var dlvRegex = regexp.MustCompile(`[Vv]ersion:\s+(.+)`)

// Option configures a runner instance.
type Option func(r *Runner)

// NewRunner creates a new runner instance with the given options.
func NewRunner(options ...Option) *Runner {
	goBin, err := exec.LookPath("go")
	if err != nil {
		panic("go compiler not found: " + err.Error())
	}

	runner := &Runner{rsCh: make(chan struct{}), stopped: make(chan struct{}), goBin: goBin}
	WorkingDirectory("./")(runner)
	TestWorkingDirectories("./")(runner)
	for _, fn := range options {
		fn(runner)
	}

	return runner
}

// CommandArgs configures the runner to pass the given arguments
// to the go program that is executed by the runner.
func CommandArgs(args ...string) Option {
	return Option(func(r *Runner) {
		r.cmdArgs = args
	})
}

// RunTests configures the runner to execute tests if true, otherwise
// tests will be skipped.
func RunTests(run bool) Option {
	return Option(func(r *Runner) {
		r.runTst = run
	})
}

// RecursiveTests configures the runner to call
//    go test ./...
// instead of
//    go test
func RecursiveTests(recursive bool) Option {
	return Option(func(r *Runner) {
		r.tstRecursive = recursive
	})
}

// WorkingDirectory configures the runner to use the given path as the working directory.
func WorkingDirectory(path string) Option {
	if path == "" {
		path = "./"
	}

	var err error
	if path, err = filepath.Abs(path); err != nil {
		// we shouldn't get here, but if so, then we bail out completely
		panic(err)
	}
	return Option(func(r *Runner) {
		r.pwd = path
	})
}

// TestWorkingDirectories configures the runner to use the given paths
// as the working directories for tests. The runner calls 'go test' for
// each path, this is useful if the runner is not configured to run tests
// recursively.
func TestWorkingDirectories(paths ...string) Option {
	return Option(func(r *Runner) {
		r.tstPwd = sanitizePaths(paths...)
	})
}

// WatchDirs configures the runner to watch recursively for file changes
// in the given directories.
func WatchDirs(paths ...string) Option {
	return Option(func(r *Runner) {
		r.watchDirs = sanitizePaths(paths...)
	})
}

// ExcludeDirs configures the runner not to listen
// to file changes in the given directories.
func ExcludeDirs(paths ...string) Option {
	return Option(func(r *Runner) {
		if len(paths) == 0 {
			r.excludeDirs = []string{}
		} else {
			r.excludeDirs = sanitizePaths(paths...)
		}
	})
}

// UseTags sets the tags for the go build command.
func UseTags(tags ...string) Option {
	return Option(func(r *Runner) {
		r.tags = tags
	})
}

// UseRaceDetector builds the binary with enabled race detector.
func UseRaceDetector(useRD bool) Option {
	return Option(func(r *Runner) {
		r.useRD = useRD
	})
}

// UseLDFlags sets the ldflags for the go build command.
func UseLDFlags(flags string) Option {
	return Option(func(r *Runner) {
		r.ldflags = flags
	})
}

// UseGCFlags sets the gcflags for the go build command.
func UseGCFlags(flags string) Option {
	return Option(func(r *Runner) {
		r.gcflags = flags
	})
}

// UseDelve will use delve to start the binary.
func UseDelve(useDlv bool, apiVerion int, port uint16, address net.IP) Option {
	return Option(func(r *Runner) {
		r.useDlv = useDlv
		r.dlvP = port
		r.dlvIP = address
		r.dlvAPIV = apiVerion
	})
}

func sanitizePaths(paths ...string) []string {
	if len(paths) == 0 {
		d, err := os.Getwd()
		if err != nil {
			panic(err)
		}
		return []string{d}
	}

	var dirs []string
	for _, p := range paths {
		if p == "" {
			p = "./"
		}

		var err error
		if p, err = filepath.Abs(p); err != nil {
			// we shouldn't get here, but if so, then we bail out completely
			panic(err)
		}
		if !containsPath(dirs, p) {
			dirs = append(dirs, p)
		}
	}
	return dirs
}

// Runner listen to file changes in *.go, go.mod and go.sum
// files and then recompiles/runs the main go program in the
// current working directory.
type Runner struct {
	goBin        string
	cmdArgs      []string
	pwd          string
	tstPwd       []string
	runTst       bool
	tstRecursive bool
	watchDirs    []string
	excludeDirs  []string
	tags         []string
	ldflags      string
	gcflags      string
	useRD        bool
	useDlv       bool
	dlvP         uint16
	dlvIP        net.IP
	dlvAPIV      int
	main         *exec.Cmd
	watcher      *fsnotify.Watcher
	rsCh         chan struct{}
	done         chan struct{}
	stopped      chan struct{}
}

// Stop stops the runner from listening to file changes and
// shuts down the main go program.
func (r *Runner) Stop() (err error) {
	if r.watcher != nil {
		log.Infoln("Stop looking for file changes")

		close(r.done)
		err = r.watcher.Close()

		// Wait for goroutines to finish
		<-r.stopped
		<-r.stopped

		os.Remove(r.getBinPath())
		r.watcher = nil
	}
	return err
}

// Watch starts listening for file changes.
func (r *Runner) Watch() (err error) {
	if r.watcher != nil {
		return errors.New("Runner already watching for file changes")
	}

	r.done = make(chan struct{})
	r.watcher, err = fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	for i := range r.watchDirs {
		log.Infoln("Adding directory to watch list: ", r.watchDirs[i])
		r.watcher.Add(r.watchDirs[i])
		filepath.Walk(r.watchDirs[i], func(path string, info os.FileInfo, err error) error {
			if err != nil || !info.IsDir() {
				return nil
			}

			if startsWithAny(r.excludeDirs, path) || strings.HasPrefix(filepath.Base(path), ".") {
				return filepath.SkipDir
			}

			r.watcher.Add(path)
			return nil
		})
	}
	go r.watch()
	go r.run()

	return nil
}

func (r *Runner) watch() {
OUTER:
	for {
		select {
		case event, ok := <-r.watcher.Events:
			if !ok {
				break OUTER
			}

			// if event is a dir add or remove it from the watcher
			if isDir(event.Name) {
				switch event.Op {
				case fsnotify.Rename:
					fallthrough
				case fsnotify.Remove:
					if !startsWithAny(r.excludeDirs, event.Name) {
						r.watcher.Remove(event.Name)
					}
				case fsnotify.Create:
					if startsWithAny(r.excludeDirs, event.Name) {
						continue
					}

					if err := r.watcher.Add(event.Name); err != nil {
						log.Warnf("Failed to add directory '%v' to the watcher: %v\n", event.Name, err)
					}
				}
			} else {
				switch {
				case strings.HasSuffix(event.Name, ".go"):
					fallthrough
				case filepath.Base(event.Name) == "go.mod":
					fallthrough
				case filepath.Base(event.Name) == "go.sum":
					select {
					case r.rsCh <- struct{}{}:
						log.Infof("File '%v' changed, recompile...\n", event.Name)
					default:
					}
				}
			}
		case err, ok := <-r.watcher.Errors:
			if !ok {
				break OUTER
			}
			log.Errorln("Error while watching files:", err)
		}
	}
	r.stopped <- struct{}{}
}

func (r *Runner) run() {
MAINLOOP:
	for {
		select {
		case <-r.done:
			break MAINLOOP
		default:
			if r.runTst {
				log.Infoln("Running tests...")
				for i := range r.tstPwd {
					select {
					case <-r.done:
						break MAINLOOP
					case <-r.rsCh:
						continue MAINLOOP
					default:
						if err := r.runTest(r.tstPwd[i]); err != nil {
							log.Errorf("Test run for '%v' failed: %v\n", r.tstPwd[i], err)
							continue
						}
						log.Infof("Test run for '%v' successful\n", r.tstPwd[i])
					}
				}
			}

			if err := r.buildMain(); err == nil {
				r.createCommand()
				if err := r.main.Start(); err != nil {
					log.Errorf("Failed to start process: %v\n", err)
				}
			} else {
				log.Errorln("Failed to build binary: ", err)
			}

			select {
			case <-r.done:
				r.killMain()
				break MAINLOOP
			case <-r.rsCh:
				r.killMain()
				continue MAINLOOP
			}
		}
	}
	r.stopped <- struct{}{}
}

func (r *Runner) createCommand() {
	r.main = &exec.Cmd{Dir: r.pwd}
	r.main.Stdout = os.Stdout
	r.main.Stderr = os.Stderr

	if r.useDlv {
		dlvPath, err := exec.LookPath("dlv")
		// If we want to run delve and we don't find it
		// we abort the program with a panic
		if err != nil {
			panic("Couldn't find dlv: " + err.Error())
		}

		args := []string{"dlv", "--headless", "--api-version", fmt.Sprintf("%v", r.dlvAPIV),
			"-l", fmt.Sprintf("%v:%v", r.dlvIP, r.dlvP), "exec", r.getBinPath()}
		if canUseContinue(dlvPath) {
			args = append(args, "--continue", "--accept-multiclient")
		}
		args = append(args, "--")

		r.main.Path = dlvPath
		r.main.Args = append(args, r.cmdArgs...)
	} else {
		r.main.Path = r.getBinPath()
		r.main.Args = append([]string{binName}, r.cmdArgs...)
	}
}

func (r Runner) buildMain() error {
	args := []string{"go", "build"}
	args = append(args, "-o", r.getBinPath())

	if len(r.tags) > 0 {
		args = append(args, "-tags", strings.Join(r.tags, " "))
	}

	if r.useRD {
		args = append(args, "-race")
	}

	if r.gcflags != "" {
		args = append(args, "-gcflags", r.gcflags)
	}

	if r.ldflags != "" {
		args = append(args, "-ldflags", r.ldflags)
	}

	os.Remove(r.getBinPath())

	log.Infof("Building binary with cmd: '%v'\n", strings.Join(args, " "))
	cmd := exec.Cmd{Dir: r.pwd, Path: r.goBin, Args: args}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (r Runner) getBinPath() string {
	p := filepath.Join(r.pwd, binName)
	if runtime.GOOS == "windows" {
		p = p + ".exe"
	}
	return p
}

func (r *Runner) killMain() {
	if r.main != nil && r.main.Process != nil {
		if err := r.main.Process.Kill(); err != nil {
			log.Errorf("Failed to kill process with pid '%v': %v \n", r.main.Process.Pid, err)
		}

		if _, err := r.main.Process.Wait(); err != nil {
			log.Warnln("Failed to wait for process:", err)
		}
		r.main = nil
	}
}

func (r Runner) runTest(path string) error {
	args := []string{"go", "test"}
	if r.tstRecursive {
		args = append(args, "./...")
	}

	cmd := exec.Cmd{Dir: path, Path: r.goBin, Args: args}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func isDir(path string) bool {
	fi, err := os.Stat(path)
	if err != nil {
		return false
	}
	return fi.IsDir()
}

func startsWithAny(prefixes []string, s string) bool {
	for i := range prefixes {
		if strings.HasPrefix(s, prefixes[i]) {
			return true
		}
	}
	return false
}

func containsPath(paths []string, path string) bool {
	for i := range paths {
		if paths[i] == path {
			return true
		}
	}
	return false
}

func canUseContinue(dlvPath string) bool {
	stdOut, err := exec.Command(dlvPath, "version").Output()
	if err != nil {
		return false
	}

	matches := dlvRegex.FindStringSubmatch(string(stdOut))
	if len(matches) != 2 {
		return false
	}

	c, _ := semver.NewConstraint(" >=1.3.0")
	v, err := semver.NewVersion(matches[1])
	if err != nil {
		return false
	}

	return c.Check(v)
}
