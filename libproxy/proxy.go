package libproxy

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

type statusChangeFunction func(status string, isListening bool)

var (
	accessToken        string
	sessionFingerprint string
	allowedOrigins     []string
	bannedOutputs      []string
)

type Request struct {
	AccessToken string
	WantsBinary bool
	Method      string
	Url         string
	Auth        struct {
		Username string
		Password string
	}
	Headers map[string]string
	Data    string
}

type Response struct {
	Success    bool              `json:"success"`
	IsBinary   bool              `json:"isBinary"`
	Status     int               `json:"status"`
	Data       string            `json:"data"`
	StatusText string            `json:"statusText"`
	Headers    map[string]string `json:"headers"`
}

func isAllowedOrigin(origin string) bool {
	if allowedOrigins[0] == "*" {
		return true
	}

	for _, b := range allowedOrigins {
		if b == origin {
			return true
		}
	}

	return false
}

func Initialize(
	initialAccessToken string,
	proxyURL string,
	initialAllowedOrigins string,
	initialBannedOutputs string,
	onStatusChange statusChangeFunction,
	withSSL bool,
	finished chan bool,
) {
	if initialBannedOutputs != "" {
		bannedOutputs = strings.Split(initialBannedOutputs, ",")
	}
	allowedOrigins = strings.Split(initialAllowedOrigins, ",")
	accessToken = initialAccessToken
	sessionFingerprint = uuid.New().String()
	log.Println("Starting proxy server...")

	http.HandleFunc("/", proxyHandler)

	if !withSSL {
		go func() {
			httpServerError := http.ListenAndServe(proxyURL, nil)

			if httpServerError != nil {
				onStatusChange("An error occurred: "+httpServerError.Error(), false)
			}

			finished <- true
		}()

		onStatusChange("Listening on http://"+proxyURL+"/", true)
	} else {
		onStatusChange("Checking SSL certificate...", false)

		err := EnsurePrivateKeyInstalled()
		if err != nil {
			log.Println(err.Error())
			onStatusChange("An error occurred.", false)
		}

		go func() {
			httpServerError := http.ListenAndServeTLS(proxyURL, GetOrCreateDataPath()+"/cert.pem", GetOrCreateDataPath()+"/key.pem", nil)

			if httpServerError != nil {
				onStatusChange("An error occurred.", false)
			}
		}()

		onStatusChange("Listening on https://"+proxyURL+"/", true)
		log.Println("Proxy server listening on https://" + proxyURL + "/")
	}
}

func GetAccessToken() string {
	return accessToken
}

func SetAccessToken(newAccessToken string) {
	accessToken = newAccessToken
}

