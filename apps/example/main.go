package main

import (
	"log"
	"net/http"

	"github.com/tomyedwab/yesterday/applib"
	"github.com/tomyedwab/yesterday/applib/httputils"
)

func main() {
	application, err := applib.Init("1.0.0")
	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/api/hello", func(w http.ResponseWriter, r *http.Request) {
		httputils.HandleAPIResponse(w, r, map[string]string{
			"message": "Hello, World!",
		}, nil, http.StatusOK)
	})

	err = application.GetDatabase().Initialize()
	if err != nil {
		panic(err)
	}

	application.Serve()
}
