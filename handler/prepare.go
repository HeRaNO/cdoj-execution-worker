package handler

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/HeRaNO/cdoj-execution-worker/config"
	"github.com/HeRaNO/cdoj-execution-worker/model"
	"github.com/HeRaNO/cdoj-execution-worker/util"
	"github.com/opencontainers/runc/libcontainer"
	"github.com/opencontainers/runc/libcontainer/configs"
	"golang.org/x/sys/unix"
)

func prepareCodeFile(fileDesc model.SourceCodeDescriptor, filePath string) error {
	fileRealPath := filepath.Join(filePath, fileDesc.Name)
	return os.WriteFile(fileRealPath, []byte(fileDesc.Content), 0644)
}

func deleteCodeFile(fileDesc model.SourceCodeDescriptor, filePath string) error {
	fileRealPath := filepath.Join(filePath, fileDesc.Name)
	return os.Remove(fileRealPath)
}

func prepareContainer(phase model.Phase, readOnly bool) (libcontainer.Container, error) {
	id, err := util.GenToken(20)
	if err != nil {
		return nil, err
	}
	conf := config.BaseConfig
	cgroupsConfig := &configs.Cgroup{
		Name:   "test-container",
		Parent: "system",
		Resources: &configs.Resources{
			MemorySwappiness:  nil,
			Devices:           config.DefaultDevices,
			Memory:            phase.Limits.Memory,
			MemoryReservation: phase.Limits.Memory,
		},
	}
	if readOnly {
		conf.ReadonlyPaths = append(conf.ReadonlyPaths, "/")
	}
	conf.Cgroups = cgroupsConfig
	stackLimit := phase.Limits.Memory
	if phase.Limits.Stack != nil {
		stackLimit = *phase.Limits.Stack
	}
	conf.Rlimits = append(conf.Rlimits, configs.Rlimit{
		Type: unix.RLIMIT_STACK,
		Hard: uint64(stackLimit),
		Soft: uint64(stackLimit),
	})
	return config.Factory.Create(id, &conf)
}

func PrepareTestCases(problemID string) ([]model.TestCase, bool, error) {
	testCasesPath := filepath.Join(config.DataFilesPath, problemID)
	ls, err := os.ReadDir(testCasesPath)
	if err != nil {
		util.ErrorLog(err, "PrepareTestCases(): read directory")
		return nil, false, err
	}
	allFilesName := make(map[string]bool, 0)
	testCasesInput := make(map[string]bool, 0)
	customChecker := false
	for _, f := range ls {
		if f.Type().IsRegular() {
			fileFullName := f.Name()
			allFilesName[fileFullName] = true
			fileExt := filepath.Ext(fileFullName)
			fileName := strings.TrimSuffix(fileFullName, fileExt)
			if fileExt == ".in" {
				testCasesInput[fileName] = true
			}
			if fileFullName == "spj.cpp" {
				customChecker = true
			}
		}
	}
	testCases := make([]model.TestCase, 0)
	for inputName := range testCasesInput {
		outputExt := ""
		if _, ok := allFilesName[inputName+".out"]; ok {
			outputExt = ".out"
		}
		if _, ok := allFilesName[inputName+".ans"]; ok {
			if outputExt != "" {
				err := errors.New("cannot recognise answer file: multipile answer file")
				util.ErrorLog(err, "PrepareTestCases(): find answer file")
				return nil, false, err
			}
			outputExt = ".ans"
		}
		if outputExt == "" {
			err := errors.New("cannot recognise answer file: no answer file")
			util.ErrorLog(err, "PrepareTestCases(): find answer file")
			return nil, false, err
		}
		testCases = append(testCases, model.TestCase{
			Input:  filepath.Join(testCasesPath, inputName+".in"),
			Output: filepath.Join(testCasesPath, inputName+outputExt),
		})
	}
	if len(testCases) == 0 {
		err := errors.New("problemID: " + problemID + ": no test cases")
		util.ErrorLog(err, "PrepareTestCases(): find answer file")
		return nil, false, err
	}
	return testCases, customChecker, nil
}
