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

/*
package endpoints_server
 implements a Google Cloud Endpoints server.

The server does simple transforms on requests that come in to /_ah/api
and then re-dispatches them to /_ah/spi.  It does not do any authentication,
quota checking, DoS checking, etc.

In addition, the server loads api configs from
/_ah/spi/BackendService.getApiConfigs prior to each call, in case the
configuration has changed.
*/
package endpoints_server
