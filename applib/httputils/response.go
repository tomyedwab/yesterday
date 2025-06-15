package httputils

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func HandleAPIResponse(w http.ResponseWriter, r *http.Request, resp interface{}, err error, status int) {
	if err != nil {
		fmt.Printf("%s - %s %s ERROR: %v\n",
			r.RemoteAddr,
			r.Method,
			r.URL.Path,
			err,
		)
		http.Error(w, err.Error(), status)
		return
	}
	json, err := json.Marshal(resp)
	if err != nil {
		fmt.Printf("%s - %s %s ERROR: %v\n",
			r.RemoteAddr,
			r.Method,
			r.URL.Path,
			err,
		)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(json)
}
