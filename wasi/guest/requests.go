package guest

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/tomyedwab/yesterday/wasi/types"
)

func RespondSuccess(body string) types.Response {
	return types.Response{
		Body:    body,
		Status:  http.StatusOK,
		Headers: make(map[string]string),
	}
}

func RespondSuccessWithHeaders(body string, headers map[string]string) types.Response {
	return types.Response{
		Body:    body,
		Status:  http.StatusOK,
		Headers: headers,
	}
}

func RespondError(status int, err error) types.Response {
	return types.Response{
		Body:    err.Error(),
		Status:  status,
		Headers: make(map[string]string),
	}
}

func CreateResponse(ret any, err error, message string) types.Response {
	if err != nil {
		return RespondError(http.StatusInternalServerError, fmt.Errorf("%s: %v", message, err))
	}
	responseJson, err := json.Marshal(ret)
	if err != nil {
		return RespondError(http.StatusInternalServerError, fmt.Errorf("Error marshaling JSON: %v", err))
	}
	return RespondSuccess(string(responseJson))
}

type RequestHandler func(params types.RequestParams) types.Response

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

//go:wasmimport env cross_service_request
func cross_service_request(request string, destPtr *uint32) uint32

//go:wasmexport handle_request
func handle_request(paramsPtr, paramsSize, handlerId uint32) int32 {
	handler := REQUEST_HANDLERS[int(handlerId)]
	if handler == nil {
		write_response("Internal error: missing handler")
		return -1
	}

	var params types.RequestParams
	err := json.Unmarshal(GetBytesFromPtr(paramsPtr, paramsSize), &params)
	if err != nil {
		resp := RespondError(http.StatusInternalServerError, err)
		respJson, _ := json.Marshal(resp)
		write_response(string(respJson))
		return 0
	}

	response := handler(params)
	respJson, _ := json.Marshal(response)
	write_response(string(respJson))
	return 0
}

func RegisterHandler(uri string, handler RequestHandler) {
	register_handler(uri, uint32(NEXT_HANDLER_ID))
	REQUEST_HANDLERS[NEXT_HANDLER_ID] = handler
	NEXT_HANDLER_ID++
}

func CrossServiceRequest(path, applicationId string, body []byte, response interface{}) (int, error) {
	csRequest := types.CrossServiceRequest{
		Path:          path,
		ApplicationID: applicationId,
		Body:          string(body),
	}
	csRequestJson, err := json.Marshal(csRequest)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to marshal cross service request: %v", err)
	}
	var destPtr uint32
	destSize := cross_service_request(string(csRequestJson), &destPtr)

	var crossServiceResponse types.CrossServiceResponse
	responseJson := GetBytesFromPtr(destPtr, destSize)
	unmarshalErr := json.Unmarshal(responseJson, &crossServiceResponse)
	if unmarshalErr != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to unmarshal cross service response: %v", unmarshalErr)
	}

	if crossServiceResponse.Status != http.StatusOK {
		return crossServiceResponse.Status, fmt.Errorf("cross service request failed with status %d: %s", crossServiceResponse.Status, crossServiceResponse.Body)
	}
	unmarshalErr = json.Unmarshal([]byte(crossServiceResponse.Body), response)
	if unmarshalErr != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to unmarshal cross service response body: %v", unmarshalErr)
	}

	return crossServiceResponse.Status, nil
}
