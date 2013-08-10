
package endpoints

const _GET_REST_API = "apisdev.getRest"
const _GET_RPC_API = "apisdev.getRpc"
const _LIST_API = "apisdev.list"

var DISCOVERY_API_CONFIG = ApiDescriptor{
	Name: "discovery",
	Version: "v1",
	Methods: map[string]*ApiMethod{
		"discovery.apis.getRest": &ApiMethod{
			"path": "apis/{api}/{version}/rest",
			"httpMethod": "GET",
			"rosyMethod": _GET_REST_API,
		},
		"discovery.apis.getRpc": &ApiMethod{
			"path": "apis/{api}/{version}/rpc",
			"httpMethod": "GET",
			"rosyMethod": _GET_RPC_API,
		},
		"discovery.apis.list": &ApiMethod{
			"path": "apis",
			"httpMethod": "GET",
			"rosyMethod": _LIST_API,
		},
	},
}
