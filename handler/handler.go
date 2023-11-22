package handler

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"syscall"

	"github.com/HeRaNO/cdoj-execution-worker/config"
	"github.com/HeRaNO/cdoj-execution-worker/model"
	"github.com/HeRaNO/cdoj-execution-worker/util"
	"github.com/goccy/go-json"
	"github.com/opencontainers/runc/libcontainer"
	"github.com/rabbitmq/amqp091-go"
)

func HandleReq(ctx context.Context, req amqp091.Delivery, ch *amqp091.Channel) {
	execReq := model.ExecRequest{}
	err := json.Unmarshal(req.Body, &execReq)

	if err != nil {
		util.ErrorLog(err, "Unmarshal")
		ch.PublishWithContext(ctx, "", req.ReplyTo, false, false, util.InternalError(err, req.CorrelationId))
		req.Ack(false)
		return
	}

	testCases, ok := IDTestCasesMap[execReq.RunPhases.ProblemID]
	if !ok {
		err := errors.New("cannot find test cases for problemID: " + execReq.RunPhases.ProblemID)
		ch.PublishWithContext(ctx, "", req.ReplyTo, false, false, util.InternalError(err, req.CorrelationId))
		req.Ack(false)
		return
	}

	runTestCaseDir, compileResult, parentPath, err := HandleCompilePhases(execReq.CompilePhases)
	if err != nil {
		ch.PublishWithContext(ctx, "", req.ReplyTo, false, false, util.InternalError(err, req.CorrelationId))
		req.Ack(false)
		parentPath = filepath.Join(config.WorkDirGlobal, parentPath)
		os.RemoveAll(parentPath)
		return
	}
	if !compileResult.Succeed {
		ch.PublishWithContext(ctx, "", req.ReplyTo, false, false, util.CompileError(compileResult.ErrMsg, req.CorrelationId))
		req.Ack(false)
		parentPath = filepath.Join(config.WorkDirGlobal, parentPath)
		os.RemoveAll(parentPath)
		return
	}

	checkPhase, runCheckDir, err := handleCheckerPrepare(execReq.CheckPhase, execReq.RunPhases.ProblemID, parentPath)
	if err != nil {
		ch.PublishWithContext(ctx, "", req.ReplyTo, false, false, util.InternalError(err, req.CorrelationId))
		req.Ack(false)
		parentPath = filepath.Join(config.WorkDirGlobal, parentPath)
		os.RemoveAll(parentPath)
		return
	}
	runPhases := execReq.RunPhases

	maxUserTime := int64(0)
	maxMemory := int64(0)
	failed := false

	for i, testCase := range testCases {
		ch.PublishWithContext(ctx, "", req.ReplyTo, false, false, util.RunningResp(i+1, req.CorrelationId))
		result, outFile, err := HandleTestCaseRun(runPhases.Run, testCase.Input, runTestCaseDir)
		if err != nil {
			ch.PublishWithContext(ctx, "", req.ReplyTo, false, false, util.InternalError(err, req.CorrelationId))
			failed = true
			break
		}
		if result.Err != nil || result.ProcessState.ExitCode() != 0 {
			os.Remove(outFile)
			rusage := result.ProcessState.SysUsage().(*syscall.Rusage)
			runRes := model.ExecResult{
				Case:         int32(i + 1),
				ExitCode:     result.ProcessState.ExitCode(),
				UserTimeUsed: result.ProcessState.UserTime().Nanoseconds(),
				SysTimeUsed:  result.ProcessState.SystemTime().Nanoseconds(),
				MemoryUsed:   rusage.Maxrss,
			}
			ch.PublishWithContext(ctx, "", req.ReplyTo, false, false, util.RunError(result.Err, runRes, req.CorrelationId))
			failed = true
			break
		}
		checkerResult, err := HandleCheckerRun(checkPhase, testCase, outFile, runCheckDir)
		if err != nil {
			os.Remove(outFile)
			ch.PublishWithContext(ctx, "", req.ReplyTo, false, false, util.InternalError(err, req.CorrelationId))
			failed = true
			break
		}
		os.Remove(outFile)
		if checkerResult.S[0] != 'o' {
			rusage := result.ProcessState.SysUsage().(*syscall.Rusage)
			runRes := model.ExecResult{
				Case:          int32(i + 1),
				ExitCode:      result.ProcessState.ExitCode(),
				UserTimeUsed:  result.ProcessState.UserTime().Nanoseconds(),
				SysTimeUsed:   result.ProcessState.SystemTime().Nanoseconds(),
				MemoryUsed:    rusage.Maxrss,
				CheckerResult: checkerResult,
			}
			ch.PublishWithContext(ctx, "", req.ReplyTo, false, false, util.WAResp(runRes, req.CorrelationId))
			failed = true
			break
		}
		if result.ProcessState.UserTime().Nanoseconds() > maxUserTime {
			maxUserTime = result.ProcessState.UserTime().Nanoseconds()
		}
		if result.ProcessState.SysUsage().(*syscall.Rusage).Maxrss > maxMemory {
			maxMemory = result.ProcessState.SysUsage().(*syscall.Rusage).Maxrss
		}
	}
	if !failed {
		runRes := model.ExecResult{
			UserTimeUsed: maxUserTime,
			MemoryUsed:   maxMemory,
		}
		ch.PublishWithContext(ctx, "", req.ReplyTo, false, false, util.OKResp(runRes, req.CorrelationId))
	}
	parentPath = filepath.Join(config.WorkDirGlobal, parentPath)
	os.RemoveAll(parentPath)
	req.Ack(false)
}

