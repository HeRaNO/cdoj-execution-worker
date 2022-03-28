package main

import (
	"flag"
	"log"
)

func main() {
	initConfigFile := flag.String("c", "./config.yaml", "the path of configure file")
	channel, msgQ := Init(initConfigFile)

	for req := range msgQ {
		HandleReq(req, channel)
	}

	log.Panicln("[FATAL] Why execute this line???")
}
