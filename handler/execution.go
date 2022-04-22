package handler

import (
	"context"
	"os"

	"github.com/HeRaNO/cdoj-execution-worker/model"
	"github.com/HeRaNO/cdoj-execution-worker/util"
	"github.com/opencontainers/runc/libcontainer"
)

func executeSingle(container libcontainer.Container, process *libcontainer.Process, timeLimit int32) (*model.ProcessResult, error) {
	oneErr := util.OneError{}

	err := container.Run(process)
	if err != nil {
		util.ErrorLog(err, "container.Run()")
		return nil, err
	}
	chOOM, err := container.NotifyOOM()
	if err != nil {
		util.ErrorLog(err, "container.NotifyOOM()")
		process.Signal(os.Kill)
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	go RunDaemon(ctx, process, timeLimit, chOOM, &oneErr)

	p, err := process.Wait()
	cancel()

	if err != nil {
		return &model.ProcessResult{
			ProcessState: p,
			Err:          err,
		}, nil
	}
	return &model.ProcessResult{
		ProcessState: p,
		Err:          oneErr.Err,
	}, nil
}
