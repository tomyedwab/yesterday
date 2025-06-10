package types

import "net/url"

type RequestParams struct {
	Path     string
	RawQuery string
	Body     string
	Cookies  map[string]string
}

type Response struct {
	Body    string
	Status  int
	Headers map[string]string
}

func (params RequestParams) Query() url.Values {
	v, _ := url.ParseQuery(params.RawQuery)
	return v
}

type CrossServiceRequest struct {
	ApplicationID string
	Path          string
	Body          string
}

type CrossServiceResponse struct {
	Status int
	Body   string
}
