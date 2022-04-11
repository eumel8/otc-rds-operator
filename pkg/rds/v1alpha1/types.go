package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type Rds struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              RdsSpec `json:"spec"`
}

type RdsSpec struct {
	Message           string `json:"message"`
	Datastoretype     string `json:"datastoretype"`
	Datastoreversion  string `json:"datastoreversion"`
	Volumetype        string `json:"volumetype"`
	Volumesize        int    `json:"volumesize"`
	Hamode            string `json:"hamode"`
	Hareplicationmode string `json:"hareplicationmode"`
	Port              string `json:"port"`
	Password          string `json:"password"`
	Backupstarttime   string `json:"backupstarttime"`
	Backupkeepdays    int    `json:"backupkeepdays"`
	Flavorref         string `json:"flavorref"`
	Region            string `json:"region"`
	Availabilityzone  string `json:"Availabilityzone"`
	Vpc               string `json:"vpc"`
	Subnet            string `json:"subnet"`
	Securitygroup     string `json:"securitygroup"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type RdsList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []Rds `json:"items"`
}
