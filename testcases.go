package main

import "os"

var IDTestCasesMap map[string][]TestCase

func InitTestCases() {
	IDTestCasesMap = make(map[string][]TestCase, 0)
	problems, err := os.ReadDir(DataFilesPath)
	if err != nil {
		ErrorLog(err, "ReadDir()")
		panic(err)
	}
	for _, problem := range problems {
		if problem.IsDir() {
			problemID := problem.Name()
			testCase, err := prepareTestCases(problemID)
			if err != nil {
				ErrorLog(err, "prepareTestCases for problem: "+problemID)
				panic(err)
			}
			IDTestCasesMap[problemID] = testCase
		}
	}
}
