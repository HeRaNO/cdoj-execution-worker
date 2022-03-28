package main

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"syscall"

	jsoniter "github.com/json-iterator/go"
	"github.com/opencontainers/runc/libcontainer"
	"github.com/rabbitmq/amqp091-go"
)

func HandleReq(req amqp091.Delivery, ch *amqp091.Channel) {
	execReq := ExecRequest{}
	err := jsoniter.Unmarshal(req.Body, &execReq)

	if err != nil {
		ErrorLog(err, "Unmarshal")
		ch.Publish("", req.ReplyTo, false, false, InternalError(err, req.CorrelationId))
		req.Ack(false)
		return
	}

	testCases, err := prepareTestCases(execReq.RunPhases.ProblemID)
	if err != nil {
		ch.Publish("", req.ReplyTo, false, false, InternalError(err, req.CorrelationId))
		req.Ack(false)
		return
	}

	exePath, compileResult, parentPath, err := handleCompilePhases(execReq.CompilePhases)
	if err != nil {
		ch.Publish("", req.ReplyTo, false, false, InternalError(err, req.CorrelationId))
		req.Ack(false)
		return
	}
	compileError := false
	checkerError := false
	for name, result := range compileResult {
		if !result.Succeed {
			compileError = true
			if name == execReq.CheckPhase.Exec {
				checkerError = true
			}
		}
	}
	if checkerError {
		err := errors.New("checker compile error")
		errMsg := compileResult[execReq.CheckPhase.Exec]
		ch.Publish("", req.ReplyTo, false, false, CompileError(IE, err, errMsg.ErrMsg, req.CorrelationId))
		req.Ack(false)
		parentPath = filepath.Join(WorkDirGlobal, parentPath)
		os.RemoveAll(parentPath)
		return
	} else if compileError {
		errMsg := compileResult[execReq.RunPhases.Run.Exec]
		ch.Publish("", req.ReplyTo, false, false, CompileError(CE, nil, errMsg.ErrMsg, req.CorrelationId))
		req.Ack(false)
		parentPath = filepath.Join(WorkDirGlobal, parentPath)
		os.RemoveAll(parentPath)
		return
	}

	runPhases := execReq.RunPhases
	runTestCaseDir, ok := exePath[runPhases.Run.Exec]
	if !ok {
		err := errors.New("cannot find run directory")
		ErrorLog(err, "handleRun()")
		ch.Publish("", req.ReplyTo, false, false, InternalError(err, req.CorrelationId))
		req.Ack(false)
		parentPath = filepath.Join(WorkDirGlobal, parentPath)
		os.RemoveAll(parentPath)
		return
	}
	checkPhase := execReq.CheckPhase
	runCheckDir, ok := exePath[checkPhase.Exec]
	if !ok {
		err := errors.New("cannot find check directory")
		ErrorLog(err, "handleRun()")
		ch.Publish("", req.ReplyTo, false, false, InternalError(err, req.CorrelationId))
		req.Ack(false)
		parentPath = filepath.Join(WorkDirGlobal, parentPath)
		os.RemoveAll(parentPath)
		return
	}
	for _, testCase := range testCases {
		result, outFile, err := handleTestCaseRun(runPhases.Run, testCase.Input, runTestCaseDir)
		if err != nil {
			ch.Publish("", req.ReplyTo, false, false, InternalError(err, req.CorrelationId))
			req.Ack(false)
			break
		}
		if result.Err != nil {
			ch.Publish("", req.ReplyTo, false, false, InternalError(err, req.CorrelationId))
			break
		}
		checkerResult, err := handleCheckerRun(checkPhase, testCase, outFile, runCheckDir)
		if err != nil {
			ch.Publish("", req.ReplyTo, false, false, InternalError(err, req.CorrelationId))
			req.Ack(false)
			break
		}
		rusage := result.ProcessState.SysUsage().(*syscall.Rusage)
		runRes := ExecResult{
			ExitCode:      result.ProcessState.ExitCode(),
			UserTimeUsed:  result.ProcessState.UserTime().Nanoseconds(),
			SysTimeUsed:   result.ProcessState.SystemTime().Nanoseconds(),
			MemoryUsed:    rusage.Maxrss,
			CheckerResult: checkerResult,
		}
		os.Remove(outFile)
		ch.Publish("", req.ReplyTo, false, false, OKResp(runRes, req.CorrelationId))
	}
	parentPath = filepath.Join(WorkDirGlobal, parentPath)
	os.RemoveAll(parentPath)
	req.Ack(false)
}

