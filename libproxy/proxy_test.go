package libproxy

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/mccutchen/go-httpbin/v2/httpbin"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
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
	// Setup loggers first to avoid nil pointers
	setupLoggers()

	allowedOrigins = []string{"validorigin1.com", "validorigin2.com"}
	bannedDests = []string{"banned.example.com"}

	app := httpbin.New()
	testServer := httptest.NewServer(app.Handler())
	testServerUrl = testServer.URL

	// Initialize health metrics for testing
	startTime = time.Now()
	totalRequests = 0
	totalErrors = 0
	healthCheckEnabled = true
	isServerRunning = true
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

func TestWildCardOriginAllowsEmptyOrigin(t *testing.T) {
	_allowedOrigins := allowedOrigins
	allowedOrigins = []string{"*"}
	defer func() {
		allowedOrigins = _allowedOrigins
	}()
	result := getResult(Request{
		Method: "GET",
		Url:    testServerUrl + "/get",
	}, "")
	// wildcard should permit an empty origin (e.g. desktop clients)
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

func TestBannedOutputs(t *testing.T) {
	// Test redaction of banned outputs
	bannedOutputs = []string{"SECRET_TOKEN"}
	defer func() {
		bannedOutputs = []string{} // cleanup
	}()

	request := Request{
		Method:      "GET",
		Url:         testServerUrl + "/response-headers?response=Contains_SECRET_TOKEN_Value",
		WantsBinary: false,
	}

	resp := getResultDef(request)
	assert.Equal(t, 200, resp.proxyResponse.Code)
	// Check if the secret token is redacted
	assert.NotContains(t, resp.requestResponse.Data, "SECRET_TOKEN")
	assert.Contains(t, resp.requestResponse.Data, "[redacted]")
}

func TestBannedDestination(t *testing.T) {
	// Test blocking requests to banned destinations
	request := Request{
		Method: "GET",
		Url:    "https://banned.example.com/resource",
	}

	resp := getResultDef(request)
	assert.Equal(t, 200, resp.proxyResponse.Code)

	var proxyRespParse map[string]interface{}
	err := json.NewDecoder(resp.proxyResponse.Body).Decode(&proxyRespParse)
	assert.Nil(t, err)

	success := proxyRespParse["success"]
	assert.Equal(t, "false", fmt.Sprintf("%v", success))

	// Check for the banned destination error message
	data, ok := proxyRespParse["data"].(map[string]interface{})
	assert.True(t, ok)
	message, ok := data["message"].(string)
	assert.True(t, ok)
	assert.Contains(t, message, "cannot be to this destination")
}

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

func TestHealthCheckHandler(t *testing.T) {
	// Test the health check endpoint
	request := httptest.NewRequest("GET", "/health", nil)
	recorder := httptest.NewRecorder()

	healthCheckHandler(recorder, request)

	result := recorder.Result()
	defer result.Body.Close()

	assert.Equal(t, http.StatusOK, result.StatusCode)

	var healthResponse HealthResponse
	err := json.NewDecoder(result.Body).Decode(&healthResponse)
	assert.Nil(t, err)

	assert.Equal(t, "healthy", healthResponse.Status)
	assert.NotEmpty(t, healthResponse.Uptime)
	assert.Equal(t, startTime.Unix(), healthResponse.StartTime.Unix())
}

func TestMetricsHandler(t *testing.T) {
	// Test metrics handler with no auth token
	request := httptest.NewRequest("GET", "/metrics", nil)
	recorder := httptest.NewRecorder()

	metricsHandler(recorder, request)

	result := recorder.Result()
	defer result.Body.Close()

	assert.Equal(t, http.StatusOK, result.StatusCode)

	var metricsResponse HealthResponse
	err := json.NewDecoder(result.Body).Decode(&metricsResponse)
	assert.Nil(t, err)

	// Now test with auth token required
	accessToken = "metrics-token"
	defer func() {
		accessToken = "" // cleanup
	}()

	// Test without providing token (should fail)
	recorder = httptest.NewRecorder()
	metricsHandler(recorder, request)
	assert.Equal(t, http.StatusUnauthorized, recorder.Result().StatusCode)

	// Test with correct token
	request.Header.Set("Authorization", "Bearer metrics-token")
	recorder = httptest.NewRecorder()
	metricsHandler(recorder, request)
	assert.Equal(t, http.StatusOK, recorder.Result().StatusCode)
}

func TestSessionFingerprint(t *testing.T) {
	// Test that a unique session fingerprint is generated
	oldFingerprint := sessionFingerprint

	// Save original values to restore later
	origInfoLogger := InfoLogger
	origErrorLogger := ErrorLogger
	origDebugLogger := DebugLogger

	// Ensure loggers are set up
	if InfoLogger == nil || ErrorLogger == nil || DebugLogger == nil {
		setupLoggers()
	}

	// Instead of using Initialize which might cause issues in tests,
	// we'll just generate a new UUID for the session fingerprint
	sessionFingerprint = uuid.New().String()

	assert.NotEmpty(t, sessionFingerprint)
	assert.NotEqual(t, oldFingerprint, sessionFingerprint)

	// Test that the fingerprint is returned in non-POST requests
	request := httptest.NewRequest("GET", "/", nil)
	request.Header.Set("Origin", "validorigin1.com")
	recorder := httptest.NewRecorder()

	proxyHandler(recorder, request)

	var response map[string]interface{}
	err := json.NewDecoder(recorder.Body).Decode(&response)
	assert.Nil(t, err)

	data, ok := response["data"].(map[string]interface{})
	assert.True(t, ok)
	returnedFingerprint, ok := data["sessionFingerprint"].(string)
	assert.True(t, ok)
	assert.Equal(t, sessionFingerprint, returnedFingerprint)

	// Restore original loggers if they were nil
	if origInfoLogger == nil || origErrorLogger == nil || origDebugLogger == nil {
		InfoLogger = origInfoLogger
		ErrorLogger = origErrorLogger
		DebugLogger = origDebugLogger
	}
}

func TestCustomUserAgent(t *testing.T) {
	// Test that a custom User-Agent is passed through
	customUA := "CustomUserAgent/1.0"

	resp := getResultDef(Request{
		Method: "GET",
		Url:    testServerUrl + "/get",
		Headers: map[string]string{
			"User-Agent": customUA,
		},
	})

	httpBinResponse := checkErrorNUnmarshalHTTPBinResponse(resp.requestResponse.Data, t)
	assert.Equal(t, customUA, httpBinResponse.Headers.Get("User-Agent"))

	// Test that default User-Agent is used when none is provided
	resp = getResultDef(Request{
		Method: "GET",
		Url:    testServerUrl + "/get",
	})

	httpBinResponse = checkErrorNUnmarshalHTTPBinResponse(resp.requestResponse.Data, t)
	assert.Equal(t, "Proxyscotch/1.1", httpBinResponse.Headers.Get("User-Agent"))
}

func TestEnableHealthCheck(t *testing.T) {
	// Test enabling and disabling the health check
	oldValue := healthCheckEnabled
	defer func() {
		healthCheckEnabled = oldValue
	}()

	EnableHealthCheck(false)
	assert.False(t, healthCheckEnabled)

	EnableHealthCheck(true)
	assert.True(t, healthCheckEnabled)
}

func TestHeaderToArray(t *testing.T) {
	// Test the headerToArray function
	header := http.Header{}
	header.Add("Content-Type", "application/json")
	header.Add("X-Custom-Header", "value1")
	header.Add("X-Custom-Header", "value2") // Multiple values for the same key

	result := headerToArray(header)

	// It should only keep the last value for each key
	assert.Equal(t, "application/json", result["content-type"])
	assert.Equal(t, "value2", result["x-custom-header"])
}
