package handler_test

import (
	"flag"
	"testing"

	"github.com/HeRaNO/cdoj-execution-worker/config"
	"github.com/HeRaNO/cdoj-execution-worker/handler"
)

func TestPrepareTestCase(t *testing.T) {
	initConfigFile := flag.String("c", "./config.yaml", "the path of configure file")

	config.InitConfig(initConfigFile)
	testCases, customChecker, err := handler.PrepareTestCases("1")
	if err != nil {
		t.Fatal(err)
	}
	t.Error(testCases, customChecker)
}
