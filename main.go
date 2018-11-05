package main

import (
	"fmt"
	"google.golang.org/appengine"
	"net/http"
)

func main() {
	http.HandleFunc("/", indexHandler)

	appengine.Main()
}

// Index
func indexHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "")
}
