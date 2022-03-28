package main

import (
	"context"
	"os"
	"time"

	"github.com/opencontainers/runc/libcontainer"
)

func StopProcess(process *libcontainer.Process) {
	err := process.Signal(os.Kill)
	if err != nil {
		ErrorLog(err, "StopProcess(): signal")
		panic(err)
	}
}

func RunDaemon(ctx context.Context, process *libcontainer.Process, timeLimit int32, chOOM <-chan struct{}, oneErr *OneError) {
	ticker := time.NewTicker(GetWallTimeLimit(int64(timeLimit)))
	defer ticker.Stop()
	for {
		select {
		case <-chOOM:
			oneErr.Add(ErrOOM)
			StopProcess(process)
			return
		case <-ticker.C:
			oneErr.Add(ErrTLE)
			StopProcess(process)
			return
		case <-ctx.Done():
			return
		}
	}
}
