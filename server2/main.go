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
		l:= log.New(os.Stdout, "[server2]", log.Ldate|log.Ltime)
		l.Printf("running...")
		io.WriteString(w, "Hello, World!")
	}
	http.HandleFunc("/", homeHandler)
	fmt.Println("Server 2 is running on port 8082...")
	log.Fatal(http.ListenAndServe(":8082", nil))
}