package controller

import (
	"fmt"
	"net/http"
)

func SmnReceiver() error {
	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			r := r.Body
			fmt.Println(r)
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			// _, _ = fmt.Fprint(w, ProviderGetResponse)
		case "POST":
			r := r.Body
			fmt.Println(r)
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			// _, _ = fmt.Fprint(w, ProviderPostResponse)
		}
	})
	mux.HandleFunc("/notify", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			r := r.Body
			fmt.Println(r)
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			// _, _ = fmt.Fprint(w, ProviderGetResponse)
		case "POST":
			r := r.Body
			fmt.Println(r)
			w.Header().Add("X-Subject-Token", "dG9rZW46IDEyMzQK")
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			// _, _ = fmt.Fprint(w, ProviderPostResponse)
		}
	})
	fmt.Println("Listening...")

	var retries int = 3

	for retries > 0 {
		err := http.ListenAndServe("0.0.0.0:8080", mux)
		if err != nil {
			fmt.Println("Restart http server ... ", err)
			retries -= 1
		} else {
			break
		}
	}
	return nil
}
