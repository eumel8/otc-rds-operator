package controller

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

type Subscriber struct {
	Subscribeurl     string `json:"subscribe_url"`
	Signature        string `json:"signature"`
	Topicurn         string `json:"topic_urn"`
	Messageid        string `json:"message_id"`
	Signatureversion string `json:"signature_version"`
	Type             string `json:"type"`
	Message          string `json:"message"`
	Signaturecerturl string `json:"signing_cert_url"`
	Timestamp        string `json:"timestamp"`
}

func SmnReceiver() error {
	var subscriber Subscriber

	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			req, err := ioutil.ReadAll(r.Body)
			if err != nil {
				fmt.Println(err)
			}
			fmt.Printf("server: request body: %s\n", req)
			err = json.Unmarshal([]byte(req), &subscriber)
			if err != nil {
				fmt.Println(err)
			}
			if subscriber.Subscribeurl != "" {
				fmt.Println(subscriber.Subscribeurl)
				_, err = http.Get(subscriber.Subscribeurl)
				if err != nil {
					fmt.Println(err)
				}
			}
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			// _, _ = fmt.Fprint(w, ProviderGetResponse)
		case "POST":
			req, err := ioutil.ReadAll(r.Body)
			if err != nil {
				fmt.Println(err)
			}
			fmt.Printf("server: request body: %s\n", req)
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			// _, _ = fmt.Fprint(w, ProviderPostResponse)
		}
	})
	mux.HandleFunc("/notify", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			req, err := ioutil.ReadAll(r.Body)
			if err != nil {
				fmt.Println(err)
			}
			fmt.Printf("server: request body: %s\n", req)
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			// _, _ = fmt.Fprint(w, ProviderGetResponse)
		case "POST":
			req, err := ioutil.ReadAll(r.Body)
			if err != nil {
				fmt.Println(err)
			}
			fmt.Printf("server: request body: %s\n", req)
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