func handleTestCaseRun(phase Phase, inputPath string, workDir string) (*ProcessResult, string, error) {
	workDir = filepath.Join(WorkDirInRootfs, workDir)
	container, err := prepareContainer(phase, true)
	if err != nil {
		ErrorLog(err, "prepareContainer()")
		return nil, "", errors.New("cannot init container: " + err.Error())
	}
	defer container.Destroy()
	outFileName, err := GenToken(20)
	if err != nil {
		return nil, "", errors.New("cannot create tempfile: " + err.Error())
	}
	outFilePath := filepath.Join(CacheFilesPath, outFileName)
	outFile, err := os.OpenFile(outFilePath, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		ErrorLog(err, "handleTestCaseRun(): create temp file")
		return nil, "", errors.New("cannot create temp file: " + err.Error())
	}
	inFile, err := os.Open(inputPath)
	if err != nil {
		ErrorLog(err, "handleTestCaseRun(): open input file")
		return nil, "", errors.New("cannot open input file: " + err.Error())
	}
	noNewPriv := true
	process := &libcontainer.Process{
		Args:            phase.RunArgs,
		Env:             DefaultEnv,
		User:            conf.Rootfs.WorkUser,
		Cwd:             workDir,
		Stdin:           inFile,
		Stdout:          outFile,
		Stderr:          nil,
		NoNewPrivileges: &noNewPriv,
		Init:            true,
	}
	state, err := executeSingle(container, process, phase.Limits.Time)
	if err != nil {
		return nil, "", err
	}
	return state, outFilePath, err
}

func handleCheckerRun(phase Phase, testCase TestCase, userOutput string, workDir string) (*OmitString, error) {
	workDirInRootfs := filepath.Join(WorkDirInRootfs, workDir)
	workDirGlobal := filepath.Join(WorkDirGlobal, workDir)
	container, err := prepareContainer(phase, false)
	if err != nil {
		ErrorLog(err, "prepareContainer()")
		return nil, errors.New("cannot init container: " + err.Error())
	}
	defer container.Destroy()
	errFileName, err := GenToken(20)
	if err != nil {
		return nil, errors.New("cannot create temp file: " + err.Error())
	}
	errFilePath := filepath.Join(CacheFilesPath, errFileName)
	errFile, err := os.OpenFile(errFilePath, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		ErrorLog(err, "handleCheckerRun(): open error file")
		return nil, errors.New("cannot create temp file: " + err.Error())
	}
	err = SafeCopy(testCase.Input, filepath.Join(workDirGlobal, "input"))
	if err != nil {
		return nil, errors.New("cannot copy input: " + err.Error())
	}
	err = SafeCopy(testCase.Output, filepath.Join(workDirGlobal, "answer"))
	if err != nil {
		return nil, errors.New("cannot copy answer: " + err.Error())
	}
	err = SafeCopy(userOutput, filepath.Join(workDirGlobal, "user_out"))
	if err != nil {
		return nil, errors.New("cannot copy user_out: " + err.Error())
	}
	noNewPriv := true
	process := &libcontainer.Process{
		Args:            phase.RunArgs,
		Env:             DefaultEnv,
		User:            conf.Rootfs.WorkUser,
		Cwd:             workDirInRootfs,
		Stdin:           nil,
		Stdout:          nil,
		Stderr:          errFile,
		NoNewPrivileges: &noNewPriv,
		Init:            true,
	}
	state, err := executeSingle(container, process, phase.Limits.Time)
	if err != nil {
		return nil, err
	}
	if state.Err != nil {
		ErrorLog(state.Err, "handleCheckerRun(): checker run error")
		return nil, state.Err
	}
	errMsg, err := LimitFileReader(errFilePath)
	if err != nil {
		return nil, errors.New("cannot read errFile: " + err.Error())
	}
	os.Remove(errFilePath)
	return errMsg, nil
}

