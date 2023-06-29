// main.go
package main

import (
	"fmt"
	"net/http"
	"os"
	"time"
)

func main() {
	status := 0
	start := time.Now()
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("%v / request\n", time.Now().Format("2006-01-02 15:04:05"))
		host, _ := os.Hostname()
		if status == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "[%v] Internal Server Error, interval: %v", host,
				time.Now().Sub(start)/time.Nanosecond)
			start = time.Now()
		} else {
			fmt.Fprintln(w, fmt.Sprintf("[%v] hello", host))
		}
	})
	http.HandleFunc("/fail", func(w http.ResponseWriter, r *http.Request) {
		status = 1
		w.Write([]byte("ok"))
	})
	http.HandleFunc("/success", func(w http.ResponseWriter, r *http.Request) {
		status = 0
		w.Write([]byte("ok"))
	})
	http.HandleFunc("/healthCheck", func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("%v /healthCheck request\n", time.Now().Format("2006-01-02 15:04:05"))
		if status == 1 {
			time.Sleep(5 * time.Second)
		} else {
			w.Write([]byte("ok"))
		}
	})

	http.ListenAndServe(":8090", nil)
}