func HandleTestCaseRun(phase model.Phase, inputPath string, workDir string) (*model.ProcessResult, string, error) {
	workDir = filepath.Join(config.WorkDirInRootfs, workDir)
	container, err := prepareContainer(phase, true)
	if err != nil {
		util.ErrorLog(err, "prepareContainer()")
		return nil, "", errors.New("cannot init container: " + err.Error())
	}
	defer container.Destroy()
	outFileName, err := util.GenToken(20)
	if err != nil {
		return nil, "", errors.New("cannot create tempfile: " + err.Error())
	}
	outFilePath := filepath.Join(config.CacheFilesPath, outFileName)
	outFile, err := os.OpenFile(outFilePath, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		util.ErrorLog(err, "HandleTestCaseRun(): create temp file")
		return nil, "", errors.New("cannot create temp file: " + err.Error())
	}
	inFile, err := os.Open(inputPath)
	if err != nil {
		util.ErrorLog(err, "HandleTestCaseRun(): open input file")
		return nil, "", errors.New("cannot open input file: " + err.Error())
	}
	noNewPriv := true
	process := &libcontainer.Process{
		Args:            phase.RunArgs,
		Env:             config.DefaultEnv,
		User:            config.WorkUser,
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
	return state, outFilePath, nil
}

func HandleCheckerRun(phase model.Phase, testCase model.TestCase, userOutput string, workDir string) (*model.OmitString, error) {
	workDirInRootfs := filepath.Join(config.WorkDirInRootfs, workDir)
	workDirGlobal := filepath.Join(config.WorkDirGlobal, workDir)
	container, err := prepareContainer(phase, false)
	if err != nil {
		util.ErrorLog(err, "prepareContainer()")
		return nil, errors.New("cannot init container: " + err.Error())
	}
	defer container.Destroy()
	errFileName, err := util.GenToken(20)
	if err != nil {
		return nil, errors.New("cannot create temp file: " + err.Error())
	}
	errFilePath := filepath.Join(config.CacheFilesPath, errFileName)
	errFile, err := os.OpenFile(errFilePath, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		util.ErrorLog(err, "HandleCheckerRun(): open error file")
		return nil, errors.New("cannot create temp file: " + err.Error())
	}
	err = util.SafeCopy(testCase.Input, filepath.Join(workDirGlobal, "input"))
	if err != nil {
		return nil, errors.New("cannot copy input: " + err.Error())
	}
	err = util.SafeCopy(testCase.Output, filepath.Join(workDirGlobal, "answer"))
	if err != nil {
		return nil, errors.New("cannot copy answer: " + err.Error())
	}
	err = util.SafeCopy(userOutput, filepath.Join(workDirGlobal, "user_out"))
	if err != nil {
		return nil, errors.New("cannot copy user_out: " + err.Error())
	}
	noNewPriv := true
	process := &libcontainer.Process{
		Args:            phase.RunArgs,
		Env:             config.DefaultEnv,
		User:            config.WorkUser,
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
	if state.Err != nil && state.ProcessState.ExitCode() > 2 {
		util.ErrorLog(state.Err, "HandleCheckerRun(): checker run error")
		return nil, state.Err
	}
	errMsg, err := util.LimitFileReader(errFilePath)
	if err != nil {
		return nil, errors.New("cannot read errFile: " + err.Error())
	}
	os.Remove(errFilePath)
	return errMsg, nil
}

func HandleCompilePhase(phase model.Phase, workDir string) (*model.CompileResult, error) {
	container, err := prepareContainer(phase, false)
	if err != nil {
		util.ErrorLog(err, "create container")
		return nil, errors.New("cannot init container: " + err.Error())
	}
	defer container.Destroy()
	errFileName, err := util.GenToken(20)
	if err != nil {
		return nil, errors.New("cannot create tempfile: " + err.Error())
	}
	errFilePath := filepath.Join(config.CacheFilesPath, errFileName)
	errFile, err := os.OpenFile(errFilePath, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		util.ErrorLog(err, "handleCompilePhase(): create temp file")
		return nil, errors.New("cannot create temp file: " + err.Error())
	}
	noNewPriv := true
	process := &libcontainer.Process{
		Args:            phase.RunArgs,
		Env:             config.DefaultEnv,
		User:            config.WorkUser,
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
	errMsg, err := util.LimitFileReader(errFilePath)
	if err != nil {
		return nil, errors.New("cannot read errFile: " + err.Error())
	}
	err = os.Remove(errFilePath)
	if err != nil {
		err := errors.New("cannot remove tempfile: " + err.Error())
		util.ErrorLog(err, "handleCompilePhase(): remove file")
		return nil, err
	}
	succeed := true
	if state.ProcessState.ExitCode() != 0 {
		succeed = false
	}
	return &model.CompileResult{
		Succeed: succeed,
		ErrMsg:  errMsg,
	}, nil
}

func HandleCompilePhases(phase model.CompilePhase) (string, *model.CompileResult, string, error) {
	folderName, compileParentPath, err := util.Mkdir(config.WorkDirGlobal)
	if err != nil {
		return "", nil, "", err
	}
	compileRootfsPath := filepath.Join(config.WorkDirInRootfs, folderName)

	compileFolderName, compilePath, err := util.Mkdir(compileParentPath)
	if err != nil {
		return "", nil, "", err
	}
	compilePathInRootfs := filepath.Join(compileRootfsPath, compileFolderName)
	err = prepareCodeFile(phase.SourceCode, compilePath)
	if err != nil {
		util.ErrorLog(err, "prepareCodeFile()")
		return "", nil, "", err
	}
	msg, err := HandleCompilePhase(phase.Compile, compilePathInRootfs)
	if err != nil {
		return "", msg, "", err
	}
	err = deleteCodeFile(phase.SourceCode, compilePath)
	if err != nil {
		util.ErrorLog(err, "deleteCodeFile()")
		return "", msg, "", err
	}

	return filepath.Join(folderName, compileFolderName), msg, folderName, nil
}

func handleCheckerPrepare(checkMethod string, problemID string, parentPath string) (model.Phase, string, error) {
	phase := model.Phase{}
	globalParentPath := filepath.Join(config.WorkDirGlobal, parentPath)
	folderName, _, err := util.Mkdir(globalParentPath)
	if err != nil {
		return phase, "", err
	}
	checkerPath := filepath.Join(globalParentPath, folderName)
	checkerRelativePath := filepath.Join(parentPath, folderName)
	oriChecker := ""
	if checkMethod == "wcmp" {
		oriChecker = filepath.Join(config.DataFilesPath, "fecmp")
	} else {
		if !IDCustomCheckerMap[problemID] {
			return phase, "", errors.New("cannot find custom checker for problemID: " + problemID)
		}
		oriChecker = filepath.Join(config.DataFilesPath, problemID, "spj")
	}
	err = util.SafeCopy(oriChecker, filepath.Join(checkerPath, "checker"))
	if err != nil {
		return phase, "", errors.New("cannot copy fecmp: " + err.Error())
	}
	phase = model.Phase{
		Exec:    "checker",
		RunArgs: []string{"./checker", "input", "user_out", "answer"},
		Limits: model.Limitation{
			Time:   10000,
			Memory: 1024 << 20,
		},
	}
	return phase, checkerRelativePath, nil
}
