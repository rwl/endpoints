package server_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/rwl/endpoints/server"
	"github.com/rwl/go-endpoints/endpoints"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"
	"net/http/httptest"
)

// Request type for the Message.Echo method.
type EchoReq struct {
	Message string `json:"message"`
	Delay   int    `json:"delay" endpoints:"d=2"`
}

// Response type for the MessageService.Echo method.
type EchoResp struct {
	Message string    `json:"message"`
	Date    time.Time `json:"date"`
}

// EchoService echoes a message after a delay.
type EchoService struct {
}

// Echo responds with the given message after the specified delay.
func (s *EchoService) Echo(_ *http.Request, req *EchoReq, resp *EchoResp) error {
	if req.Delay < 0 {
		req.Delay = 0
	}

	time.Sleep(time.Duration(req.Delay) * time.Second)

	resp.Message = req.Message
	resp.Date = time.Now()
	return nil
}

func ExampleEndpointsServer() {
	spi := endpoints.NewServer("")

	echoService := &EchoService{}
	echoApi, _ := spi.RegisterService(echoService, "echo",
		"v1", "Echo API", true)

	info := echoApi.MethodByName("Echo").Info()
	info.Name = "message.echo"
	info.HttpMethod = "GET"
	info.Path = "echo"
	info.Desc = "Echo messages."

	mux := http.NewServeMux()
	ts := httptest.NewServer(mux)
	defer ts.Close()

	spi.HandleHttp(mux)


	// The URL that SPI requests should be dispatched to.
	u, _ := url.Parse(ts.URL)
	api := server.NewEndpointsServer(u)

	api.HandleHttp(mux)


	message := map[string]interface{} {
		"message": "The quick red fox jumped over the lazy brown dog",
		//"delay":   2,
	}

	body, _ := json.Marshal(message)
	buf := bytes.NewBuffer(body)

	uu := ts.URL+"/_ah/api/echo/v1/echo?delay=0"

	req, _ := http.NewRequest("GET", uu, buf)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, _ := client.Do(req)

	bytes, _ := ioutil.ReadAll(resp.Body)
	resp.Body.Close()

	var jsonResp map[string]interface{}
	json.Unmarshal(bytes, &jsonResp)
	fmt.Print(jsonResp["message"])
	// Output: The quick red fox jumped over the lazy brown dog
}
