package libproxy

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/mccutchen/go-httpbin/v2/httpbin"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

type RespResult struct {
	proxyResponse   httptest.ResponseRecorder
	err             error
	requestResponse Response
}

func getResultDef(request Request) RespResult {
	return getResult(request, "validorigin1.com")
}

var (
	testServerUrl string
)

func getResult(_req Request, origin string) RespResult {
	var respResult RespResult
	marshal, err := json.Marshal(_req)
	respResult.proxyResponse = *httptest.NewRecorder()
	respResult.err = err
	if err != nil {
		return respResult
	}
	request := httptest.NewRequest("POST", "/", bytes.NewReader(marshal))
	request.Header.Set("Origin", origin)
	proxyHandler(&respResult.proxyResponse, request)
	result := respResult.proxyResponse.Result()
	err = json.NewDecoder(result.Body).Decode(&respResult.requestResponse)
	respResult.err = err
	return respResult
}

func init() {
	allowedOrigins = []string{"validorigin1.com", "validorigin2.com"}

	app := httpbin.New()
	testServer := httptest.NewServer(app.Handler())
	testServerUrl = testServer.URL
}

func checkErrorNUnmarshalHTTPBinResponse(data string, t *testing.T) HTTPBinResponse {
	var r HTTPBinResponse
	err := json.Unmarshal([]byte(data), &r)
	assert.Nil(t, err)
	return r
}

type HTTPBinResponse struct {
	Args    url.Values             `json:"args"`
	Headers http.Header            `json:"headers"`
	Origin  string                 `json:"origin"`
	URL     string                 `json:"url"`
	Data    string                 `json:"data"`
	Files   map[string]interface{} `json:"files"`
	Form    map[string]interface{} `json:"form"`
	JSON    map[string]interface{} `json:"json"`
}

// TestRedirectInCaseOriginNotSpecified
func TestNotAllowedOrigin(t *testing.T) {
	result := getResult(Request{
		Url:    testServerUrl + "/get",
		Method: "GET",
	}, "invalidorigin.com")
	// redirect in case of unknown origin
	assert.Equal(t, 301, result.proxyResponse.Code)
}

func TestWildCardOrigin(t *testing.T) {
	_allowedOrigins := allowedOrigins
	allowedOrigins = []string{"*"}
	defer func() {
		// reset allowedOrigins
		// for rest of test cases are not thread safe, will have to run one after others
		allowedOrigins = _allowedOrigins
	}()
	result := getResult(Request{
		Method: "GET",
		Url:    testServerUrl + "/get",
	}, "invalidorigin.com")
	// valid origin => 200 status
	assert.Equal(t, 200, result.proxyResponse.Code)
}

func TestUrlParamsInUrl(t *testing.T) {
	resp := getResultDef(Request{
		Method: "GET",
		Url:    testServerUrl + "/get?ram=ranga",
	})
	assert.Equal(t, 200, resp.proxyResponse.Code)
	httpBinResponse := checkErrorNUnmarshalHTTPBinResponse(resp.requestResponse.Data, t)
	// url params are sent
	assert.Equal(t, "ranga", httpBinResponse.Args.Get("ram"))
}

func TestUrlParamsInParams(t *testing.T) {
	resp := getResultDef(Request{
		Method: "GET",
		Url:    testServerUrl + "/get",
		Params: map[string]string{
			"ram": "ranga",
		},
	})
	assert.Equal(t, 200, resp.proxyResponse.Code)
	httpBinResponse := checkErrorNUnmarshalHTTPBinResponse(resp.requestResponse.Data, t)
	// url params are sent
	assert.Equal(t, "ranga", httpBinResponse.Args.Get("ram"))
}

