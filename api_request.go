// Cloud Endpoints API request-related data and functions.
package endpoint

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
)

const API_PREFIX = "/_ah/api/"

// Simple data object representing an API request.
type ApiRequest struct {
	*http.Request

	RelativeUrl string
	// Only single-element batch requests are handled (RPC and JS calls
	// typically show up as batch requests). Pull the request out of the
	// list and record the fact that we're processing a batch.
	IsBatch   bool
	BodyJson  map[string]interface{}
	RequestId string
}

func NewApiRequest(r *http.Request) (*ApiRequest, error) {
	ar := &ApiRequest{
		Request:  r,
		IsBatch: false,
		RelativeUrl: r.URL.Path,
	}

	if !strings.HasPrefix(ar.URL.Path, API_PREFIX) {
		return nil, fmt.Errorf("Invalid request path: %s", ar.URL.Path)
	}
	ar.URL.Path = ar.URL.Path[len(_API_PREFIX):]

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("Problem parsing request body: %s", r.Body)
	}

	ar.IsBatch = false
	var bodyJson map[string]interface{}
	var bodyJsonArray []map[string]interface{}
	if len(body) > 0 {
		err := json.Unmarshal(body, &bodyJson)
		if err != nil {
			err = json.Unmarshal(body, &bodyJsonArray)
			if err != nil {
				return nil, fmt.Errorf("Problem unmarshalling request body: %s", body)
			}
			ar.IsBatch = true
		}
	} else {
		bodyJson = make(map[string]interface{})
	}
	ar.RequestId = ""

	// Check if it's a batch request.  We'll only handle single-element batch
	// requests on the dev server (and we need to handle them because that's
	// what RPC and JS calls typically show up as).  Pull the request out of the
	// list and record the fact that we're processing a batch.
	//	body_json_array, ok := body_json.([]interface{})
	if ar.IsBatch {
		switch n := len(bodyJsonArray); n {
		case 0:
			return nil, errors.New("Batch request has zero parts")
		case 1:
		default:
			log.Printf(`Batch requests with more than 1 element aren't supported. Only the first element will be handled. Found %d elements.`, n)
		}
		log.Print("Converting batch request to single request.")
		ar.BodyJson = bodyJsonArray[0]
		bodyBytes, err := json.Marshal(ar.BodyJson)
		ar.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))
		if err != nil {
			return ar, err
		}
	} else {
		ar.BodyJson = bodyJson
		ar.Body = ioutil.NopCloser(bytes.NewBuffer(body)) // reset buffer fixme: use a reader?
	}
	return ar, nil
}

func (ar *ApiRequest) Copy() (*ApiRequest, error) {
	body, err := ioutil.ReadAll(ar.Body)
	if err != nil {
		return nil, err
	}
	ar.Body.Close()
	ar.Body = ioutil.NopCloser(bytes.NewBuffer(body))
	bodyCopy := ioutil.NopCloser(bytes.NewBuffer(body))

	urlCopy := &url.URL{
		Scheme: ar.URL.Scheme,
		Opaque: ar.URL.Opaque,
		User: ar.URL.User,
		Host: ar.URL.Host,
		Path: ar.URL.Path,
		RawQuery: ar.URL.RawQuery,
		Fragment: ar.URL.Fragment,
	}
	if err != nil {
		return nil, err
	}

	headerCopy := make(http.Header)
	for k, _ := range ar.Header {
		headerCopy.Set(k, ar.Header.Get(k))
	}

	request := &http.Request{
		Method:           ar.Method,
		URL:              url_copy,
		Proto:            ar.Proto,
		ProtoMajor:       ar.ProtoMajor,
		ProtoMinor:       ar.ProtoMinor,
		Header:           header_copy,
		Body:             body_copy,
		ContentLength:    ar.ContentLength,
		TransferEncoding: ar.TransferEncoding,
		Close:            ar.Close,
		Host:             ar.Host,
		Form:             ar.Form,
		PostForm:         ar.PostForm,
		MultipartForm:    ar.MultipartForm,
		Trailer:          ar.Trailer,
		RemoteAddr:       ar.RemoteAddr,
		RequestURI:       ar.RequestURI,
		TLS:              ar.TLS,
	}

	return &ApiRequest{
		Request:    request,
		IsBatch:   ar.IsBatch,
		BodyJson:  ar.BodyJson,
		RequestId: ar.RequestId,
		RelativeUrl: ar.RelativeUrl,
	}, nil
}

// Google's JsonRPC protocol creates a handler at /rpc for any Cloud
// Endpoints API, with api name, version, and method name being in the
// body of the request.
// If the request is sent to /rpc, we will treat it as JsonRPC.
// The client libraries for iOS's Objective C use RPC and not the REST
// versions of the API.
func (ar *ApiRequest) IsRPC() bool {
	return ar.URL.Path == "rpc"
}
