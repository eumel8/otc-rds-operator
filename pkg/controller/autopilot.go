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

func (c *Controller) SmnReceiver() error {
	var subscriber Subscriber

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
		case "POST":
			req, err := ioutil.ReadAll(r.Body)
			if err != nil {
				fmt.Println(err)
			}
			// fmt.Printf("server: request body / POST: %s\n", req)
			err = json.Unmarshal([]byte(req), &subscriber)
			if err != nil {
				fmt.Println(err)
			}
			// subscribe to smn topic
			if subscriber.Subscribeurl != "" {
				c.logger.Info("Subscriber request: ", subscriber.Topicurn)
				_, err = http.Get(subscriber.Subscribeurl)
				if err != nil {
					fmt.Println(err)
				}
			}
			// action on events
			if subscriber.Signature != "" {
				c.logger.Info("Event request: ", subscriber.Topicurn)
				c.logger.Info("Event message: ", subscriber.Message)
				for _, sm := range subscriber.Message {
					if err != nil {
						fmt.Println(err)
					}
					fmt.Printf("%s", string(sm))
				}
			}
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			// w.WriteHeader(http.StatusCreated)
			// _, _ = fmt.Fprint(w, ProviderPostResponse)
		}
	})

	c.logger.Info("starting smn listener")

	var retries int = 3

	for retries > 0 {
		err := http.ListenAndServe("0.0.0.0:8080", mux)
		if err != nil {
			c.logger.Info("restart smn listener", err)
			retries -= 1
		} else {
			break
		}
	}
	return nil
}
