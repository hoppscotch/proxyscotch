package libproxy

import (
    "encoding/json"
    "fmt"
    "io/ioutil"
    "log"
    "net/http"
    "net/url"
    "strings"
);

type statusChangeFunction func(status string, isListening bool);
var accessToken string;

type Request struct {
    AccessToken string;
    Method string;
    Url string;
    Auth struct {
        Username string;
        Password string;
    };
    Headers map[string]string;
    Data string;
}

type Response struct {
    Success bool                `json:"success"`;
    Data string                 `json:"data"`;
    Status int                  `json:"status"`;
    StatusText string           `json:"statusText"`;
    Headers map[string]string   `json:"headers"`;
}

func Initialize(initialAccessToken string, proxyURL string, onStatusChange statusChangeFunction, withSSL bool) {
    accessToken = initialAccessToken;
    fmt.Println("Starting proxy server...");

    http.HandleFunc("/", proxyHandler);

    if(!withSSL){
        go func() {
            httpServerError := http.ListenAndServe(proxyURL, nil);

            if httpServerError != nil {
                onStatusChange("An error occurred.", false);
            }
        }();

        onStatusChange("Listening on http://" + proxyURL + "/", true);
        fmt.Println("Proxy server listening on http://" + proxyURL + "/");
    }else{
        onStatusChange("Checking SSL certificate...", false);

        err := EnsurePrivateKeyInstalled();
        if err != nil {
            onStatusChange("An error occurred.", false);
        }

        go func() {
            httpServerError := http.ListenAndServeTLS(proxyURL, GetDataPath() + "/cert.pem", GetDataPath() + "/key.pem", nil);

            if httpServerError != nil {
                onStatusChange("An error occurred.", false);
            }
        }();

        onStatusChange("Listening on https://" + proxyURL + "/", true);
        fmt.Println("Proxy server listening on https://" + proxyURL + "/");
    }
}

func GetAccessToken() string {
    return accessToken;
}

func SetAccessToken(newAccessToken string) {
    accessToken = newAccessToken;
}

func proxyHandler(response http.ResponseWriter, request *http.Request) {
    // We want to allow all types of requests to the proxy, though we only want to allow certain
    // origins.
    response.Header().Add("Access-Control-Allow-Origin", "https://postwoman.io");
    response.Header().Add("Access-Control-Allow-Headers", "*");
    if request.Method == "OPTIONS" {
        response.WriteHeader(200);
        return;
    }

    // Then, for anything other than an POST request, we'll return an empty JSON object.
    response.Header().Add("Content-Type", "application/json; charset=utf-8");
    if request.Method != "POST" {
        _, _ = fmt.Fprintln(response, "{}");
        return;
    }

    // Attempt to parse request body.
    var requestData Request;
    err := json.NewDecoder(request.Body).Decode(&requestData);
    if (err != nil || len(requestData.Url) == 0 || len(requestData.Method) == 0) {
        // If the logged err is nil here, it means either the URL or method were not supplied
        // in the request data.
        log.Printf("Failed to parse request body: %v", err);
        _, _ = fmt.Fprintln(response, "{\"success\": false, \"data\":{\"message\":\"(Proxy Error) Invalid request.\"}}");
        return;
    }

    if(len(accessToken) > 0 && requestData.AccessToken != accessToken){
        log.Print("An unauthorized request was made.");
        _, _ = fmt.Fprintln(response, "{\"success\": false, \"data\":{\"message\":\"(Proxy Error) Unauthorized request; you may need to set your access token in Settings.\"}}");
        return;
    }

    // Make the request
    var proxyRequest http.Request;
    proxyRequest.Method = requestData.Method;
    proxyRequest.URL, err = url.Parse(requestData.Url);
    if(len(requestData.Auth.Username) > 0 && len(requestData.Auth.Password) > 0) {
        proxyRequest.SetBasicAuth(requestData.Auth.Username, requestData.Auth.Password);
    }
    for k, v := range requestData.Headers {
        proxyRequest.Header.Set(k, v);
    }

    if(len(requestData.Data) > 0) {
        var proxyRequestBody []byte;
        proxyRequestBody, err = ioutil.ReadAll(strings.NewReader(requestData.Data));
        _, err = proxyRequest.Body.Read(proxyRequestBody);
    }

    var client http.Client;
    var proxyResponse *http.Response;
    proxyResponse, err = client.Do(&proxyRequest);

    var responseData Response;
    responseData.Success = true;
    responseData.Status = proxyResponse.StatusCode;
    responseData.StatusText = strings.Join(strings.Split(proxyResponse.Status, " ")[1:], " ");
    responseBytes, err := ioutil.ReadAll(proxyResponse.Body);
    responseData.Data = string(responseBytes);
    responseData.Headers = headerToArray(proxyResponse.Header);

    // Write the request body to the response.
    err = json.NewEncoder(response).Encode(responseData);

    // Return the response.
    if err != nil {
        log.Printf("Failed to write response body: %v. %d bytes written.", err);
        _, _ = fmt.Fprintln(response, "{\"success\": false, \"data\":{\"message\":\"(Proxy Error) Request failed.\"}}");
        return;
    }
}

///
/// Converts http.Header to a map.
/// Original Source: https://stackoverflow.com/a/37030039/2872279 (modified)
///
func headerToArray(header http.Header) (res map[string]string) {
    res = make(map[string]string);

    for name, values := range header {
        for _, value := range values {
            res[name] = value;
        }
    }

    return res;
}