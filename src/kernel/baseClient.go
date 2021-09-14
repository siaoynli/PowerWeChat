package kernel

import (
	"fmt"
	"github.com/ArtisanCloud/go-libs/http/request"
	"github.com/ArtisanCloud/go-libs/http/response"
	"github.com/ArtisanCloud/go-libs/object"
	"github.com/ArtisanCloud/power-wechat/src/kernel/support"
	"github.com/google/uuid"
	http2 "net/http"
)

type BaseClient struct {
	*request.HttpRequest
	*response.HttpResponse

	*support.ResponseCastable

	App   *ApplicationInterface
	Token *AccessToken
}

func NewBaseClient(app *ApplicationInterface, token *AccessToken) *BaseClient {
	config := (*app).GetContainer().GetConfig()

	if token == nil {
		token = (*app).GetAccessToken()
	}

	client := &BaseClient{
		HttpRequest: request.NewHttpRequest(config),
		App:         app,
		Token:       token,
	}
	return client

}

func (client *BaseClient) HttpGet(url string, query interface{}, outHeader interface{}, outBody interface{}) (interface{}, error) {
	return client.Request(
		url,
		"GET",
		&object.HashMap{
			"query": query,
		},
		false,
		outHeader,
		outBody,
	)
}

func (client *BaseClient) HttpPost(url string, data interface{}, outHeader interface{}, outBody interface{}) (interface{}, error) {
	return client.Request(
		url,
		"POST",
		&object.HashMap{
			"form_params": data,
		},
		false,
		outHeader,
		outBody,
	)
}

func (client *BaseClient) HttpPostJson(url string, data interface{}, query interface{}, outHeader interface{}, outBody interface{}) (interface{}, error) {
	return client.Request(
		url,
		"POST",
		&object.HashMap{
			"query":       query,
			"form_params": data,
		},
		false,
		outHeader,
		outBody,
	)
}

func (client *BaseClient) HttpUpload(url string, files *object.HashMap, form *object.HashMap, query interface{}, outHeader interface{}, outBody interface{}) (interface{}, error) {

	multipart := []*object.HashMap{}
	headers := object.HashMap{}

	if form != nil {
		fileName := uuid.New().String()
		if (*form)["filename"] != nil {
			fileName = (*form)["filename"].(string)
		}
		headers["Content-Disposition"] = fmt.Sprintf("form-data; name=\"media\"; filename=\"%s\"", fileName)
	}

	if files != nil {
		for name, path := range *files {
			multipart = append(multipart, &object.HashMap{
				"name":    name,
				"value":   path,
				"headers": headers,
			})
		}
	}

	if form != nil {
		multipart = append(multipart, &object.HashMap{
			"name": (*form)["name"],
			//"filename": (*form)["filename"],
			"value": (*form)["value"],
		})
	}

	return client.Request(url, "POST", &object.HashMap{
		"query":           query,
		"multipart":       multipart,
		"connect_timeout": 30,
		"timeout":         30,
		"read_timeout":    30,
	}, false, nil, outBody)
}

func (client *BaseClient) Request(url string, method string, options *object.HashMap,
	returnRaw bool, outHeader interface{}, outBody interface{},
) (interface{}, error) {

	// to be setup middleware here
	if client.Middlewares == nil {
		client.registerHttpMiddlewares()
	}
	// http client request
	response, err := client.PerformRequest(url, method, options, returnRaw, outHeader, outBody)
	if err != nil {
		return nil, err
	}

	if returnRaw {
		return response, err
	} else {
		// tbf
		config := *(*client.App).GetContainer().Config
		var rs http2.Response = http2.Response{
			StatusCode: response.GetStatusCode(),
			Header:     response.GetHeader(),
			Body:       response.GetBody(),
		}
		returnResponse, err := client.CastResponseToType(&rs, config["response_type"].(string))
		return returnResponse, err
	}
}

func (client *BaseClient) RequestRaw(url string, method string, options *object.HashMap, outHeader interface{}, outBody interface{}) (interface{}, error) {
	return client.Request(url, method, options, true, outHeader, outBody)
}

func (client *BaseClient) registerHttpMiddlewares() {

	client.Middlewares = []interface{}{}

	// retry
	//client.PushMiddleware(client.retryMiddleware(), "retry")
	// access token
	client.PushMiddleware(client.accessTokenMiddleware(), "access_token")
	// log
	//client.PushMiddleware(client.logMiddleware(), "log")

}

// ----------------------------------------------------------------------
type MiddlewareAccessToken struct {
	*BaseClient
}
type MiddlewareLogMiddleware struct {
	*BaseClient
}
type MiddlewareRetry struct {
	*BaseClient
}

func (d *MiddlewareAccessToken) ModifyRequest(req *http2.Request) (err error) {
	accessToken := (*d.App).GetAccessToken()

	if accessToken != nil {
		config := (*d.App).GetContainer().Config
		_, err = accessToken.ApplyToRequest(req, config)
	}

	return err
}
func (d *MiddlewareLogMiddleware) ModifyRequest(req *http2.Request) error {
	fmt.Println("logMiddleware")
	return nil
}
func (d *MiddlewareRetry) ModifyRequest(req *http2.Request) error {
	fmt.Println("retryMiddleware")
	return nil
}

func (client *BaseClient) accessTokenMiddleware() interface{} {
	return &MiddlewareAccessToken{
		client,
	}
}
func (client *BaseClient) logMiddleware() interface{} {
	return &MiddlewareLogMiddleware{
		client,
	}
}
func (client *BaseClient) retryMiddleware() interface{} {
	return &MiddlewareRetry{
		client,
	}
}