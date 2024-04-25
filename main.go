package main

import (
	_ "embed"
	"log"
	"net/http"
)

//go:embed index.html
var assetIndex []byte

func main() {
	log.Println("starting on port", "8080")
	http.ListenAndServe(":8080", http.HandlerFunc(handler))
}

func handler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.Write(assetIndex)
}
