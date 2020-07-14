package main

import (
	"flag"
	"log"
	"postwoman.io/proxy/libproxy"
)

func main(){
	hostPtr := flag.String("host", "localhost:9159", "the hostname that the server should listen on.");
	tokenPtr := flag.String("token", "", "the Proxy Access Token used to restrict access to the server.");
	allowedOriginsPtr := flag.String("allowed-origins", "https://postwoman.io", "a comma separated list of allowed origins.");
	flag.Parse();

	finished := make(chan bool);
	libproxy.Initialize(*tokenPtr, *hostPtr, *allowedOriginsPtr, onProxyStateChangeServer, false, finished);

	<- finished;
}

func onProxyStateChangeServer(status string, isListening bool){
	log.Printf("[ready=%v] %s", isListening, status);
}