package main

import (
	"flag"
	"testing"
)

func TestHandleCompilePhases(t *testing.T) {
	initConfigFile := flag.String("c", "./config.yaml", "the path of configure file")

	InitConfig(initConfigFile)
	InitContainer()
	mainCpp := SourceCodeDescriptor{
		Name:    "main.cpp",
		Content: "#include <cstdio>\n\nint main()\n{\n\tint a, b;\n\tscanf(\"%d %d\", &a, &b);\n\tprintf(\"%d\\n\", a + b);\n\treturn 0;\n}\n",
	}
	checkCpp := SourceCodeDescriptor{
		Name:    "check.cpp",
		Content: "#include \"testlib.h\"\n#include <string>\n#include <vector>\n#include <sstream>\n\nusing namespace std;\n\nint main(int argc, char * argv[])\n{\n    setName(\"compare files as sequence of lines\");\n    registerTestlibCmd(argc, argv);\n\n    std::string strAnswer;\n\n    int n = 0;\n    while (!ans.eof()) \n    {\n        std::string j = ans.readString();\n\n        if (j == \"\" && ans.eof())\n          break;\n\n        strAnswer = j;\n        std::string p = ouf.readString();\n\n        n++;\n\n        if (j != p)\n            quitf(_wa, \"%d%s lines differ - expected: '%s', found: '%s'\", n, englishEnding(n).c_str(), compress(j).c_str(), compress(p).c_str());\n    }\n    \n    if (n == 1)\n        quitf(_ok, \"single line: '%s'\", compress(strAnswer).c_str());\n    \n    quitf(_ok, \"%d lines\", n);\n}\n",
	}
	execReq := CompilePhase{
		Compile: Phase{
			Exec:    "g++",
			RunArgs: []string{"g++", "main.cpp", "-o", "main", "-O2", "-std=c++17"},
			Limits: Limitation{
				Time:   10000,
				Memory: 1024 * 1024 * 1024,
			},
		},
		SourceCode: []SourceCodeDescriptor{
			mainCpp,
		},
		ExecName: "main",
	}
	execReq2 := CompilePhase{
		Compile: Phase{
			Exec:    "g++",
			RunArgs: []string{"g++", "check.cpp", "-o", "check", "-O2", "-std=c++17"},
			Limits: Limitation{
				Time:   10000,
				Memory: 1024 * 1024 * 1024,
			},
		},
		SourceCode: []SourceCodeDescriptor{
			checkCpp,
		},
		ExecName: "check",
	}
	exePath, compileErrMsg, parentPath, err := handleCompilePhases([]CompilePhase{execReq, execReq2})
	if err != nil {
		t.Fatal(err)
	}
	t.Log(exePath, compileErrMsg, parentPath)
}

func TestHandleTestCaseRun(t *testing.T) {
	initConfigFile := flag.String("c", "./config.yaml", "the path of configure file")

	InitConfig(initConfigFile)
	InitContainer()
	runPhase := RunPhase{
		Run: Phase{
			Exec:    "main",
			RunArgs: []string{"./main"},
			Limits: Limitation{
				Time:   1000,
				Memory: 256 * 1024 * 1024,
			},
		},
		ProblemID: "1",
	}
	state, outFile, err := handleTestCaseRun(runPhase.Run, "/home/ubuntu/dataFiles/1/1.in", "FOiK9Oly6qZjYS5OpdxK/MEllxkYJ9Pe4u5aMJpoq")
	if err != nil {
		t.Fatal(err)
	}
	t.Fatalf("%+v, %+v", state, outFile)
}

func TestHandleCheckerRun(t *testing.T) {
	initConfigFile := flag.String("c", "./config.yaml", "the path of configure file")

	InitConfig(initConfigFile)
	InitContainer()
	checkPhase := Phase{
		Exec:    "check",
		RunArgs: []string{"./check", "input", "user_out", "answer"},
		Limits: Limitation{
			Time:   5000,
			Memory: 1024 * 1024 * 1024,
		},
	}
	testCase := TestCase{
		Input:  "/home/ubuntu/dataFiles/1/1.in",
		Output: "/home/ubuntu/dataFiles/1/1.out",
	}
	errMsg, err := handleCheckerRun(checkPhase, testCase, "/home/ubuntu/cacheFiles/vBFhk4RS4kcDzcWtAi44", "FOiK9Oly6qZjYS5OpdxK/p1PrOjvSp8HH8difqa0a")
	if err != nil {
		t.Fatal(err)
	}
	t.Fatalf("%+v", errMsg)
}
