package main

import (
	"log"

	"github.com/tomyedwab/yesterday/applib"
)

func main() {
	application, err := applib.Init("0.0.1")
	if err != nil {
		log.Fatal(err)
	}

	err = application.GetDatabase().Initialize()
	if err != nil {
		panic(err)
	}

	application.Serve()
}
