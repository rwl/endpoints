
package endpoints

import (
	"testing"
	"encoding/json"
)

var api_config_map map[string]interface{}

func init() {
	json.Unmarshal(api_config_json, &api_config_map)
}

func prepare_discovery_request(status_code int, body string) {
	self.mox.StubOutWithMock(httplib.HTTPSConnection, "request")
	self.mox.StubOutWithMock(httplib.HTTPSConnection, "getresponse")
	self.mox.StubOutWithMock(httplib.HTTPSConnection, "close")

	httplib.HTTPSConnection.request(mox.IsA(basestring), mox.IsA(basestring),
		mox.IgnoreArg(), mox.IsA(dict))
	httplib.HTTPSConnection.getresponse().AndReturn(
		test_utils.MockConnectionResponse(status_code, body))
	httplib.HTTPSConnection.close()
}

func test_generate_discovery_doc_rest(t *testing.T) {
	discovery_api := &DiscoveryApiProxy{}
	baseUrl := "https://tictactoe.appspot.com/_ah/api/tictactoe/v1/"

	var body map[string]interface{}
	body["baseUrl"] = baseUrl
	prepare_discovery_request(200, json.dumps(body))

	self.mox.ReplayAll()
	doc = generate_discovery_doc(api_config_map, "rest")
	self.mox.VerifyAll()

	if doc == nil {
		t.Fail()
	}
	api_config := json.loads(doc)
	if api_config["baseUrl"] != baseUrl {
		t.Fail()
	}
}

func test_generate_discovery_doc_rpc(t *testing.T) {
	rpcUrl := "https://tictactoe.appspot.com/_ah/api/rpc"
	var body map[string]interface{}
	body["rpcUrl"] = rpcUrl
	prepare_discovery_request(200, json.dumps(body))

	self.mox.ReplayAll()
	doc = generate_discovery_doc(api_config_map, "rpc")
	self.mox.VerifyAll()

	if doc == nil {
		t.Fail()
	}
	api_config = json.loads(doc)
	if api_config["rpcUrl"] != rpcUrl {
		t.Fail()
	}
}

func test_generate_discovery_doc_invalid_format(t *testing.T) {
	discovery_api := &DiscoveryApiProxy{}

	_response := test_utils.MockConnectionResponse(400, "Error")
	doc, err := discovery_api.generate_discovery_doc(api_config_map, "blah")
	if err == nil {
		t.Fail()
	}
}

func test_generate_discovery_doc_bad_api_config(t *testing.T) {
	prepare_discovery_request(503, nil)

	mox.ReplayAll()
	doc, _ = generate_discovery_doc(`{ "name": "none" }`, "rpc")
	self.mox.VerifyAll()

	if doc != nil {
		t.Fail()
	}
}

func test_get_static_file_existing(t *testing.T) {
	body := "static file body"
	prepare_discovery_request(200, body)

	mox.ReplayAll()
	response, response_body = get_static_file("/_ah/api/static/proxy.html")
	self.mox.VerifyAll()

	if response.status != 200 {
		t.Fail()
	}
	if body != response_body {
		t.Fail()
	}
}

