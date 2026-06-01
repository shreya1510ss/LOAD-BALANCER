package main

import(
	"net/http"
	"net/url"
	"time"
	"fmt"
)

func main(){
	num:=1
	for {
		fmt.Println("total number of requessts servered", num)
		healthCheck("localhost:8080")
		num+=1
	}

}

func healthCheck(addr string) bool{
     url, err := url.Parse("http://"+ addr)
	 if err != nil {
		 return false
	 }

	 client := http.Client{
		 Timeout: 5 * time.Second,
	 }

	 resp, err:= client.Get(url.String())
	 if err != nil {
		 return false
	 }

	 defer resp.Body.Close()
	 return resp.StatusCode == http.StatusOK
	 


}