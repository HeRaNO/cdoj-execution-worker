package main

import (
	"log"
	"os"
	"runtime"

	"github.com/opencontainers/runc/libcontainer"
	_ "github.com/opencontainers/runc/libcontainer/nsenter"
	amqp "github.com/rabbitmq/amqp091-go"
)

func init() {
	if len(os.Args) > 1 && os.Args[1] == "init" {
		runtime.GOMAXPROCS(1)
		runtime.LockOSThread()
		factory, _ := libcontainer.New("")
		if err := factory.StartInitialization(); err != nil {
			log.Fatal(err)
		}
		panic("--this line should have never been executed, congratulations--")
	}
}

func Init(filePath *string) (*amqp.Channel, <-chan amqp.Delivery) {
	InitConfig(filePath)
	InitContainer()
	InitTestCases()
	return InitMQ()
}
