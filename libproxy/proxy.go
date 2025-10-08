package libproxy

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

// sanitizeLogInput removes potentially dangerous newline characters from user-controlled input
func sanitizeLogInput(input string) string {
	input = strings.ReplaceAll(input, "\n", "")
	input = strings.ReplaceAll(input, "\r", "")
	return input
}

// Initialize loggers
var (
	InfoLogger  *log.Logger
	ErrorLogger *log.Logger
	DebugLogger *log.Logger
)

// Health metrics
var (
	startTime          time.Time
	totalRequests      uint64
	totalErrors        uint64
	lastRequestTime    time.Time
	isServerRunning    bool
	healthCheckEnabled bool
)

type statusChangeFunction func(status string, isListening bool)

var (
	accessToken        string
	sessionFingerprint string
	allowedOrigins     []string
	bannedOutputs      []string
	bannedDests        []string
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

// HealthResponse contains the health check information
type HealthResponse struct {
	Status           string    `json:"status"`
	Uptime           string    `json:"uptime"`
	StartTime        time.Time `json:"startTime"`
	TotalRequests    uint64    `json:"totalRequests"`
	TotalErrors      uint64    `json:"totalErrors"`
	LastRequestTime  time.Time `json:"lastRequestTime,omitempty"`
	Version          string    `json:"version"`
	GoVersion        string    `json:"goVersion"`
	NumGoroutine     int       `json:"numGoroutine"`
	MemoryAllocated  uint64    `json:"memoryAllocated"`
	MemoryTotal      uint64    `json:"memoryTotal"`
	MemorySystemUsed uint64    `json:"memorySystemUsed"`
}

func setupLoggers() {
	// Create logs directory if it doesn't exist
	logDir := "logs"
	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		err := os.Mkdir(logDir, 0755)
		if err != nil {
			log.Println("Failed to create logs directory:", err)
		}
	}

	// Create log file with date in filename
	currentTime := time.Now().Format("2006-01-02")
	logFile, err := os.OpenFile(fmt.Sprintf("%s/proxy-%s.log", logDir, currentTime), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Println("Failed to open log file:", err)
	}

	// Create multi-writer to write logs to both file and stdout
	multiWriter := io.MultiWriter(os.Stdout, logFile)

	// Initialize loggers with different prefixes
	InfoLogger = log.New(multiWriter, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	ErrorLogger = log.New(multiWriter, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
	DebugLogger = log.New(multiWriter, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)
}

func isAllowedDest(dest string) bool {
	for _, b := range bannedDests {
		if b == dest {
			return false
		}
	}
	return true
}

// matchWildcard checks if a string matches a wildcard pattern
// Supports patterns like "*.hoppscotch.io"
func matchWildcard(pattern, str string) bool {
	// If pattern doesn't contain wildcard, do exact match
	if !strings.Contains(pattern, "*") {
		return pattern == str
	}

	// Handle wildcard patterns
	if strings.HasPrefix(pattern, "*.") {
		// Pattern like "*.hoppscotch.io"
		domain := strings.TrimPrefix(pattern, "*.")
		// Check if str ends with the domain
		if strings.HasSuffix(str, "."+domain) {
			return true
		}
		// Also match the exact domain (e.g., "hoppscotch.io" matches "*.hoppscotch.io")
		if str == domain {
			return true
		}
	} else if strings.HasSuffix(pattern, ".*") {
		// Pattern like "https://hoppscotch.*"
		prefix := strings.TrimSuffix(pattern, ".*")
		if strings.HasPrefix(str, prefix) {
			return true
		}
	} else if pattern == "*" {
		// Match everything
		return true
	}

	return false
}

func isAllowedOrigin(origin string) bool {
	// If first entry is wildcard "*", allow all
	if len(allowedOrigins) > 0 && allowedOrigins[0] == "*" {
		return true
	}

	// Check each allowed origin pattern
	for _, allowedPattern := range allowedOrigins {
		// Try exact match first
		if allowedPattern == origin {
			return true
		}
		// Try wildcard match
		if matchWildcard(allowedPattern, origin) {
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
	initialBannedDests string,
	onStatusChange statusChangeFunction,
	withSSL bool,
	finished chan bool,
) {
	// Set up loggers first
	setupLoggers()

	// Initialize health metrics
	startTime = time.Now()
	totalRequests = 0
	totalErrors = 0
	healthCheckEnabled = true

	if initialBannedOutputs != "" {
		bannedOutputs = strings.Split(initialBannedOutputs, ",")
	}

	if initialBannedDests != "" {
		bannedDests = strings.Split(initialBannedDests, ",")
	} else {
		bannedDests = []string{}
	}

	// Read allowed origins from environment variable
	envOrigins := os.Getenv("ALLOWED_ORIGINS")

	// If environment variable is set, use it; otherwise use the parameter or default
	if envOrigins != "" {
		allowedOrigins = strings.Split(envOrigins, ",")
		// Trim whitespace from each origin
		for i := range allowedOrigins {
			allowedOrigins[i] = strings.TrimSpace(allowedOrigins[i])
		}
		InfoLogger.Printf("Using allowed origins from ALLOWED_ORIGINS env: %v", allowedOrigins)
	} else if initialAllowedOrigins != "" {
		allowedOrigins = strings.Split(initialAllowedOrigins, ",")
		// Trim whitespace from each origin
		for i := range allowedOrigins {
			allowedOrigins[i] = strings.TrimSpace(allowedOrigins[i])
		}
		InfoLogger.Printf("Using allowed origins from parameter: %v", allowedOrigins)
	} else {
		// Default to hoppscotch.io if nothing is specified
		allowedOrigins = []string{"https://hoppscotch.io"}
		InfoLogger.Println("Using default allowed origin: https://hoppscotch.io")
	}

	accessToken = initialAccessToken
	sessionFingerprint = uuid.New().String()

	InfoLogger.Println("Starting proxy server...")
	InfoLogger.Printf("Allowed origins: %v", allowedOrigins)

	// Register handlers
	http.HandleFunc("/", proxyHandler)
	http.HandleFunc("/health", healthCheckHandler)
	http.HandleFunc("/metrics", metricsHandler)

	if !withSSL {
		go func() {
			InfoLogger.Printf("Attempting to listen on http://%s/", proxyURL)
			isServerRunning = true
			httpServerError := http.ListenAndServe(proxyURL, nil)
			if httpServerError != nil {
				isServerRunning = false
				errorMsg := fmt.Sprintf("Server failed to start: %v", httpServerError)
				ErrorLogger.Println(errorMsg)
				onStatusChange("An error occurred: "+httpServerError.Error(), false)
			}
			finished <- true
		}()
		onStatusChange("Listening on http://"+proxyURL+"/", true)
		InfoLogger.Printf("Proxy server listening on http://%s/", proxyURL)
		InfoLogger.Printf("Health check available at http://%s/health", proxyURL)
	} else {
		onStatusChange("Checking SSL certificate...", false)
		InfoLogger.Println("Checking SSL certificate...")

		err := EnsurePrivateKeyInstalled()
		if err != nil {
			ErrorLogger.Printf("SSL certificate error: %v", err)
			onStatusChange("An SSL certificate error occurred: "+err.Error(), false)
		}

		go func() {
			InfoLogger.Printf("Attempting to listen on https://%s/", proxyURL)
			isServerRunning = true
			httpServerError := http.ListenAndServeTLS(proxyURL, GetOrCreateDataPath()+"/cert.pem", GetOrCreateDataPath()+"/key.pem", nil)
			if httpServerError != nil {
				isServerRunning = false
				errorMsg := fmt.Sprintf("HTTPS server failed to start: %v", httpServerError)
				ErrorLogger.Println(errorMsg)
				onStatusChange("An error occurred: "+httpServerError.Error(), false)
			}
		}()
		onStatusChange("Listening on https://"+proxyURL+"/", true)
		InfoLogger.Printf("Proxy server listening on https://%s/", proxyURL)
		InfoLogger.Printf("Health check available at https://%s/health", proxyURL)
	}
}

func GetAccessToken() string {
	return accessToken
}

func SetAccessToken(newAccessToken string) {
	accessToken = newAccessToken
	InfoLogger.Println("Access token updated")
}

// GetHealthStatus returns the current health status of the proxy
func GetHealthStatus() HealthResponse {
	var memory runtime.MemStats
	runtime.ReadMemStats(&memory)

	return HealthResponse{
		Status:           getStatusString(),
		Uptime:           time.Since(startTime).String(),
		StartTime:        startTime,
		TotalRequests:    atomic.LoadUint64(&totalRequests),
		TotalErrors:      atomic.LoadUint64(&totalErrors),
		LastRequestTime:  lastRequestTime,
		Version:          "1.1.0",
		GoVersion:        runtime.Version(),
		NumGoroutine:     runtime.NumGoroutine(),
		MemoryAllocated:  memory.Alloc,
		MemoryTotal:      memory.TotalAlloc,
		MemorySystemUsed: memory.Sys,
	}
}

func getStatusString() string {
	if isServerRunning {
		return "healthy"
	}
	return "unhealthy"
}

// healthCheckHandler provides a simple health check endpoint
func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	healthStatus := GetHealthStatus()

	if healthStatus.Status == "healthy" {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	err := json.NewEncoder(w).Encode(healthStatus)
	if err != nil {
		ErrorLogger.Printf("Failed to encode health check response: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprintln(w, "{\"status\":\"error\",\"message\":\"Failed to generate health response\"}")
	}

	InfoLogger.Printf("Health check from %s returned status: %s", r.RemoteAddr, healthStatus.Status)
}

// metricsHandler provides detailed metrics for monitoring
func metricsHandler(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")

	if len(accessToken) > 0 && (authHeader != "Bearer "+accessToken) {
		ErrorLogger.Printf("Unauthorized metrics access from %s", r.RemoteAddr)
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = fmt.Fprintln(w, "{\"status\":\"error\",\"message\":\"Unauthorized access\"}")
		return
	}

	healthStatus := GetHealthStatus()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err := json.NewEncoder(w).Encode(healthStatus)
	if err != nil {
		ErrorLogger.Printf("Failed to encode metrics response: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprintln(w, "{\"status\":\"error\",\"message\":\"Failed to generate metrics response\"}")
	}

	InfoLogger.Printf("Metrics requested from %s", r.RemoteAddr)
}

const ErrorBodyInvalidRequest = "{\"success\": false, \"data\":{\"message\":\"(Proxy Error) Invalid request.\"}}"
const ErrorBodyProxyRequestFailed = "{\"success\": false, \"data\":{\"message\":\"(Proxy Error) Request failed.\"}}"
const maxMemory = int64(32 << 20)

func proxyHandler(response http.ResponseWriter, request *http.Request) {
	atomic.AddUint64(&totalRequests, 1)
	lastRequestTime = time.Now()
	startTime := time.Now()
	clientIP := request.RemoteAddr
	method := request.Method
	requestURL := request.URL.String()
	userAgent := request.Header.Get("User-Agent")

	// Sanitize user-controlled input before logging
	safeRequestURL := sanitizeLogInput(requestURL)
	safeUserAgent := sanitizeLogInput(userAgent)

	InfoLogger.Printf("Received %s request from %s for %s (User-Agent: %s)", method, clientIP, safeRequestURL, safeUserAgent)

	if request.URL.Path == "/health" || request.URL.Path == "/metrics" {
		return
	}

	response.Header().Add("Access-Control-Allow-Headers", "*")

	if request.Method == "OPTIONS" {
		response.Header().Add("Access-Control-Allow-Origin", "*")
		response.WriteHeader(200)
		DebugLogger.Printf("Responded to OPTIONS request from %s in %v", clientIP, time.Since(startTime))
		return
	}

	origin := request.Header.Get("Origin")
	if origin == "" || !isAllowedOrigin(origin) {
		atomic.AddUint64(&totalErrors, 1)
		if strings.HasPrefix(request.Header.Get("Content-Type"), "application/json") {
			response.Header().Add("Access-Control-Allow-Headers", "*")
			response.Header().Add("Access-Control-Allow-Origin", "*")
			response.WriteHeader(200)
			_, err := fmt.Fprintln(response, ErrorBodyProxyRequestFailed)
			if err != nil {
				ErrorLogger.Printf("Failed to write error response: %v", err)
			}
			ErrorLogger.Printf("Denied access to %s from disallowed origin: %s", clientIP, sanitizeLogInput(origin))
			return
		}

		response.Header().Add("Location", "https://hoppscotch.io/")
		response.WriteHeader(301)
		InfoLogger.Printf("Redirected request from %s with disallowed origin: %s", clientIP, sanitizeLogInput(origin))
		return
	} else {
		response.Header().Add("Access-Control-Allow-Origin", origin)
		DebugLogger.Printf("Allowed request from origin: %s", sanitizeLogInput(origin))
	}

	response.Header().Add("Content-Type", "application/json; charset=utf-8")

	if request.Method != "POST" {
		_, err := fmt.Fprintln(response, "{\"success\": true, \"data\":{\"sessionFingerprint\":\""+sessionFingerprint+"\", \"isProtected\":"+strconv.FormatBool(len(accessToken) > 0)+"}}")
		if err != nil {
			atomic.AddUint64(&totalErrors, 1)
			ErrorLogger.Printf("Failed to write non-POST response: %v", err)
		}
		InfoLogger.Printf("Responded to %s request from %s in %v", method, clientIP, time.Since(startTime))
		return
	}

	var requestData Request
	isMultipart := strings.HasPrefix(request.Header.Get("content-type"), "multipart/form-data")
	var multipartRequestDataKey = request.Header.Get("multipart-part-key")
	if multipartRequestDataKey == "" {
		multipartRequestDataKey = "proxyRequestData"
	}

	if isMultipart {
		DebugLogger.Printf("Processing multipart form data request from %s", clientIP)
		err := request.ParseMultipartForm(maxMemory)
		if err != nil {
			atomic.AddUint64(&totalErrors, 1)
			ErrorLogger.Printf("Failed to parse multipart form from %s: %v", clientIP, err)
			_, writeErr := fmt.Fprintln(response, ErrorBodyInvalidRequest)
			if writeErr != nil {
				ErrorLogger.Printf("Failed to write error response: %v", writeErr)
			}
			return
		}

		if request.MultipartForm == nil || request.MultipartForm.Value == nil {
			atomic.AddUint64(&totalErrors, 1)
			ErrorLogger.Printf("Invalid multipart form from %s: MultipartForm or Value is nil", clientIP)
			_, writeErr := fmt.Fprintln(response, ErrorBodyInvalidRequest)
			if writeErr != nil {
				ErrorLogger.Printf("Failed to write error response: %v", writeErr)
			}
			return
		}

		r, exists := request.MultipartForm.Value[multipartRequestDataKey]
		if !exists || len(r) == 0 {
			atomic.AddUint64(&totalErrors, 1)
			ErrorLogger.Printf("Invalid multipart form from %s: missing %s key", clientIP, sanitizeLogInput(multipartRequestDataKey))
			_, writeErr := fmt.Fprintln(response, ErrorBodyInvalidRequest)
			if writeErr != nil {
				ErrorLogger.Printf("Failed to write error response: %v", writeErr)
			}
			return
		}

		err = json.Unmarshal([]byte(r[0]), &requestData)
		if err != nil || len(requestData.Url) == 0 || len(requestData.Method) == 0 {
			atomic.AddUint64(&totalErrors, 1)
			ErrorLogger.Printf("Failed to parse request body from %s: %v", clientIP, err)
			_, writeErr := fmt.Fprintln(response, ErrorBodyInvalidRequest)
			if writeErr != nil {
				ErrorLogger.Printf("Failed to write error response: %v", writeErr)
			}
			return
		}
	} else {
		DebugLogger.Printf("Processing JSON request from %s", clientIP)
		err := json.NewDecoder(request.Body).Decode(&requestData)
		if err != nil || len(requestData.Url) == 0 || len(requestData.Method) == 0 {
			atomic.AddUint64(&totalErrors, 1)
			ErrorLogger.Printf("Failed to parse JSON request body from %s: %v", clientIP, err)
			_, writeErr := fmt.Fprintln(response, ErrorBodyInvalidRequest)
			if writeErr != nil {
				ErrorLogger.Printf("Failed to write error response: %v", writeErr)
			}
			return
		}
	}

	// Log the proxied request details with sanitized input
	InfoLogger.Printf("Proxying %s request from %s to %s", sanitizeLogInput(requestData.Method), clientIP, sanitizeLogInput(requestData.Url))

	if len(accessToken) > 0 && requestData.AccessToken != accessToken {
		atomic.AddUint64(&totalErrors, 1)
		ErrorLogger.Printf("Unauthorized request from %s: Invalid access token", clientIP)
		_, err := fmt.Fprintln(response, "{\"success\": false, \"data\":{\"message\":\"(Proxy Error) Unauthorized request; you may need to set your access token in Settings.\"}}")
		if err != nil {
			ErrorLogger.Printf("Failed to write unauthorized error: %v", err)
		}
		return
	}

	var proxyRequest http.Request
	proxyRequest.Header = make(http.Header)
	proxyRequest.Method = requestData.Method

	parsedURL, err := url.Parse(requestData.Url)
	if err != nil {
		atomic.AddUint64(&totalErrors, 1)
		ErrorLogger.Printf("Invalid URL from %s: %v", clientIP, err)
		_, writeErr := fmt.Fprintln(response, "{\"success\": false, \"data\":{\"message\":\"(Proxy Error) Invalid URL: "+err.Error()+"\"}}")
		if writeErr != nil {
			ErrorLogger.Printf("Failed to write error response: %v", writeErr)
		}
		return
	}
	proxyRequest.URL = parsedURL

	if !isAllowedDest(proxyRequest.URL.Hostname()) {
		atomic.AddUint64(&totalErrors, 1)
		ErrorLogger.Printf("Request to banned destination %s from %s", proxyRequest.URL.Hostname(), clientIP)
		_, err := fmt.Fprintln(response, "{\"success\": false, \"data\":{\"message\":\"(Proxy Error) Request cannot be to this destination.\"}}")
		if err != nil {
			ErrorLogger.Printf("Failed to write banned destination error: %v", err)
		}
		return
	}

	var params = proxyRequest.URL.Query()
	for k, v := range requestData.Params {
		params.Set(k, v)
	}
	proxyRequest.URL.RawQuery = params.Encode()

	if len(requestData.Auth.Username) > 0 && len(requestData.Auth.Password) > 0 {
		proxyRequest.SetBasicAuth(requestData.Auth.Username, requestData.Auth.Password)
		DebugLogger.Printf("Using basic auth for request to %s", sanitizeLogInput(proxyRequest.URL.String()))
	}

	for k, v := range requestData.Headers {
		proxyRequest.Header.Set(k, v)
	}

	proxyRequest.Header.Set("X-Forwarded-For", request.RemoteAddr)
	proxyRequest.Header.Set("Via", "Proxyscotch/1.1")

	if len(strings.TrimSpace(proxyRequest.Header.Get("User-Agent"))) < 1 {
		proxyRequest.Header.Set("User-Agent", "Proxyscotch/1.1")
	}

	if isMultipart {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)

		for key := range request.MultipartForm.Value {
			if key == multipartRequestDataKey {
				continue
			}
			for _, val := range request.MultipartForm.Value[key] {
				err := writer.WriteField(key, val)
				if err != nil {
					atomic.AddUint64(&totalErrors, 1)
					ErrorLogger.Printf("Failed to write multipart field key: %s error: %v", key, err)
					_, writeErr := fmt.Fprintln(response, ErrorBodyProxyRequestFailed)
					if writeErr != nil {
						ErrorLogger.Printf("Failed to write error response: %v", writeErr)
					}
					return
				}
			}
		}

		for fileKey := range request.MultipartForm.File {
			for _, val := range request.MultipartForm.File[fileKey] {
				f, err := val.Open()
				if err != nil {
					ErrorLogger.Printf("Failed to open file %s: %v", sanitizeLogInput(val.Filename), err)
					continue
				}

				field, err := writer.CreatePart(val.Header)
				if err != nil {
					ErrorLogger.Printf("Failed to create part for file %s: %v", sanitizeLogInput(val.Filename), err)
					err = f.Close()
					if err != nil {
						ErrorLogger.Printf("Failed to close file: %v", err)
					}
					continue
				}

				_, err = io.Copy(field, f)
				if err != nil {
					ErrorLogger.Printf("Failed to copy file %s: %v", sanitizeLogInput(val.Filename), err)
				}

				err = f.Close()
				if err != nil {
					ErrorLogger.Printf("Failed to close file %s: %v", sanitizeLogInput(val.Filename), err)
				}
			}
		}

		err := writer.Close()
		if err != nil {
			atomic.AddUint64(&totalErrors, 1)
			ErrorLogger.Printf("Failed to finalize multipart content: %v", err)
			_, writeErr := fmt.Fprintln(response, ErrorBodyProxyRequestFailed)
			if writeErr != nil {
				ErrorLogger.Printf("Failed to write error response: %v", writeErr)
			}
			return
		}

		contentType := fmt.Sprintf("multipart/form-data; boundary=%v", writer.Boundary())
		proxyRequest.Header.Set("content-type", contentType)
		proxyRequest.Body = io.NopCloser(bytes.NewReader(body.Bytes()))
		proxyRequest.ContentLength = int64(len(body.Bytes()))
	} else if len(requestData.Data) > 0 {
		proxyRequest.Body = io.NopCloser(strings.NewReader(requestData.Data))
		proxyRequest.ContentLength = int64(len(requestData.Data))
	}

	var client = &http.Client{
		Timeout: 30 * time.Second,
	}

	DebugLogger.Printf("Sending proxied request to %s", sanitizeLogInput(proxyRequest.URL.String()))
	proxyStartTime := time.Now()

	proxyResponse, err := client.Do(&proxyRequest)
	if err != nil {
		atomic.AddUint64(&totalErrors, 1)
		ErrorLogger.Printf("Failed to execute proxied request to %s: %v", sanitizeLogInput(proxyRequest.URL.String()), err)
		_, writeErr := fmt.Fprintln(response, "{\"success\": false, \"data\":{\"message\":\"(Proxy Error) Request failed: "+err.Error()+"\"}}")
		if writeErr != nil {
			ErrorLogger.Printf("Failed to write error response: %v", writeErr)
		}
		return
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			ErrorLogger.Printf("Failed to close response body: %v", err)
		}
	}(proxyResponse.Body)

	InfoLogger.Printf("Received response from %s with status %d in %v", sanitizeLogInput(proxyRequest.URL.String()), proxyResponse.StatusCode, time.Since(proxyStartTime))

	var responseData Response
	responseData.Success = true
	responseData.Status = proxyResponse.StatusCode
	responseData.StatusText = strings.Join(strings.Split(proxyResponse.Status, " ")[1:], " ")

	responseBytes, err := io.ReadAll(proxyResponse.Body)
	if err != nil {
		atomic.AddUint64(&totalErrors, 1)
		ErrorLogger.Printf("Failed to read response body: %v", err)
		_, writeErr := fmt.Fprintln(response, ErrorBodyProxyRequestFailed)
		if writeErr != nil {
			ErrorLogger.Printf("Failed to write error response: %v", writeErr)
		}
		return
	}

	responseData.Headers = headerToArray(proxyResponse.Header)

	if requestData.WantsBinary {
		for _, bannedOutput := range bannedOutputs {
			responseBytes = bytes.ReplaceAll(responseBytes, []byte(bannedOutput), []byte("[redacted]"))
		}
		responseData.Data = base64.RawStdEncoding.EncodeToString(responseBytes)
		responseData.IsBinary = true
		DebugLogger.Printf("Returning binary response of %d bytes", len(responseBytes))
	} else {
		responseData.Data = string(responseBytes)
		for _, bannedOutput := range bannedOutputs {
			responseData.Data = strings.Replace(responseData.Data, bannedOutput, "[redacted]", -1)
		}
		DebugLogger.Printf("Returning text response of %d bytes", len(responseData.Data))
	}

	err = json.NewEncoder(response).Encode(responseData)
	if err != nil {
		atomic.AddUint64(&totalErrors, 1)
		ErrorLogger.Printf("Failed to encode response: %v", err)
		_, writeErr := fmt.Fprintln(response, ErrorBodyProxyRequestFailed)
		if writeErr != nil {
			ErrorLogger.Printf("Failed to write error response: %v", writeErr)
		}
		return
	}

	// Log completion with sanitized user input
	InfoLogger.Printf("Completed %s request from %s to %s in %v", sanitizeLogInput(requestData.Method), clientIP, sanitizeLogInput(requestData.Url), time.Since(startTime))
}

// EnableHealthCheck turns on or off the health check functionality
func EnableHealthCheck(enable bool) {
	healthCheckEnabled = enable
	InfoLogger.Printf("Health check endpoint %s", map[bool]string{true: "enabled", false: "disabled"}[enable])
}

func headerToArray(header http.Header) (res map[string]string) {
	res = make(map[string]string)
	for name, values := range header {
		for _, value := range values {
			res[strings.ToLower(name)] = value
		}
	}
	return res
}
