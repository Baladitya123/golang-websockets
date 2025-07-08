package main

import (
	"context"
	"log"
	"net/http"
)

func main() {
	setUpApi()
	log.Fatal(http.ListenAndServeTLS(":8080", "server.crt", "server.key", nil))
}

func setUpApi() {
	ctx := context.Background()
	Manager := newManager(ctx)
	http.Handle("/", http.FileServer(http.Dir("./frontend")))
	http.HandleFunc("/ws", Manager.serveWS)
	http.HandleFunc("/login", Manager.loginHandler)
}
