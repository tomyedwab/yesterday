package httputils

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
)

func CrossServiceRequest(path, applicationID string, body []byte, response any) (int, error) {
	csReq := http.Request{
		Method: "POST",
		URL:    &url.URL{Scheme: "https", Host: "internal.yesterday.localhost:8443", Path: path},
		Header: http.Header{
			"Content-Type":     []string{"application/json"},
			"X-Application-Id": []string{applicationID},
			"Authorization":    []string{"Bearer " + os.Getenv("INTERNAL_SECRET")},
		},
		Body: io.NopCloser(bytes.NewReader([]byte(body))),
	}
	// TODO(tom) Hopefully we can come up with a better solution for
	// certificates that doesn't require disabling verification.
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	resp, err := client.Do(&csReq)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(response)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	return http.StatusOK, nil
}
