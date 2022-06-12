package handler

import (
	"io/fs"
	"os"
	"sync"

	"github.com/HeRaNO/cdoj-execution-worker/config"
	"github.com/HeRaNO/cdoj-execution-worker/model"
	"github.com/HeRaNO/cdoj-execution-worker/util"
)

var IDTestCasesMap map[string][]model.TestCase
var IDCustomCheckerMap map[string]bool

func InitTestCases() {
	IDTestCasesMap = make(map[string][]model.TestCase, 0)
	problems, err := os.ReadDir(config.DataFilesPath)
	if err != nil {
		util.ErrorLog(err, "ReadDir()")
		panic(err)
	}
	wg := sync.WaitGroup{}
	for _, problem := range problems {
		wg.Add(1)
		go func(wg *sync.WaitGroup, problem fs.DirEntry) {
			defer wg.Done()
			if problem.IsDir() {
				problemID := problem.Name()
				testCase, customChecker, err := PrepareTestCases(problemID)
				if err != nil {
					util.ErrorLog(err, "PrepareTestCases for problem: "+problemID)
					panic(err)
				}
				IDTestCasesMap[problemID] = testCase
				IDCustomCheckerMap[problemID] = customChecker
			}
		}(&wg, problem)
	}
	wg.Wait()
	util.InfoLog("init test cases successully", nil)
}
