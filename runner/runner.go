package runner

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/go-sharp/go-runner/log"
)

type Runner struct {
	cmdArgs      []string
	tstArgs      []string
	pwd          string
	tstPwd       []string
	runTst       bool
	tstRecursive bool
	watchDirs    []string
	watchPattern []string
	excludeDirs  []string
	restartLimit int
	proc         *os.Process
	rsCh         chan struct{}
	done         chan struct{}
}

func (r *Runner) Watch() error {
	panic("not implemented yet")
}

func (r *Runner) run() error {
	var err error
MAINLOOP:
	for {
		select {
		case <-r.done:
			return nil
		default:
			if r.runTst {
				log.Infoln("Running tests...")
				for i := range r.tstPwd {
					select {
					case <-r.rsCh:
						continue MAINLOOP
					default:
						if err := r.runTest(r.tstPwd[i]); err != nil {
							log.Errorln("Test run for '%v' failed: %v", r.tstPwd[i], err)
							continue
						}
						log.Infoln("Test run for '%v' successful", r.tstPwd[i])
					}
				}
			}

			if r.proc != nil {
				if err := r.proc.Kill(); err != nil {
					log.Errorf("Failed to kill process with pid '%v': %v \n", r.proc.Pid, err)
				}
				r.proc = nil
			}

			for i := 0; ; i++ {
				log.Infof("Try to start process 'go run *.go %v'")
				if r.proc, err = os.StartProcess("go", append([]string{"run", "*.go"}, r.cmdArgs...), &os.ProcAttr{Dir: r.pwd}); err != nil {
					log.Errorf("Failed to start process: %v\n", err)
					if i >= r.restartLimit {
						return fmt.Errorf("maxium restart limit reached: %v", r.restartLimit)
					}
					time.Sleep(1 * time.Second)
				}
				break
			}

			state, err := r.proc.Wait()
			if err != nil {
				log.Errorln("Failed to ")
			}
			log.Infof("Process stopped with exit code: %v", state.ExitCode())
		}
	}
}

func (r *Runner) runTest(path string) error {
	args := []string{"test"}
	if r.tstRecursive {
		args = append(args, "./...")
	}
	args = append(args, r.tstArgs...)

	cmd := exec.Cmd{Dir: path, Path: "go", Args: args}
	cmd.Stdout = log.CreateWriter(log.InfoLevel, "Testing")
	cmd.Stderr = log.CreateWriter(log.ErrorLevel, "Testing")
	log.Infoln("Running test command 'go %v'", strings.Join(args, " "))
	return cmd.Run()
}
