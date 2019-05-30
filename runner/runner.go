package runner

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/go-sharp/go-runner/log"
)

type RunnerOption func(r *Runner)

func NewRunner(options ...RunnerOption) *Runner {
	goBin, err := exec.LookPath("go")
	if err != nil {
		panic("go compiler not found: " + err.Error())
	}

	runner := &Runner{rsCh: make(chan struct{}), goBin: goBin}
	WorkingDirectory("./")(runner)
	TestWorkingDirectories("./")(runner)
	for _, fn := range options {
		fn(runner)
	}

	return runner
}

func CommandArgs(args ...string) RunnerOption {
	return RunnerOption(func(r *Runner) {
		r.cmdArgs = args
	})
}

func RunTests(run bool) RunnerOption {
	return RunnerOption(func(r *Runner) {
		r.runTst = run
	})
}

func RecursiveTests(recursive bool) RunnerOption {
	return RunnerOption(func(r *Runner) {
		r.tstRecursive = recursive
	})
}

func WorkingDirectory(path string) RunnerOption {
	if path == "" {
		path = "./"
	}

	var err error
	if path, err = filepath.Abs(path); err != nil {
		// we shouldn't get here, but if so, then we bail out completely
		panic(err)
	}
	return RunnerOption(func(r *Runner) {
		r.pwd = path
	})
}

func TestWorkingDirectories(paths ...string) RunnerOption {
	return RunnerOption(func(r *Runner) {
		r.tstPwd = sanitizePaths(paths...)
	})
}

func WatchDirs(paths ...string) RunnerOption {
	return RunnerOption(func(r *Runner) {
		r.watchDirs = sanitizePaths(paths...)
	})
}

func ExcludeDirs(paths ...string) RunnerOption {
	return RunnerOption(func(r *Runner) {
		if len(paths) == 0 {
			r.excludeDirs = []string{}
		} else {
			r.excludeDirs = sanitizePaths(paths...)
		}
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

type Runner struct {
	goBin        string
	cmdArgs      []string
	pwd          string
	tstPwd       []string
	runTst       bool
	tstRecursive bool
	watchDirs    []string
	excludeDirs  []string
	main         *exec.Cmd
	watcher      *fsnotify.Watcher
	rsCh         chan struct{}
	done         chan struct{}
}

func (r *Runner) Stop() error {
	if r.watcher != nil {
		log.Infoln("Stop looking for file changes")
		close(r.done)
		return r.watcher.Close()
	}
	return nil
}

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
		log.Infoln("Adding directory to watch list:", r.watchDirs[i])
		r.watcher.Add(r.watchDirs[i])
		filepath.Walk(r.watchDirs[i], func(path string, info os.FileInfo, err error) error {
			if err != nil || !info.IsDir() {
				return nil
			}

			if startsAnyWith(r.excludeDirs, path) || strings.HasPrefix(filepath.Base(path), ".") {
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
	defer func() {
		r.watcher = nil
	}()
	for {
		select {
		case event, ok := <-r.watcher.Events:
			if !ok {
				return
			}

			// if event is a dir add or remove it from the watcher
			if isDir(event.Name) {
				switch event.Op {
				case fsnotify.Rename:
					fallthrough
				case fsnotify.Remove:
					if !startsAnyWith(r.excludeDirs, event.Name) {
						r.watcher.Remove(event.Name)
					}
				case fsnotify.Create:
					if startsAnyWith(r.excludeDirs, event.Name) {
						continue
					}

					if err := r.watcher.Add(event.Name); err != nil {
						log.Warnf("Failed to add directory '%v' to the watcher: %v\n", event.Name, err)
					}
				}
			} else {
				switch filepath.Ext(event.Name) {
				case ".go":
					fallthrough
				case "go.mod":
					fallthrough
				case "go.sum":
					select {
					case r.rsCh <- struct{}{}:
						log.Infof("File '%v' changed, recompile...\n", event.Name)
					default:
					}
				}
			}
		case err, ok := <-r.watcher.Errors:
			if !ok {
				return
			}
			log.Errorln("Error while watching files:", err)
		}
	}
}

func (r *Runner) run() {
MAINLOOP:
	for {
		select {
		case <-r.done:
			return
		default:
			if r.runTst {
				log.Infoln("Running tests...")
				for i := range r.tstPwd {
					select {
					case <-r.done:
						return
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

			r.main = &exec.Cmd{Dir: r.pwd, Path: r.goBin, Args: append([]string{"go", "run", "."}, r.cmdArgs...)}
			r.main.Stdout = os.Stdout
			r.main.Stderr = os.Stdout
			if err := r.main.Start(); err != nil {
				log.Errorf("Failed to start process: %v\n", err)
			}

			select {
			case <-r.done:
				r.killMain()
				return
			case <-r.rsCh:
				r.killMain()
				continue MAINLOOP
			}
		}
	}
}

func (r *Runner) killMain() {
	if r.main != nil && r.main.Process != nil {
		if err := r.main.Process.Kill(); err != nil {
			log.Errorf("Failed to kill process with pid '%v': %v \n", r.main.Process.Pid, err)
		}
		r.main = nil
	}
}

func (r *Runner) runTest(path string) error {
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

func startsAnyWith(prefixes []string, s string) bool {
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