const api_config_json = `
{
  "extends" : "thirdParty.api",
  "abstract" : false,
  "root" : "https://tictactoe.appspot.com/_ah/api",
  "name" : "tictactoe",
  "version" : "v1",
  "description" : "",
  "defaultVersion" : false,
  "adapter" : {
    "bns" : "http://tictactoe.appspot.com/_ah/spi",
    "type" : "lily"
  },
  "methods" : {
    "tictactoe.scores.get" : {
      "path" : "scores/{key}",
      "httpMethod" : "GET",
      "rosyMethod" : "ScoreEndpoint.get",
      "request" : {
        "parameters" : {
          "key" : {
            "required" : true,
            "type" : "string"
          }
        },
	"parameterOrder" : [ "key" ],
        "body" : "autoTemplate(backendRequest)",
        "bodyName" : "resource"
      },
      "response" : {
        "body" : "autoTemplate(backendResponse)"
      }
    },
    "tictactoe.scores.list" : {
      "path" : "scores",
      "httpMethod" : "GET",
      "rosyMethod" : "ScoreEndpoint.list",
      "request" : {
        "body" : "autoTemplate(backendRequest)",
        "bodyName" : "resource"
      },
      "response" : {
        "body" : "autoTemplate(backendResponse)"
      }
    },
    "tictactoe.scores.insert" : {
      "path" : "scores",
      "httpMethod" : "POST",
      "rosyMethod" : "ScoreEndpoint.insert",
      "request" : {
        "body" : "autoTemplate(backendRequest)",
        "bodyName" : "resource"
      },
      "response" : {
        "body" : "autoTemplate(backendResponse)"
      }
    },
    "tictactoe.scores.echo1" : {
      "path" : "echo1/{value2}",
      "httpMethod" : "GET",
      "rosyMethod" : "ScoreEndpoint.echo1",
      "request" : {
        "parameters" : {
          "value2" : {
            "required" : true,
            "type" : "string"
          }
        },
	"parameterOrder" : [ "value2" ],
        "body" : "autoTemplate(backendRequest)",
        "bodyName" : "resource"
      },
      "response" : {
        "body" : "autoTemplate(backendResponse)"
      }
    },
    "tictactoe.scores.echo2" : {
      "path" : "echo2/{name}",
      "httpMethod" : "GET",
      "rosyMethod" : "ScoreEndpoint.echo2",
      "request" : {
        "parameters" : {
          "name" : {
            "required" : true,
            "type" : "string"
          }
        },
	"parameterOrder" : [ "name" ],
        "body" : "autoTemplate(backendRequest)",
        "bodyName" : "resource"
      },
      "response" : {
        "body" : "autoTemplate(backendResponse)"
      }
    },
    "tictactoe.scores.echo3" : {
      "path" : "echo3/{version}",
      "httpMethod" : "GET",
      "rosyMethod" : "ScoreEndpoint.echo3",
      "request" : {
        "parameters" : {
          "version" : {
            "required" : true,
            "type" : "string"
          }
        },
	"parameterOrder" : [ "version" ],
        "body" : "autoTemplate(backendRequest)",
        "bodyName" : "resource"
      },
      "response" : {
        "body" : "autoTemplate(backendResponse)"
      }
    },
    "tictactoe.scores.echo4" : {
      "path" : "echo4/{version2}",
      "httpMethod" : "GET",
      "rosyMethod" : "ScoreEndpoint.echo4",
      "request" : {
        "parameters" : {
          "version2" : {
            "required" : true,
            "type" : "string"
          }
        },
	"parameterOrder" : [ "version2" ],
        "body" : "autoTemplate(backendRequest)",
        "bodyName" : "resource"
      },
      "response" : {
        "body" : "autoTemplate(backendResponse)"
      }
    },
    "tictactoe.scores.cors" : {
      "path" : "cors/{echo}",
      "httpMethod" : "GET",
      "rosyMethod" : "ScoreEndpoint.cors",
      "request" : {
        "parameters" : {
          "echo" : {
            "required" : true,
            "type" : "string"
          }
        },
	"parameterOrder" : [ "echo" ],
        "body" : "autoTemplate(backendRequest)",
        "bodyName" : "resource"
      },
      "response" : {
        "body" : "autoTemplate(backendResponse)"
      }
    },
    "tictactoe.board.getmove" : {
      "path" : "board",
      "httpMethod" : "POST",
      "rosyMethod" : "BoardEndpoint.getmove",
      "request" : {
        "body" : "autoTemplate(backendRequest)",
        "bodyName" : "resource"
      },
      "response" : {
        "body" : "autoTemplate(backendResponse)"
      }
    }
  },
  "descriptor" : {
    "schemas" : {
      "Score" : {
        "id" : "Score",
        "type" : "object",
        "properties" : {
          "encodedKey" : {
            "type" : "string"
          },
          "outcome" : {
            "type" : "string"
          },
          "played" : {
            "type" : "string",
            "format" : "date"
          }
        }
      },
      "Scores" : {
        "id" : "Scores",
        "type" : "object",
        "properties" : {
          "items" : {
            "type" : "array",
            "items" : {
              "$ref" : "Score"
            }
          }
        }
      },
      "Board" : {
        "id" : "Board",
        "type" : "object",
        "properties" : {
          "state" : {
            "type" : "string"
          }
        }
      }
    },
    "methods" : {
      "ScoreEndpoint.get" : {
        "response" : {
          "$ref" : "Score"
        }
      },
      "ScoreEndpoint.list" : {
        "response" : {
          "$ref" : "Scores"
        }
      },
      "ScoreEndpoint.insert" : {
        "request" : {
          "$ref" : "Score"
        },
        "response" : {
          "$ref" : "Score"
        }
      },
      "ScoreEndpoint.echo1" : {
        "response" : {
          "$ref" : "Score"
        }
      },
      "ScoreEndpoint.echo2" : {
        "response" : {
          "$ref" : "Score"
        }
      },
      "ScoreEndpoint.echo3" : {
        "response" : {
          "$ref" : "Score"
        }
      },
      "ScoreEndpoint.echo4" : {
        "response" : {
          "$ref" : "Score"
        }
      },
      "ScoreEndpoint.cors" : {
        "response" : {
          "$ref" : "Score"
        }
      },
      "BoardEndpoint.getmove" : {
        "request" : {
          "$ref" : "Board"
        },
        "response" : {
          "$ref" : "Board"
        }
      }
    }
  }
}
`
