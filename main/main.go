package main

import (
	"github.com/opsgenie/marid2/conf"
	"github.com/opsgenie/marid2/queue"
	"log"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	err := conf.ReadConfFile()
	if err != nil {
		panic(err.Error())
	}
	queueProcessor := queue.NewQueueProcessor()

	err = queueProcessor.Start()
	if err != nil {
		panic(err.Error())
	}

	queueProcessor.Wait()
}
