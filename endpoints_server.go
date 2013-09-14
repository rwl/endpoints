// Copyright 2013 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package endpoint

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/rwl/go-endpoints/endpoints"
	"io/ioutil"
	"github.com/golang/glog"
	"net/http"
	"strings"
)

const DEFAULT_URL = "http://localhost:8080"

var (
	ApiServingPattern = "_ah/api/.*" // Pattern for paths handled by this package.
	SpiRootFormat     = "/_ah/spi/%s"
	ApiExplorerUrl    = "https://developers.google.com/apis-explorer/?base="
)

var DefaultServer *EndpointsServer = NewEndpointsServer()

func HandleHttp() {
	DefaultServer.HandleHttp(nil)
}

// Dispatcher that handles requests to the built-in apiserver handlers.
type EndpointsServer struct {
	// An ApiConfigManager instance that allows a caller to set up an
	// existing configuration for testing.
	configManager *ApiConfigManager
	URL string
}

func NewEndpointsServer() *EndpointsServer {
	return NewEndpointsServerConfig(NewApiConfigManager(), DEFAULT_URL)
}

func NewEndpointsServerConfig(configManager *ApiConfigManager, url string) *EndpointsServer {
	return &EndpointsServer{configManager, url}
}

func (ed *EndpointsServer) HandleHttp(mux *http.ServeMux) {
	if mux == nil {
		mux = http.DefaultServeMux
	}
	r := NewRouter()
	r.HandleFunc("/_ah/api/explorer", ed.HandleApiExplorerRequest)
	r.HandleFunc("/_ah/api/static", ed.HandleApiStaticRequest)
	r.HandleFunc("/_ah/api/", ed.ServeHTTP)
	mux.Handle("/", r)
}

// EndpointsServer implements the http.Handler interface.
func (ed *EndpointsServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ar, err := NewApiRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
	ed.serveHTTP(w, ar)
}

