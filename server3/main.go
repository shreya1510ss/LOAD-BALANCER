package main

import (
	"net/http"
	"io"
	"log"
	"os"
	"fmt"
)



func main(){
    
	 homeHandler := func(w http.ResponseWriter, r *http.Request) {
		l:= log.New(os.Stdout, "[server3]", log.Ldate|log.Ltime)
		l.Printf("running...")
		io.WriteString(w, "Hello, World coming from server 3!")
	}
	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	fmt.Println("Server 3 is running on port 8083...")
	log.Fatal(http.ListenAndServe(":8083", nil))
}