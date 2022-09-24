package main

import (
	"fmt"
	"os"
	"testing"

	"io/ioutil"
	"net/http"
	// kubernetes "k8s.io/client-go/kubernetes/fake"
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

	mux.HandleFunc("/apis/otc.mcsps.de/v1alpha1/rdss?limit=500&resourceVersion=0", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, RdsApiReply)
		}
	})
	// watch: /apis/otc.mcsps.de/v1alpha1/rdss?allowWatchBookmarks=true&resourceVersion=265294298&timeout=5m29s&timeoutSeconds=329&watch=true
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)

			//_, _ = fmt.Fprint(w, "OK")
			_, _ = fmt.Fprint(w, RdsApiReply)
			// Debug output of the request
			uri := r.URL.String()
			fmt.Printf("Uri: %s\n", string(uri))
			body, err := ioutil.ReadAll(r.Body)
			if err != nil {
				panic(err)
			}
			fmt.Printf("Body: %s\n", body)
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
	tests := []struct {
		name string
	}{
		// TODO: Add test cases.
		{"maintest"},
	}
	os.Setenv("KUBECONFIG", "test-fixtures/kube-config.yaml")
	go MockMuxer()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			main()
		})
	}
}
