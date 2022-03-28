package main

import (
	"os"
)

type Limitation struct {
	Time   int32  `json:"time"`
	Memory int64  `json:"mem"`
	Stack  *int64 `json:"stack,omitempty"`
}

type Phase struct {
	Exec    string     `json:"exec"`
	RunArgs []string   `json:"run_args"`
	Limits  Limitation `json:"limits"`
}

type SourceCodeDescriptor struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

type CompilePhase struct {
	Compile    Phase                  `json:"compile"`
	SourceCode []SourceCodeDescriptor `json:"code"`
	ExecName   string                 `json:"exec_name"`
}

type RunPhase struct {
	Run       Phase  `json:"run"`
	ProblemID string `json:"pid"`
}

type ExecRequest struct {
	CompilePhases []CompilePhase `json:"compile_phases"`
	RunPhases     RunPhase       `json:"run_phases"`
	CheckPhase    Phase          `json:"check_phase"`
}

type ExecResult struct {
	ExitCode      int         `json:"exit_code"`
	UserTimeUsed  int64       `json:"user_time"`
	SysTimeUsed   int64       `json:"sys_time"`
	MemoryUsed    int64       `json:"memory"`
	CheckerResult *OmitString `json:"checker_res"`
}

type Response struct {
	ErrCode ErrorCode   `json:"err"`
	ErrMsg  string      `json:"msg"`
	Data    interface{} `json:"data,omitempty"`
}

type TestCase struct {
	Input  string
	Output string
}

type ProcessResult struct {
	ProcessState *os.ProcessState
	Err          error
}

type OmitString struct {
	S        string `json:"s"`
	OmitSize int64  `json:"omit_size"`
}

type CompileResult struct {
	Succeed bool
	ErrMsg  *OmitString
}