func proxyHandler(response http.ResponseWriter, request *http.Request) {
	// We want to allow all types of requests to the proxy, though we only want to allow certain
	// origins.
	response.Header().Add("Access-Control-Allow-Headers", "*")
	if request.Method == "OPTIONS" {
		response.Header().Add("Access-Control-Allow-Origin", "*")
		response.WriteHeader(200)
		return
	}

	if request.Header.Get("Origin") == "" || !isAllowedOrigin(request.Header.Get("Origin")) {
		if strings.HasPrefix(request.Header.Get("Content-Type"), "application/json") {
			response.Header().Add("Access-Control-Allow-Headers", "*")
			response.Header().Add("Access-Control-Allow-Origin", "*")
			response.WriteHeader(200)
			_, _ = fmt.Fprintln(response, "{\"success\": false, \"data\":{\"message\":\"(Proxy Error) Request failed.\"}}")
			return
		}

		// If it is not an allowed origin, redirect back to hoppscotch.io.
		response.Header().Add("Location", "https://hoppscotch.io/")
		response.WriteHeader(301)
		return
	} else {
		// Otherwise set the appropriate CORS polciy and continue.
		response.Header().Add("Access-Control-Allow-Origin", request.Header.Get("Origin"))
	}

	// For anything other than an POST request, we'll return an empty JSON object.
	response.Header().Add("Content-Type", "application/json; charset=utf-8")
	if request.Method != "POST" {
		_, _ = fmt.Fprintln(response, "{\"success\": true, \"data\":{\"sessionFingerprint\":\""+sessionFingerprint+"\", \"isProtected\":"+strconv.FormatBool(len(accessToken) > 0)+"}}")
		return
	}

	// Attempt to parse request body.
	var requestData Request
	err := json.NewDecoder(request.Body).Decode(&requestData)
	if err != nil || len(requestData.Url) == 0 || len(requestData.Method) == 0 {
		// If the logged err is nil here, it means either the URL or method were not supplied
		// in the request data.
		log.Printf("Failed to parse request body: %v", err)
		_, _ = fmt.Fprintln(response, "{\"success\": false, \"data\":{\"message\":\"(Proxy Error) Invalid request.\"}}")
		return
	}

	if len(accessToken) > 0 && requestData.AccessToken != accessToken {
		log.Print("An unauthorized request was made.")
		_, _ = fmt.Fprintln(response, "{\"success\": false, \"data\":{\"message\":\"(Proxy Error) Unauthorized request; you may need to set your access token in Settings.\"}}")
		return
	}

	// Make the request
	var proxyRequest http.Request
	proxyRequest.Header = make(http.Header)
	proxyRequest.Method = requestData.Method
	proxyRequest.URL, err = url.Parse(requestData.Url)
	if len(requestData.Auth.Username) > 0 && len(requestData.Auth.Password) > 0 {
		proxyRequest.SetBasicAuth(requestData.Auth.Username, requestData.Auth.Password)
	}
	for k, v := range requestData.Headers {
		proxyRequest.Header.Set(k, v)
	}

	proxyRequest.Header.Set("User-Agent", "Proxyscotch/1.0")

	if len(requestData.Data) > 0 {
		proxyRequest.Body = ioutil.NopCloser(strings.NewReader(requestData.Data))
		proxyRequest.Body.Close()
	}

	var client http.Client
	var proxyResponse *http.Response
	proxyResponse, err = client.Do(&proxyRequest)

	if err != nil {
		log.Print("Failed to write response body: ", err.Error())
		_, _ = fmt.Fprintln(response, "{\"success\": false, \"data\":{\"message\":\"(Proxy Error) Request failed.\"}}")
		return
	}

	var responseData Response
	responseData.Success = true
	responseData.Status = proxyResponse.StatusCode
	responseData.StatusText = strings.Join(strings.Split(proxyResponse.Status, " ")[1:], " ")
	responseBytes, err := ioutil.ReadAll(proxyResponse.Body)
	responseData.Headers = headerToArray(proxyResponse.Header)

	if requestData.WantsBinary {
		for _, bannedOutput := range bannedOutputs {
			responseBytes = bytes.ReplaceAll(responseBytes, []byte(bannedOutput), []byte("[redacted]"))
		}

		// If using the new binary format, encode the response body.
		responseData.Data = base64.RawStdEncoding.EncodeToString(responseBytes)
		responseData.IsBinary = true
	} else {
		// Otherwise, simply return the old format.
		responseData.Data = string(responseBytes)

		for _, bannedOutput := range bannedOutputs {
			responseData.Data = strings.Replace(responseData.Data, bannedOutput, "[redacted]", -1)
		}
	}

	// Write the request body to the response.
	err = json.NewEncoder(response).Encode(responseData)

	// Return the response.
	if err != nil {
		log.Print("Failed to write response body: ", err.Error())
		_, _ = fmt.Fprintln(response, "{\"success\": false, \"data\":{\"message\":\"(Proxy Error) Request failed.\"}}")
		return
	}
}

/// Converts http.Header to a map.
/// Original Source: https://stackoverflow.com/a/37030039/2872279 (modified).
func headerToArray(header http.Header) (res map[string]string) {
	res = make(map[string]string)

	for name, values := range header {
		for _, value := range values {
			res[strings.ToLower(name)] = value
		}
	}

	return res
}
