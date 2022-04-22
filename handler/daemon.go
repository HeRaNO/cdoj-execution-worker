package handler

import (
	"context"
	"os"
	"time"

	"github.com/HeRaNO/cdoj-execution-worker/config"
	"github.com/HeRaNO/cdoj-execution-worker/util"
	"github.com/opencontainers/runc/libcontainer"
)

func StopProcess(process *libcontainer.Process) {
	err := process.Signal(os.Kill)
	if err != nil {
		util.ErrorLog(err, "StopProcess(): signal")
		panic(err)
	}
}

func RunDaemon(ctx context.Context, process *libcontainer.Process, timeLimit int32, chOOM <-chan struct{}, oneErr *util.OneError) {
	ticker := time.NewTicker(util.GetWallTimeLimit(int64(timeLimit)))
	defer ticker.Stop()
	for {
		select {
		case <-chOOM:
			oneErr.Add(config.ErrOOM)
			StopProcess(process)
			return
		case <-ticker.C:
			oneErr.Add(config.ErrTLE)
			StopProcess(process)
			return
		case <-ctx.Done():
			return
		}
	}
}
