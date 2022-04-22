package main

import (
	"flag"
	"log"

	"github.com/HeRaNO/cdoj-execution-worker/config"
	"github.com/HeRaNO/cdoj-execution-worker/handler"
)

func main() {
	initConfigFile := flag.String("c", "./config.yaml", "the path of configure file")
	channel, msgQ := config.Init(initConfigFile)
	handler.InitTestCases()
	for req := range msgQ {
		handler.HandleReq(req, channel)
	}

	log.Panicln("[FATAL] Why execute this line???")
}
