package main

import (
	"context"
	"os"

	"github.com/opencontainers/runc/libcontainer"
)

func executeSingle(container libcontainer.Container, process *libcontainer.Process, timeLimit int32) (*ProcessResult, error) {
	oneErr := OneError{}

	err := container.Run(process)
	if err != nil {
		ErrorLog(err, "container.Run()")
		return nil, err
	}
	chOOM, err := container.NotifyOOM()
	if err != nil {
		ErrorLog(err, "container.NotifyOOM()")
		process.Signal(os.Kill)
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	go RunDaemon(ctx, process, timeLimit, chOOM, &oneErr)

	p, err := process.Wait()
	cancel()

	if err != nil {
		return &ProcessResult{
			ProcessState: p,
			Err:          err,
		}, nil
	}
	return &ProcessResult{
		ProcessState: p,
		Err:          oneErr.err,
	}, nil
}
