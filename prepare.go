package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/opencontainers/runc/libcontainer"
	"github.com/opencontainers/runc/libcontainer/configs"
	"golang.org/x/sys/unix"
)

func prepareCodeFiles(files []SourceCodeDescriptor, filePath string) error {
	for _, file := range files {
		err := prepareCodeFile(file, filePath)
		if err != nil {
			ErrorLog(err, "prepareCodeFile(): WriteFile")
			return err
		}
	}
	return nil
}

func prepareCodeFile(fileDesc SourceCodeDescriptor, filePath string) error {
	fileRealPath := filepath.Join(filePath, fileDesc.Name)
	return ioutil.WriteFile(fileRealPath, []byte(fileDesc.Content), 0644)
}

func deleteCodeFiles(files []SourceCodeDescriptor, filePath string) error {
	for _, file := range files {
		err := deleteCodeFile(file, filePath)
		if err != nil {
			ErrorLog(err, "deleteCodeFile(): remove file")
			return err
		}
	}
	return nil
}

func deleteCodeFile(fileDesc SourceCodeDescriptor, filePath string) error {
	fileRealPath := filepath.Join(filePath, fileDesc.Name)
	return os.Remove(fileRealPath)
}

func prepareContainer(phase Phase, readOnly bool) (libcontainer.Container, error) {
	id, err := GenToken(20)
	if err != nil {
		return nil, err
	}
	config := BaseConfig
	cgroupsConfig := &configs.Cgroup{
		Name:   "test-container",
		Parent: "system",
		Resources: &configs.Resources{
			MemorySwappiness:  nil,
			Devices:           Devices,
			Memory:            phase.Limits.Memory,
			MemoryReservation: phase.Limits.Memory,
		},
	}
	if readOnly {
		config.ReadonlyPaths = append(config.ReadonlyPaths, "/")
	}
	config.Cgroups = cgroupsConfig
	stackLimit := phase.Limits.Memory
	if phase.Limits.Stack != nil {
		stackLimit = *phase.Limits.Stack
	}
	config.Rlimits = append(config.Rlimits, configs.Rlimit{
		Type: unix.RLIMIT_STACK,
		Hard: uint64(stackLimit),
		Soft: uint64(stackLimit),
	})
	return Factory.Create(id, &config)
}

func prepareTestCases(problemID int32) ([]TestCase, error) {
	id := fmt.Sprintf("%d", problemID)
	testCasesPath := filepath.Join(DataFilesPath, id)
	ls, err := os.ReadDir(testCasesPath)
	if err != nil {
		ErrorLog(err, "prepareTestCases(): read directory")
		return nil, err
	}
	allFilesName := make(map[string]bool, 0)
	testCasesInput := make(map[string]bool, 0)
	for _, f := range ls {
		if f.Type().IsRegular() {
			fileFullName := f.Name()
			allFilesName[fileFullName] = true
			fileExt := filepath.Ext(fileFullName)
			fileName := strings.TrimSuffix(fileFullName, fileExt)
			if fileExt == ".in" {
				testCasesInput[fileName] = true
			}
		}
	}
	testCases := make([]TestCase, 0)
	for inputName := range testCasesInput {
		outputExt := ""
		if _, ok := allFilesName[inputName+".out"]; ok {
			outputExt = ".out"
		}
		if _, ok := allFilesName[inputName+".ans"]; ok {
			if outputExt != "" {
				err := errors.New("cannot recognise answer file: multipile answer file")
				ErrorLog(err, "prepareTestCases(): find answer file")
				return nil, err
			}
			outputExt = ".ans"
		}
		if outputExt == "" {
			err := errors.New("cannot recognise answer file: no answer file")
			ErrorLog(err, "prepareTestCases(): find answer file")
			return nil, err
		}
		testCases = append(testCases, TestCase{
			Input:  filepath.Join(testCasesPath, inputName+".in"),
			Output: filepath.Join(testCasesPath, inputName+outputExt),
		})
	}
	if len(testCases) == 0 {
		err := errors.New("problemID: " + id + ": no test cases")
		ErrorLog(err, "prepareTestCases(): find answer file")
		return nil, err
	}
	return testCases, nil
}