func (ed *EndpointsServer) serveHTTP(w http.ResponseWriter, ar *ApiRequest) {
	// Get API configuration first. We need this so we know how to
	// call the back end.
	apiConfigResponse, err := ed.getApiConfigs()
	if err != nil {
		ed.failRequest(w, ar.Request, "BackendService.getApiConfigs error: " + err.Error())
		return
	}
	err = ed.handleApiConfigResponse(apiConfigResponse)
	if err != nil {
		ed.failRequest(w, ar.Request, "BackendService.getApiConfigs handling error: " + err.Error())
		return
	}

	// Call the service.
	_, err = ed.callSpi(w, ar)
	if err != nil {
		reqErr, ok := err.(RequestError)
		if ok {
			ed.handleRequestError(w, ar, reqErr)
			return
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

// Handler for requests to _ah/api/explorer.
func (ed *EndpointsServer) HandleApiExplorerRequest(w http.ResponseWriter, r *http.Request) {
	baseUrl := fmt.Sprintf("http://%s/_ah/api", r.URL.Host)
	redirectUrl := ApiExplorerUrl + baseUrl
	sendRedirectResponse(redirectUrl, w, r, nil)
}

// Handler for requests to _ah/api/static/.*.
func (ed *EndpointsServer) HandleApiStaticRequest(w http.ResponseWriter, r *http.Request) {
	request, err := NewApiRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	response, body, err := getStaticFile(request.RelativeUrl)
	//	status_string := fmt.Sprintf("%d %s", response.status, response.reason)
	if err == nil && response.StatusCode == 200 {
		// Some of the headers that come back from the server can't be passed
		// along in our response.  Specifically, the response from the server has
		// transfer-encoding: chunked, which doesn't apply to the response that
		// we're forwarding.  There may be other problematic headers, so we strip
		// off everything but Content-Type.
		w.Header().Add("Content-Type", response.Header.Get("Content-Type"))
		fmt.Fprintf(w, body)
	} else {
		glog.Errorf("Discovery API proxy failed on %s with %d. Details: %s",
			request.RelativeUrl, response.StatusCode, body)
		http.Error(w, body, response.StatusCode)
	}
}

// Makes a call to the BackendService.getApiConfigs endpoint.
func (ed *EndpointsServer) getApiConfigs() (*http.Response, error) {
	req, err := http.NewRequest("POST",
			ed.URL + "/_ah/spi/BackendService.getApiConfigs",
		ioutil.NopCloser(bytes.NewBufferString("{}")))
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// Verifies that a response has the expected status and content type.
// Returns true if both statusCode and contentType match the response.
func verifyResponse(response *http.Response, statusCode int, contentType string) error {
	if response.StatusCode != statusCode {
		return fmt.Errorf("HTTP status code does not match the response status code: %d != %d", statusCode, response.StatusCode)
	}
	if len(contentType) == 0 {
		return nil
	}
	ct := response.Header.Get("Content-Type")
	if len(ct) == 0 {
		return errors.New("Response does not specify a Content-Type")
	}
	if ct == contentType {
		return nil
	}
	return fmt.Errorf("Incorrect response Content-Type: %s != %s", ct, contentType)
}

// Parses the result of getApiConfigs and stores its information.
func (ed *EndpointsServer) handleApiConfigResponse(apiConfigResponse *http.Response) error {
	err := verifyResponse(apiConfigResponse, 200, "application/json")
	if err == nil {
		body, err := ioutil.ReadAll(apiConfigResponse.Body)
		defer apiConfigResponse.Body.Close()
		if err != nil {
			return err
		}
		err = ed.configManager.parseApiConfigResponse(string(body))
		if err != nil {
			return err
		}
		return nil
	}
	return err
}

// Generate SPI call (from earlier-saved request).
func (ed *EndpointsServer) callSpi(w http.ResponseWriter, origRequest *ApiRequest) (string, error) {
	var methodConfig *endpoints.ApiMethod
	var params map[string]string
	if origRequest.IsRpc() {
		methodConfig = ed.lookupRpcMethod(origRequest)
		params = nil
	} else {
		methodConfig, params = ed.lookupRestMethod(origRequest)
	}
	if methodConfig == nil {
		corsHandler := newCheckCorsHeaders(origRequest.Request)
		return sendNotFoundResponse(w, corsHandler), nil
	}

	// Prepare the request for the back end.
	spiRequest, err := ed.transformRequest(origRequest, params, methodConfig)
	if err != nil {
		return err.Error(), err
	}

	// Check if this SPI call is for the Discovery service. If so, route
	// it to our Discovery handler.
	discovery := NewDiscoveryService(ed.configManager)
	discoveryResponse, ok := discovery.handleDiscoveryRequest(spiRequest.URL.Path,
		spiRequest, w)
	if ok {
		return discoveryResponse, nil
	}

	// Send the request to the user's SPI handlers.
	url := ed.URL + fmt.Sprintf(SpiRootFormat, spiRequest.URL.Path)
	req, err := http.NewRequest("POST", url, spiRequest.Body)
	if err != nil {
		return "", err
	}
	req.Header.Add("Content-Type", "application/json")
	req.RemoteAddr = spiRequest.RemoteAddr
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	return ed.handleSpiResponse(origRequest, spiRequest, resp, w)
}

// Handle SPI response, transforming output as needed.
func (ed *EndpointsServer) handleSpiResponse(origRequest, spiRequest *ApiRequest, response *http.Response, w http.ResponseWriter) (string, error) {
	respBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return "", err
	}
	response.Body.Close()

	// Verify that the response is json.  If it isn"t treat, the body as an
	// error message and wrap it in a json error response.
	for header, value := range response.Header {
		if header == "Content-Type" && !strings.HasPrefix(value[0], "application/json") {
			return ed.failRequest(w, origRequest.Request,
				fmt.Sprintf("Non-JSON reply: %s", respBody)), nil
		}
	}

	err = ed.checkErrorResponse(response)
	if err != nil {
		return "", err
	}

	// Need to check IsRpc() against the original request, because the
	// incoming request here has had its path modified.
	var body string
	if origRequest.IsRpc() {
		body, err = ed.transformJsonrpcResponse(spiRequest, string(respBody))
	} else {
		body, err = ed.transformRestResponse(string(respBody))
	}
	if err != nil {
		return body, err
	}

	corsHandler := newCheckCorsHeaders(origRequest.Request)
	corsHandler.updateHeaders(w.Header())
	for k, vals := range response.Header {
		w.Header()[k] = vals
	}
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(body)))
	w.WriteHeader(response.StatusCode)
	fmt.Fprint(w, body)
	return body, nil
}

// Write an immediate failure response to outfile, no redirect.
func (ed *EndpointsServer) failRequest(w http.ResponseWriter, origRequest *http.Request, message string) string {
	corsHandler := newCheckCorsHeaders(origRequest)
	return sendErrorResponse(message, w, corsHandler)
}

// Looks up and returns rest method for the currently-pending request.
//
// Returns a method descriptor and a parameter map, or (nil, nil) if no
// method was found for the current request.
func (ed *EndpointsServer) lookupRestMethod(origRequest *ApiRequest) (*endpoints.ApiMethod, map[string]string) {
	methodName, method, params := ed.configManager.lookupRestMethod(origRequest.URL.Path, origRequest.Method)
	origRequest.Method = methodName
	return method, params
}

// Looks up and returns RPC method for the currently-pending request.
//
// Returns the RPC method descriptor that was found for the current request,
// or nil if none was found.
func (ed *EndpointsServer) lookupRpcMethod(origRequest *ApiRequest) *endpoints.ApiMethod {
	if origRequest.BodyJson == nil {
		return nil
	}
	methodName, ok := origRequest.BodyJson["method"]
	methodNameStr, ok2 := methodName.(string)
	if !ok || !ok2 {
		methodNameStr = ""
	}
	version, ok := origRequest.BodyJson["apiVersion"]
	versionStr, ok3 := version.(string)
	if !ok || !ok3 {
		versionStr = ""
	}
	origRequest.Method = methodNameStr
	return ed.configManager.lookupRpcMethod(methodNameStr, versionStr)
}

// Transforms origRequest to an api-serving request.
//
// This method uses origRequest to determine the currently-pending request
// and returns a new transformed request ready to send to the SPI. The path
// is updated and parts of the body or other properties may also be changed.
// This method accepts a REST-style or RPC-style request.
func (ed *EndpointsServer) transformRequest(origRequest *ApiRequest, params map[string]string, methodConfig *endpoints.ApiMethod) (*ApiRequest, error) {
	var request *ApiRequest
	var err error
	if origRequest.IsRpc() {
		request, err = ed.transformJsonrpcRequest(origRequest)
	} else {
		methodParams := methodConfig.Request.Params
		request, err = ed.transformRestRequest(origRequest, params, methodParams)
	}
	if err != nil {
		return request, err
	}
	request.URL.Path = methodConfig.RosyMethod
	return request, nil
}

// Checks if the parameter value is valid if an enum.
//
// If the parameter is not an enum, it does nothing. If it is, verifies that
// its value is valid.
//
// Takes the name of the parameter (Which is either just a variable name or
// the name with the index appended. For example "var" or "var[2]".), the
// value to be used as enum for the parameter and a spec containing
// information specific to the field in question (This is retrieved from
// request.Parameters in the method config.
//
// Returns an EnumRejectionError if the given value is not among the accepted
// enum values in the field parameter.
func (ed *EndpointsServer) checkEnum(parameterName string, value string, fieldParameter *endpoints.ApiRequestParamSpec) *EnumRejectionError {
	if fieldParameter == nil || fieldParameter.Enum == nil || len(fieldParameter.Enum) == 0 {
		return nil
	}

	enumValues := make([]string, 0)
	for _, enum := range fieldParameter.Enum {
		if enum.BackendVal != "" {
			enumValues = append(enumValues, enum.BackendVal)
		}
	}

	for _, ev := range enumValues {
		if value == ev {
			return nil
		}
	}
	return NewEnumRejectionError(parameterName, value, enumValues)
}

// Recursively calls checkParameter on the values in the list.
//
// "[index-of-value]" is appended to the parameter name for
// error reporting purposes.
func (ed *EndpointsServer) checkParameters(parameterName string, values []string, fieldParameter *endpoints.ApiRequestParamSpec) *EnumRejectionError {
	for index, element := range values {
		parameterNameIndex := fmt.Sprintf("%s[%d]", parameterName, index)
		err := ed.checkParameter(parameterNameIndex, element, fieldParameter)
		if err != nil {
			return err
		}
	}
	return nil
}

// Checks if the parameter value is valid against all parameter rules.
//
// Currently only checks if value adheres to enum rule, but more checks may be
// added.
//
// Takes the name of the parameter (Which is either just a variable name or
// the name with the index appended. For example "var" or "var[2]".), the
// value to be used as enum for the parameter and a spec containing
// information specific to the field in question (This is retrieved from
// request.Parameters in the method config.
func (ed *EndpointsServer) checkParameter(parameterName, value string, fieldParameter *endpoints.ApiRequestParamSpec) *EnumRejectionError {
	return ed.checkEnum(parameterName, value, fieldParameter)
}

// Converts a . delimitied field name to a message field in parameters.
//
// This adds the field to the params map, broken out so that message
// parameters appear as sub-dicts within the outer param.
//
// For example:
//
//		{"a.b.c": ["foo"]}
//
// becomes:
//
//		{"a": {"b": {"c": ["foo"]}}}
//
// The params argument is a map holding all the parameters, where the value
// is eventually set.
func (ed *EndpointsServer) addMessageField(fieldName string, value interface{}, params map[string]interface{}) {
	pos := strings.Index(fieldName, ".")
	if pos == -1 {
		params[fieldName] = value
		return
	}

	substr := strings.SplitN(fieldName, ".", 2)
	root, remaining := substr[0], substr[1]

	var subParams map[string]interface{}
	_subParams, ok := params[root]
	if ok {
		subParams, ok = _subParams.(map[string]interface{})
		if !ok {
			glog.Errorf("Problem accessing sub-params: %#v", _subParams)
		}
	} else {
		subParams = make(map[string]interface{})
		params[root] = subParams
	}
	ed.addMessageField(remaining, value, subParams)
}

// Updates the dictionary for an API payload with the request body.
//
// The values from the body should override those already in the payload,
// but for nested fields (message objects) the values can be combined
// recursively.
//
// Takes a map containing an API payload parsed from the path and query
// parameters in a request and a map parsed from the body of the request.
func (ed *EndpointsServer) updateFromBody(destination map[string]interface{}, source map[string]interface{}) {
	for key, value := range source {
		destinationValue, ok := destination[key]
		if ok {
			val, okVal := value.(map[string]interface{})
			destValue, okDest := destinationValue.(map[string]interface{})
			if okVal && okDest {
				ed.updateFromBody(destValue, val)
			} else {
				destination[key] = value
			}
		} else {
			destination[key] = value
		}
	}
}

// Translates a REST request into an an api-serving request.
//
// This makes a copy of origRequest and transforms it to api-serving
// format (moving request parameters to the body).
//
// The request can receive values from the path, query and body and combine
// them before sending them along to the SPI server. In cases of collision,
// objects from the body take precedence over those from the query, which in
// turn take precedence over those from the path.
//
// In the case that a repeated value occurs in both the query and the path,
// those values can be combined, but if that value also occurred in the body,
// it would override any other values.
//
// In the case of nested values from message fields, non-colliding values
// from subfields can be combined. For example, if "?a.c=10" occurs in the
// query string and "{"a": {"b": 11}}" occurs in the body, then they will be
// combined as
//
//	{
//	  "a": {
//	    "b": 11,
//	    "c": 10,
//	  }
//	}
//
// before being sent to the SPI server.
//
// Takes the original request from the user, a map with URL path parameters
// extracted by the configManager lookup, a map containing the API
// configuration for the parameters for the request and returns a copy of
// the current request that's been modified so it can be sent to the SPI.
// The body is updated to include parameters from the URL.
func (ed *EndpointsServer) transformRestRequest(origRequest *ApiRequest,
params map[string]string,
methodParameters map[string]*endpoints.ApiRequestParamSpec) (*ApiRequest, error) {
	request, err := origRequest.Copy()
	if err != nil {
		return request, err
	}
	bodyJson := make(map[string]interface{})

	// Handle parameters from the URL path.
	for key, value := range params {
		// Values need to be in a list to interact with query parameter values
		// and to account for case of repeated parameters
		bodyJson[key] = []string{value}
	}

	// Add in parameters from the query string.
	if len(request.URL.RawQuery) > 0 {
		// For repeated elements, query and path work together
		for key, value := range request.URL.Query() {
			if jsonVal, ok := bodyJson[key]; ok {
				jsonArr, ok := jsonVal.([]string)
				if ok {
					bodyJson[key] = append(value, jsonArr...)
				} else {
					panic(fmt.Sprintf("String array expected: %#v", jsonVal))
				}
			} else {
				bodyJson[key] = value
			}
		}
	}

	// Validate all parameters we've merged so far and convert any "." delimited
	// parameters to nested parameters. We don't use iteritems since we may
	// modify bodyJson within the loop. For instance, "a.b" is not a valid key
	// and would be replaced with "a".
	for key, _ := range bodyJson {
		currentParameter, ok := methodParameters[key]
		repeated := false
		if ok {
			repeated = currentParameter.Repeated
		}

		if !repeated {
			val := bodyJson[key]
			valArr, ok := val.([]string)
			if ok {
				if len(valArr) > 0 {
					bodyJson[key] = valArr[0]
				} else {
					bodyJson[key] = "" //delete?
				}
			} else {
				panic(fmt.Sprintf("String array expected: %#v", val))
			}
		}

		// Order is important here. Parameter names are dot-delimited in
		// parameters instead of nested in maps as a message field is, so
		// we need to call checkParameter on them before calling
		// addMessageField.

		value := bodyJson[key]
		valStr, ok := value.(string)
		var enumErr *EnumRejectionError = nil
		if ok {
			enumErr = ed.checkParameter(key, valStr, currentParameter)
		} else if valStrArr, ok := value.([]string); ok {
			enumErr = ed.checkParameters(key, valStrArr, currentParameter)
		} else {
			panic(fmt.Sprintf("Param value not a string or string array: %v",
				value))
		}
		if enumErr == nil {
			// Remove the old key and try to convert to nested message value
			delete(bodyJson, key)
			ed.addMessageField(key, value, bodyJson)
		} else {
			return request, enumErr
		}
	}

	// Add in values from the body of the request.
	if request.BodyJson != nil {
		ed.updateFromBody(bodyJson, request.BodyJson)
	}

	//request.BodyJson = bodyJson
	body, err := json.Marshal(bodyJson)
	if err == nil {
		request.Body = ioutil.NopCloser(bytes.NewBuffer(body))
	} else {
		return request, err
	}
	json.Unmarshal(body, &request.BodyJson) // map[string]interface{}, no string or []string
	return request, nil
}

// Translates a JsonRpc request/response into an api-serving request/response.
//
// Returns a new request with the request_id updated and params moved to the
// body.
func (ed *EndpointsServer) transformJsonrpcRequest(origRequest *ApiRequest) (*ApiRequest, error) {
	request, err := origRequest.Copy()
	if err != nil {
		return request, err
	}

	requestId, okId := request.BodyJson["id"]
	if okId {
		requestIdStr, ok := requestId.(string)
		if ok {
			request.RequestId = requestIdStr
		} else {
			requestIdInt, ok := requestId.(int)
			if ok {
				request.RequestId = fmt.Sprintf("%d", requestIdInt)
			} else {
				return nil, fmt.Errorf("Problem extracting request ID: %#v", requestId)
			}
		}
	} else {
		request.RequestId = ""
	}

	bodyJson, okParam := request.BodyJson["params"]
	if okParam {
		bodyJsonObj, ok := bodyJson.(map[string]interface{})
		if ok {
			request.BodyJson = bodyJsonObj
		} else {
			bodyJsonMap, ok := bodyJson.(map[string]interface{})
			if ok {
				request.BodyJson = bodyJsonMap
			} else {
				return nil, fmt.Errorf("Problem extracting JSON body from params: %#v", bodyJson)
			}
		}
	} else {
		request.BodyJson = make(map[string]interface{})
	}

	body, err := json.Marshal(request.BodyJson)
	if err != nil {
		return request, fmt.Errorf("Problem transforming RPC request: %s", err.Error())
	}
	request.Body = ioutil.NopCloser(bytes.NewBuffer(body))
	return request, nil
}

// Returns an error if the response from the SPI was an error.
//
// Returns a BackendError if the response is an error.
func (ed *EndpointsServer) checkErrorResponse(response *http.Response) error {
	if response.StatusCode >= 300 {
		return NewBackendError(response)
	}
	return nil
}

// Translates an apiserving REST response so it's ready to return.
//
// Currently, the only thing that needs to be fixed here is indentation,
// so it's consistent with what a GAE app would return.
//
// Returns a reformatted version of the response JSON.
func (ed *EndpointsServer) transformRestResponse(responseBody string) (string, error) {
	var bodyJson map[string]interface{}
	err := json.Unmarshal([]byte(responseBody), &bodyJson)
	if err != nil {
		return responseBody, fmt.Errorf("Problem transforming REST response: %s", err.Error())
	}
	body, _ := json.MarshalIndent(bodyJson, "", "  ") // todo: sort keys
	return string(body), nil
}

// Translates an api-serving response to a JsonRpc response.
//
// Returns the updated, JsonRPC-formatted request body.
func (ed *EndpointsServer) transformJsonrpcResponse(spiRequest *ApiRequest, responseBody string) (string, error) {
	var result interface{}
	err := json.Unmarshal([]byte(responseBody), &result)
	if err != nil {
		return responseBody, fmt.Errorf("Problem unmarshalling RPC response: %s", err.Error())
	}
	bodyJson := map[string]interface{}{"result": result}
	return ed.finishRpcResponse(spiRequest.RequestId, spiRequest.IsBatch, bodyJson), nil
}

// Finish adding information to a JSON RPC response.
//
// The requestId argument may be empty if the request didn't have a
// request ID. Returns the updated, JsonRPC-formatted request body.
func (ed *EndpointsServer) finishRpcResponse(requestId string, isBatch bool, bodyJson map[string]interface{}) string {
	if len(requestId) > 0 {
		bodyJson["id"] = requestId
	}
	var body []byte
	if isBatch {
		body, _ = json.MarshalIndent([]map[string]interface{}{bodyJson}, "", "  ")
	} else {
		body, _ = json.MarshalIndent(bodyJson, "", "  ") // todo: sort keys
	}
	return string(body)
}

func (ed *EndpointsServer) handleRequestError(w http.ResponseWriter, origRequest *ApiRequest, err RequestError) string {
	var statusCode int
	var body string
	if origRequest.IsRpc() {
		// JSON RPC errors are returned with status 200 OK and the
		// error details in the body.
		statusCode = 200
		_id, _ := origRequest.BodyJson["id"]
		id, ok := _id.(string)
		if !ok {
			// fixme: handle type assertion failure
		}
		body = ed.finishRpcResponse(id, origRequest.IsBatch, err.RpcError())
	} else {
		statusCode = err.StatusCode()
		body = err.RestError()
	}

	//response_status = fmt.Sprintf("%d %s", status_code,
	//	http.StatusText(status_code)) // fixme: handle unknown status code "Unknown Error"

	newCheckCorsHeaders(origRequest.Request).updateHeaders(w.Header())
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(body)))
	w.WriteHeader(statusCode)
	fmt.Fprint(w, body)
	return body
}