func TestHeaders(t *testing.T) {
	resp := getResultDef(Request{
		Method: "GET",
		Url:    testServerUrl + "/get",
		Headers: map[string]string{
			"testheaderkey": "testheadervalue",
		},
	})
	assert.Equal(t, 200, resp.proxyResponse.Code)
	httpBinResponse := checkErrorNUnmarshalHTTPBinResponse(resp.requestResponse.Data, t)
	// headers are sent
	assert.Equal(t, "testheadervalue", httpBinResponse.Headers.Get("testheaderkey"))
}

func TestAccessControlHeaders(t *testing.T) {
	resp := getResult(Request{
		Method: "GET",
		Url:    testServerUrl + "/get",
	}, "validorigin2.com")
	assert.Equal(t, 200, resp.proxyResponse.Code)
	// These headers are required for browser client to read response and headers
	assert.Equal(t, "validorigin2.com", resp.proxyResponse.Header().Get("Access-Control-Allow-Origin"))
}

func TestPreflightOptionsRequest(t *testing.T) {
	request := httptest.NewRequest("OPTIONS", "/", nil)
	resp := httptest.ResponseRecorder{}
	proxyHandler(&resp, request)
	headers := resp.Header()
	// preflight request allow all origins
	assert.Equal(t, "*", headers.Get("Access-Control-Allow-Origin"))
	// preflight request allow set headers from browser
	assert.Equal(t, "*", headers.Get("Access-Control-Allow-Headers"))
}

func TestPostMethod(t *testing.T) {
	resp := getResultDef(Request{
		Method: "POST",
		Url:    testServerUrl + "/post",
	})
	// post method
	assert.Equal(t, 200, resp.proxyResponse.Code)
	checkErrorNUnmarshalHTTPBinResponse(resp.requestResponse.Data, t)
}

func TestPutMethod(t *testing.T) {
	resp := getResultDef(Request{
		Method: "PUT",
		Url:    testServerUrl + "/put",
	})
	assert.Equal(t, 200, resp.proxyResponse.Code)
	// putMethod
	checkErrorNUnmarshalHTTPBinResponse(resp.requestResponse.Data, t)
}

func TestWantsBinary(t *testing.T) {
	resp := getResultDef(Request{
		Method:      "GET",
		Url:         testServerUrl + "/get",
		WantsBinary: true,
	})
	// WantsBinary: true => response will be base64encoded
	decodeString, err := base64.RawStdEncoding.DecodeString(resp.requestResponse.Data)
	assert.Nil(t, err)
	checkErrorNUnmarshalHTTPBinResponse(string(decodeString), t)
}

func TestPostDataJson(t *testing.T) {
	request := Request{
		Method: "POST",
		Url:    testServerUrl + "/post",
		Headers: map[string]string{
			"content-type": "application/json",
		},
		Data: `{
				  "string": "simple",
				  "list": [
					"dothttp",
					"azure"
				  ],
				  "null": null,
				  "bool": false,
				  "bool2": true,
				  "float": 1.121212,
				  "float2": 1
				}`,
	}
	resp := getResultDef(request)
	response := checkErrorNUnmarshalHTTPBinResponse(resp.requestResponse.Data, t)
	assert.Equal(t, request.Data, response.Data)
}

func TestPostDataUrlencoded(t *testing.T) {
	request := Request{
		Method: "POST",
		Url:    testServerUrl + "/post",
		Headers: map[string]string{
			"content-type": "application/x-www-form-urlencoded",
		},
		Data: `ram=ranga`,
	}
	resp := getResultDef(request)
	response := checkErrorNUnmarshalHTTPBinResponse(resp.requestResponse.Data, t)
	assert.Equal(t, request.Data, response.Data)
	assert.Equal(t, "[ranga]", fmt.Sprintf("%v", response.Form["ram"]))
}

