package controller

import (
	"testing"

	golangsdk "github.com/opentelekomcloud/gophertelekomcloud"
)

func Test_rootURL(t *testing.T) {
	type args struct {
		c *golangsdk.ServiceClient
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := rootURL(tt.args.c); got != tt.want {
				t.Errorf("rootURL() = %v, want %v", got, tt.want)
			}
		})
	}
}
