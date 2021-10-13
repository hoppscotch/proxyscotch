package libproxy

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
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
	Params  map[string]string
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

const ErrorBodyInvalidRequest = "{\"success\": false, \"data\":{\"message\":\"(Proxy Error) Invalid request.\"}}"
const ErrorBodyProxyRequestFailed = "{\"success\": false, \"data\":{\"message\":\"(Proxy Error) Request failed.\"}}"
const maxMemory = int64(32 << 20) // multipartRequestDataKey currently its 32 MB

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
			_, _ = fmt.Fprintln(response, ErrorBodyProxyRequestFailed)
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
	isMultipart := strings.HasPrefix(request.Header.Get("content-type"), "multipart/form-data")
	var multipartRequestDataKey = request.Header.Get("multipart-part-key")
	if multipartRequestDataKey == "" {
		multipartRequestDataKey = "proxyRequestData"
	}
	if isMultipart {
		var err = request.ParseMultipartForm(maxMemory)
		if err != nil {
			log.Printf("Failed to parse request body: %v", err)
			_, _ = fmt.Fprintln(response, ErrorBodyInvalidRequest)
			return
		}
		r := request.MultipartForm.Value[multipartRequestDataKey]
		err = json.Unmarshal([]byte(r[0]), &requestData)
		if err != nil || len(requestData.Url) == 0 || len(requestData.Method) == 0 {
			// If the logged err is nil here, it means either the URL or method were not supplied
			// in the request data.
			log.Printf("Failed to parse request body: %v", err)
			_, _ = fmt.Fprintln(response, ErrorBodyInvalidRequest)
			return
		}
	} else {
		var err = json.NewDecoder(request.Body).Decode(&requestData)
		if err != nil || len(requestData.Url) == 0 || len(requestData.Method) == 0 {
			// If the logged err is nil here, it means either the URL or method were not supplied
			// in the request data.
			log.Printf("Failed to parse request body: %v", err)
			_, _ = fmt.Fprintln(response, ErrorBodyInvalidRequest)
			return
		}
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
	proxyRequest.URL, _ = url.Parse(requestData.Url)

	var params = proxyRequest.URL.Query()

	for k, v := range requestData.Params {
		params.Set(k, v)
	}
	proxyRequest.URL.RawQuery = params.Encode()

	if len(requestData.Auth.Username) > 0 && len(requestData.Auth.Password) > 0 {
		proxyRequest.SetBasicAuth(requestData.Auth.Username, requestData.Auth.Password)
	}
	for k, v := range requestData.Headers {
		proxyRequest.Header.Set(k, v)
	}

	proxyRequest.Header.Set("User-Agent", "Proxyscotch/1.1")

	if isMultipart {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		for key := range request.MultipartForm.Value {
			if key == multipartRequestDataKey {
				continue
			}
			for _, val := range request.MultipartForm.Value[key] {
				// This usually never happens, mostly memory issue
				err := writer.WriteField(key, val)
				if err != nil {
					log.Printf("Failed to write multipart field key: %s error: %v", key, err)
					return
				}
			}
		}
		for fileKey := range request.MultipartForm.File {
			for _, val := range request.MultipartForm.File[fileKey] {
				f, err := val.Open()
				if err != nil {
					log.Printf("Failed to write multipart field: %s err: %v", fileKey, err)
					continue
				}
				field, _ := writer.CreatePart(val.Header)
				_, err = io.Copy(field, f)
				if err != nil {
					log.Printf("Failed to write multipart field: %s err: %v", fileKey, err)
				}
				// Close need not be handled, as go will clear temp file
				defer func(f multipart.File) {
					err := f.Close()
					if err != nil {
						log.Printf("Failed to close file")
					}
				}(f)
			}
		}
		err := writer.Close()
		if err != nil {
			log.Printf("Failed to write multipart content: %v", err)
			_, _ = fmt.Fprintf(response, ErrorBodyProxyRequestFailed)
			if err != nil {
				return
			}
			return
		}
		contentType := fmt.Sprintf("multipart/form-data; boundary=%v", writer.Boundary())
		proxyRequest.Header.Set("content-type", contentType)
		proxyRequest.Body = ioutil.NopCloser(bytes.NewReader(body.Bytes()))
		proxyRequest.Body.Close()
	} else if len(requestData.Data) > 0 {
		proxyRequest.Body = ioutil.NopCloser(strings.NewReader(requestData.Data))
		proxyRequest.Body.Close()
	}

	var client http.Client
	var proxyResponse *http.Response
	proxyResponse, err := client.Do(&proxyRequest)

	if err != nil {
		log.Print("Failed to write response body: ", err.Error())
		_, _ = fmt.Fprintln(response, ErrorBodyProxyRequestFailed)
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
		_, _ = fmt.Fprintln(response, ErrorBodyProxyRequestFailed)
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
