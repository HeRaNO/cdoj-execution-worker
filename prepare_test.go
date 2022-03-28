package main

import (
	"flag"
	"testing"
)

func TestPrepareTestCase(t *testing.T) {
	initConfigFile := flag.String("c", "./config.yaml", "the path of configure file")

	InitConfig(initConfigFile)
	testCases, err := prepareTestCases(1)
	if err != nil {
		t.Fatal(err)
	}
	t.Error(testCases)
}
