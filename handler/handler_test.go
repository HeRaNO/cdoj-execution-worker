package handler_test

import (
	"flag"
	"testing"

	"github.com/HeRaNO/cdoj-execution-worker/config"
	"github.com/HeRaNO/cdoj-execution-worker/handler"
	"github.com/HeRaNO/cdoj-execution-worker/model"
)

func TestHandleCompilePhases(t *testing.T) {
	initConfigFile := flag.String("c", "./config.yaml", "the path of configure file")

	config.InitConfig(initConfigFile)
	config.InitContainer()
	handler.InitTestCases()
	mainCpp := model.SourceCodeDescriptor{
		Name:    "main.cpp",
		Content: "#include <cstdio>\n\nint main()\n{\n\tint a, b;\n\tscanf(\"%d %d\", &a, &b);\n\tprintf(\"%d\\n\", a + b);\n\treturn 0;\n}\n",
	}
	execReq := model.CompilePhase{
		Compile: model.Phase{
			Exec:    "g++",
			RunArgs: []string{"g++", "main.cpp", "-o", "main", "-O2", "-std=c++17"},
			Limits: model.Limitation{
				Time:   10000,
				Memory: 1024 * 1024 * 1024,
			},
		},
		SourceCode: mainCpp,
		ExecName:   "main",
	}
	exePath, compileErrMsg, parentPath, err := handler.HandleCompilePhases(execReq)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(exePath, compileErrMsg, parentPath)
}

func TestHandleTestCaseRun(t *testing.T) {
	initConfigFile := flag.String("c", "./config.yaml", "the path of configure file")

	config.InitConfig(initConfigFile)
	config.InitContainer()
	handler.InitTestCases()
	runPhase := model.RunPhase{
		Run: model.Phase{
			Exec:    "main",
			RunArgs: []string{"./main"},
			Limits: model.Limitation{
				Time:   1000,
				Memory: 256 * 1024 * 1024,
			},
		},
		ProblemID: "1",
	}
	state, outFile, err := handler.HandleTestCaseRun(runPhase.Run, "/home/ubuntu/dataFiles/1/1.in", "FOiK9Oly6qZjYS5OpdxK/MEllxkYJ9Pe4u5aMJpoq")
	if err != nil {
		t.Fatal(err)
	}
	t.Fatalf("%+v, %+v", state, outFile)
}

func TestHandleCheckerRun(t *testing.T) {
	initConfigFile := flag.String("c", "./config.yaml", "the path of configure file")

	config.InitConfig(initConfigFile)
	config.InitContainer()
	handler.InitTestCases()
	checkPhase := model.Phase{
		Exec:    "check",
		RunArgs: []string{"./check", "input", "user_out", "answer"},
		Limits: model.Limitation{
			Time:   5000,
			Memory: 1024 * 1024 * 1024,
		},
	}
	testCase := model.TestCase{
		Input:  "/home/ubuntu/dataFiles/1/1.in",
		Output: "/home/ubuntu/dataFiles/1/1.out",
	}
	errMsg, err := handler.HandleCheckerRun(checkPhase, testCase, "/home/ubuntu/cacheFiles/vBFhk4RS4kcDzcWtAi44", "FOiK9Oly6qZjYS5OpdxK/p1PrOjvSp8HH8difqa0a")
	if err != nil {
		t.Fatal(err)
	}
	t.Fatalf("%+v", errMsg)
}
