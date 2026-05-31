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
		l:= log.New(os.Stdout, "[server1]", log.Ldate|log.Ltime)
		l.Printf("running...")
		io.WriteString(w, "Hello, World!")
	}
	http.HandleFunc("/", homeHandler)
	fmt.Println("Server 1 is running on port 8081...")
	log.Fatal(http.ListenAndServe(":8081", nil))
}