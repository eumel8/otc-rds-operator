package main

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
	"io/ioutil"
	"net/http"
)

const RdsApiReply = `
{
	"apiVersion": "otc.mcsps.de/v1alpha1",
	"items": [],
	"kind": "RdsList",
	"metadata": {
	  "continue": "",
	  "resourceVersion": "265294298"
	}
  }
`
const KubeConfig = `
apiVersion: v1
clusters:
- cluster:
    server: http://127.0.0.1:8443
  name: test-server
contexts:
- context:
    cluster: test-server
    user: test-server
  name: test-server
current-context: test-server
kind: Config
preferences: {}
users:
- name: test-server
  user:
    token: kubeconfig-u-xxxx
	
`

func MockMuxer() {
	mux := http.NewServeMux()

	// https://stackoverflow.com/questions/47148240/correct-way-to-match-url-path-to-page-name-with-url-routing-in-go
	// watch: /apis/otc.mcsps.de/v1alpha1/rdss?allowWatchBookmarks=true&resourceVersion=265294298&timeout=5m29s&timeoutSeconds=329&watch=true
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path,"/apis/otc.mcsps.de/v1alpha1/rdss") {
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, RdsApiReply)
			time.Sleep(3 * time.Second)
			return
		}
		if r.URL.Path != "/" {
			
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			uri := r.URL.String()
			fmt.Printf("Uri: %s\n", string(uri))
			body, err := ioutil.ReadAll(r.Body)
			if err != nil {
				panic(err)
			}
			fmt.Printf("Body: %s\n", body)
			return	
		}
	})

	fmt.Println("Listening...")

	var retries int = 3

	for retries > 0 {
		err := http.ListenAndServe("127.0.0.1:8443", mux)
		if err != nil {
			fmt.Println("Restart http server ... ", err)
			retries -= 1
		} else {
			break
		}
	}

}

func Test_main(t *testing.T) {
	os.Setenv("KUBECONFIG", "test-fixtures/kube-config.yaml")
	go MockMuxer()
    timeout := time.After(5 * time.Second)
    done := make(chan bool)
    go func() {
        main() // We make a roughly fly over test if controller is starting
        time.Sleep(3 * time.Second)
        done <- true
    }()

    select {
    case <-timeout:
    case <-done:
    }
}