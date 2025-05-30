package wasi

import (
	"encoding/json"
	"net/url"
)

type RequestParams struct {
	Path     string
	RawQuery string
}

func (params RequestParams) Query() url.Values {
	v, _ := url.ParseQuery(params.RawQuery)
	return v
}

type RequestHandler func(params RequestParams) (string, error)

var REQUEST_HANDLERS map[int]RequestHandler
var NEXT_HANDLER_ID int

func InitRequests() {
	REQUEST_HANDLERS = make(map[int]RequestHandler)
	NEXT_HANDLER_ID = 1
}

//go:wasmimport env register_handler
func register_handler(uri string, handlerId uint32)

//go:wasmimport env write_response
func write_response(message string)

//go:wasmexport handle_request
func handle_request(byteHandle uint32, handlerId uint32) int32 {
	handler := REQUEST_HANDLERS[int(handlerId)]
	if handler == nil {
		write_response("Internal error: missing handler")
		return -1
	}

	var params RequestParams
	err := json.Unmarshal(GetBytes(byteHandle), &params)
	if err != nil {
		// TODO: Call a different function on error
		write_response("Internal error: json decoding")
		return 0
	}

	response, err := handler(params)
	if err != nil {
		// TODO: Call a different function on error
		write_response(response)
		return 0
	}

	write_response(response)
	return 0
}

func RegisterHandler(uri string, handler RequestHandler) {
	register_handler(uri, uint32(NEXT_HANDLER_ID))
	REQUEST_HANDLERS[NEXT_HANDLER_ID] = handler
	NEXT_HANDLER_ID++
}
