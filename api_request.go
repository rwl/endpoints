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
	"io/ioutil"
	"github.com/golang/glog"
	"net/http"
	"net/url"
	"strings"
)

// Cloud Endpoints API request-related types and functions.

const apiPrefix = "/_ah/api/"

// Simple data type representing an API request.
type apiRequest struct {
	*http.Request

	relativeUrl string
	// Only single-element batch requests are handled (RPC and JS calls
	// typically show up as batch requests). Pull the request out of the
	// list and record the fact that we're processing a batch.
	isBatch   bool
	bodyJson  map[string]interface{}
	requestId string
}

func newApiRequest(r *http.Request) (*apiRequest, error) {
	ar := &apiRequest{
		Request:  r,
		isBatch: false,
		relativeUrl: r.URL.Path,
	}

	if !strings.HasPrefix(ar.URL.Path, apiPrefix) {
		return nil, fmt.Errorf("Invalid request path: %s", ar.URL.Path)
	}
	ar.URL.Path = ar.URL.Path[len(apiPrefix):]

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("Problem parsing request body: %s", r.Body)
	}

	ar.isBatch = false
	var bodyJson map[string]interface{}
	var bodyJsonArray []map[string]interface{}
	if len(body) > 0 {
		err := json.Unmarshal(body, &bodyJson)
		if err != nil {
			err = json.Unmarshal(body, &bodyJsonArray)
			if err != nil {
				return nil, fmt.Errorf("Problem unmarshalling request body: %s", body)
			}
			ar.isBatch = true
		}
	} else {
		bodyJson = make(map[string]interface{})
	}
	ar.requestId = ""

	// Check if it's a batch request.  We'll only handle single-element batch
	// requests on the dev server (and we need to handle them because that's
	// what RPC and JS calls typically show up as). Pulls the request out of
	// the list and logs the fact that we're processing a batch.
	if ar.isBatch {
		switch n := len(bodyJsonArray); n {
		case 0:
			return nil, errors.New("Batch request has zero parts")
		case 1:
		default:
			glog.Errorf(`Batch requests with more than 1 element aren't supported. Only the first element will be handled. Found %d elements.`, n)
		}
		glog.Info("Converting batch request to single request.")
		ar.bodyJson = bodyJsonArray[0]
		bodyBytes, err := json.Marshal(ar.bodyJson)
		ar.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))
		if err != nil {
			return ar, err
		}
	} else {
		ar.bodyJson = bodyJson
		ar.Body = ioutil.NopCloser(bytes.NewBuffer(body)) // reset buffer todo: use a reader?
	}
	return ar, nil
}

func (ar *apiRequest) copy() (*apiRequest, error) {
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
		URL:              urlCopy,
		Proto:            ar.Proto,
		ProtoMajor:       ar.ProtoMajor,
		ProtoMinor:       ar.ProtoMinor,
		Header:           headerCopy,
		Body:             bodyCopy,
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

	return &apiRequest{
		Request:    request,
		isBatch:   ar.isBatch,
		bodyJson:  ar.bodyJson,
		requestId: ar.requestId,
		relativeUrl: ar.relativeUrl,
	}, nil
}

// Google's JsonRPC protocol creates a handler at /rpc for any Cloud
// Endpoints API, with api name, version, and method name being in the
// body of the request.
//
// If the request is sent to /rpc, we will treat it as JsonRPC.
// The client libraries for iOS's Objective C use RPC and not the REST
// versions of the API.
func (ar *apiRequest) isRpc() bool {
	return ar.URL.Path == "rpc"
}