func handleCompilePhase(phase Phase, workDir string) (*CompileResult, error) {
	container, err := prepareContainer(phase, false)
	if err != nil {
		ErrorLog(err, "create container")
		return nil, errors.New("cannot init container: " + err.Error())
	}
	defer container.Destroy()
	errFileName, err := GenToken(20)
	if err != nil {
		return nil, errors.New("cannot create tempfile: " + err.Error())
	}
	errFilePath := filepath.Join(CacheFilesPath, errFileName)
	errFile, err := os.OpenFile(errFilePath, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		ErrorLog(err, "handleCompilePhase(): create temp file")
		return nil, errors.New("cannot create temp file: " + err.Error())
	}
	noNewPriv := true
	process := &libcontainer.Process{
		Args:            phase.RunArgs,
		Env:             DefaultEnv,
		User:            conf.Rootfs.WorkUser,
		Cwd:             workDir,
		Stdin:           nil,
		Stdout:          nil,
		Stderr:          errFile,
		NoNewPrivileges: &noNewPriv,
		Init:            true,
	}
	state, err := executeSingle(container, process, phase.Limits.Time)
	if err != nil {
		return nil, err
	}
	errMsg, err := LimitFileReader(errFilePath)
	if err != nil {
		return nil, errors.New("cannot read errFile: " + err.Error())
	}
	err = os.Remove(errFilePath)
	if err != nil {
		err := errors.New("cannot remove tempfile: " + err.Error())
		ErrorLog(err, "handleCompilePhase(): remove file")
		return nil, err
	}
	succeed := true
	if state.Err != nil {
		succeed = false
		errMsg = &OmitString{
			S:        state.Err.Error(),
			OmitSize: 0,
		}
	}
	if state.ProcessState.ExitCode() != 0 {
		succeed = false
	}
	return &CompileResult{
		Succeed: succeed,
		ErrMsg:  errMsg,
	}, nil
}

func handleCompilePhases(phases []CompilePhase) (map[string]string, map[string]*CompileResult, string, error) {
	folderName, compileParentPath, err := Mkdir(WorkDirGlobal)
	if err != nil {
		return nil, nil, "", err
	}
	compileRootfsPath := filepath.Join(WorkDirInRootfs, folderName)
	wg := sync.WaitGroup{}
	oneErr := OneError{}
	exePathMap := sync.Map{}
	exeErrMap := sync.Map{}
	for _, phase := range phases {
		wg.Add(1)
		go func(phase CompilePhase) {
			defer wg.Done()
			compileFolderName, compilePath, err := Mkdir(compileParentPath)
			if err != nil {
				oneErr.Add(err)
				return
			}
			compilePathInRootfs := filepath.Join(compileRootfsPath, compileFolderName)
			err = prepareCodeFiles(phase.SourceCode, compilePath)
			if err != nil {
				oneErr.Add(err)
				return
			}
			msg, err := handleCompilePhase(phase.Compile, compilePathInRootfs)
			if err != nil {
				oneErr.Add(err)
				return
			}
			exeErrMap.Store(phase.ExecName, msg)
			err = deleteCodeFiles(phase.SourceCode, compilePath)
			if err != nil {
				oneErr.Add(err)
				return
			}
			exePathMap.Store(phase.ExecName, filepath.Join(folderName, compileFolderName))
		}(phase)
	}
	wg.Wait()

	exePath := make(map[string]string, 0)
	exeErr := make(map[string]*CompileResult, 0)
	exePathMap.Range(func(key, value interface{}) bool {
		k, _ := key.(string) // We trust key and value are all string
		v, _ := value.(string)
		exePath[k] = v
		return true
	})
	exeErrMap.Range(func(key, value interface{}) bool {
		k, _ := key.(string) // We trust key is string and value is OmitString
		v, _ := value.(*CompileResult)
		exeErr[k] = v
		return true
	})
	return exePath, exeErr, folderName, oneErr.err
}
