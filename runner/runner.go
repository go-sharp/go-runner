package runner

import (
	"os"

	"github.com/go-sharp/go-runner/log"
)

type Runner struct {
	cmdArgs  []string
	tstArgs  []string
	pwd      string
	tstPwd   []string
	runTst   bool
	watchDir string
	proc     *os.Process
	rsCh     chan struct{}
}

func (r *Runner) Watch() error {
	panic("not implemented yet")
}

func (r *Runner) run() error {
	for {
		if r.runTst {
			log.Infoln("Running tests...")
			for i := range r.tstPwd {

			}
		}
	}
}
