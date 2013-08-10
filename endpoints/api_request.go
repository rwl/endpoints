
package endpoints

import (
	"net/http"
	"encoding/json"
	"io/ioutil"
)

// Google's JsonRPC protocol creates a handler at /rpc for any Cloud
// Endpoints API, with api name, version, and method name being in the
// body of the request.
// If the request is sent to /rpc, we will treat it as JsonRPC.
// The client libraries for iOS's Objective C use RPC and not the REST
// versions of the API.
func is_rpc(r *http.Request) bool {
	return r.URL.Path == "rpc"
}

// Check if it's a batch request.  We'll only handle single-element batch
// requests on the dev server (and we need to handle them because that's
// what RPC and JS calls typically show up as).  Pull the request out of the
// list and record the fact that we're processing a batch.
func is_batch(r *http.Request) bool {
	var body_json interface{}
	body, _ := ioutil.ReadAll(r.Body)
	err := json.Unmarshal(body, &body_json)
	if err != nil {
		body_json = make(map[string]interface{})
	}

	if _, ok := body_json.([]interface{}); ok {
		// FIXME: Convert batch request to single request.
		return true
	}
	return false
}
