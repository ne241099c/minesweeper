package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	// staticフォルダの中身（html, js, wasm）をそのまま配信するだけ
	http.Handle("/", http.FileServer(http.Dir("static")))

	fmt.Println("File Server starting on :8080...")
	log.Fatal(http.ListenAndServe("0.0.0.0:8080", nil))
}