func TestPostMultipart(t *testing.T) {
	request := httptest.NewRequest("POST", "/",
		bytes.NewReader([]byte(fmt.Sprintf(`--61ed834ef57e878fad0a3d27d2b04fb1
Content-Disposition: form-data; name="proxyRequestData"

{
	"method": "POST",
	"url": "%v/post",
	"headers": {
		"content-type": "application/x-www-form-urlencoded"
	},
	"params": {
		"ram":"ranga"
	},
	"data": "",
	"wantsBinary": false
}
--61ed834ef57e878fad0a3d27d2b04fb1
Content-Disposition: form-data; name="hasi"

ranga
--61ed834ef57e878fad0a3d27d2b04fb1--
`, testServerUrl))))
	request.Header.Set("content-type", "multipart/form-data; boundary=61ed834ef57e878fad0a3d27d2b04fb1")
	request.Header.Set("origin", "validorigin1.com")
	resp := *httptest.NewRecorder()
	proxyHandler(&resp, request)
	var result Response
	err := json.NewDecoder(resp.Body).Decode(&result)
	assert.Nil(t, err)
	var r HTTPBinResponse
	json.Unmarshal([]byte(result.Data), &r)
	assert.Equal(t, "[ranga]", fmt.Sprintf("%v", r.Form["hasi"]))
}

func TestAccessTokenDisallowIncasNotAvailable(t *testing.T) {
	accessToken = "some-access-token"
	defer func() {
		accessToken = "" // delete access token(cleanup)
	}()
	request := Request{
		Method: "POST",
		Url:    testServerUrl + "/",
	}
	proxyResult := getResultDef(request)
	if proxyResult.err == nil {
		t.Error("access token is not availablie, it should error out")
	}
	var proxyRespParse map[string]interface{}
	err := json.NewDecoder(proxyResult.proxyResponse.Body).Decode(&proxyRespParse)
	assert.Nil(t, err)
	success := proxyRespParse["success"]
	assert.Equal(t, "false", fmt.Sprintf("%v", success))
}

func TestAllowWithValidAccessToken(t *testing.T) {
	accessToken = "some-access-token"
	defer func() {
		accessToken = "" // delete access token(cleanup)
	}()
	request := Request{
		Method:      "POST",
		Url:         testServerUrl + "/post",
		AccessToken: accessToken,
	}
	proxyResult := getResultDef(request)
	checkErrorNUnmarshalHTTPBinResponse(proxyResult.requestResponse.Data, t)
}

func TestInvalidAccessTokenRequestShouldFail(t *testing.T) {
	accessToken = "some-access-token"
	defer func() {
		accessToken = ""
	}()
	request := Request{
		Method:      "POST",
		Url:         testServerUrl + "/",
		AccessToken: accessToken + "1",
	}
	proxyResult := getResultDef(request)
	assert.NotNil(t, proxyResult.err)
	var proxyRespParse map[string]interface{}
	json.NewDecoder(proxyResult.proxyResponse.Body).Decode(&proxyRespParse)
	success := proxyRespParse["success"]
	assert.Equal(t, "false", fmt.Sprintf("%v", success))
}

//func TestBannedOutputs(t *testing.T) {
//	// TODO
//	// need clear understanding on banned outputs
//}

func TestBasicAuth(t *testing.T) {
	request := Request{
		Method: "GET",
		Url:    testServerUrl + "/basic-auth/username/password",
		Auth: struct {
			Username string
			Password string
		}{
			Username: "username",
			Password: "password",
		},
	}
	resp := getResultDef(request)
	assert.Equal(t, 200, resp.requestResponse.Status)
	checkErrorNUnmarshalHTTPBinResponse(resp.requestResponse.Data, t)

}

func TestBasicAuthIncorrectParams(t *testing.T) {
	// just to confirm above auth is working fine if username and password is sent wrong

	request := Request{
		Method: "GET",
		Url:    testServerUrl + "/basic-auth/username/password2",
	}
	request.Auth.Username = "username"
	request.Auth.Password = "password"
	resp := getResultDef(request)
	assert.Equal(t, 401, resp.requestResponse.Status)
	checkErrorNUnmarshalHTTPBinResponse(resp.requestResponse.Data, t)
}
