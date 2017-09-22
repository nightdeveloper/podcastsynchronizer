package main

import (
	"log"
	"fmt"
	"os"
	"io"
	"github.com/nightdeveloper/podcastsynchronizer/rsschecker"
	"github.com/nightdeveloper/podcastsynchronizer/settings"
)

func main() {

	log.Println("hello everyone!");

	// logging
	f, err := os.OpenFile("logs/app.log", os.O_APPEND | os.O_CREATE | os.O_RDWR, 0666)
	if err != nil {
		log.Fatal("error opening log file: ", err.Error())
		return;
	}

	mw := io.MultiWriter(os.Stdout, f)
	log.SetOutput(mw)

	// config
	c := settings.Config{};
	c.Load();

	// checker loop
	checker := rsschecker.NewChecker(&c)
	go checker.StartLoop();

	select {}
}