package v1alpha1

import (
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

const (
	EventTypeNormal  string = "Normal"
	EventTypeWarning string = "Warning"
)

type Rds struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              RdsSpec   `json:"spec"`
	Status            RdsStatus `json:"status"`
}

type RdsSpec struct {
	Availabilityzone  string     `json:"availabilityzone"`
	Backuprestoretime string     `json:"backuprestoretime"`
	Backupstarttime   string     `json:"backupstarttime"`
	Backupkeepdays    int        `json:"backupkeepdays"`
	Databases         *Databases `json:"databases"`
	Datastoretype     string     `json:"datastoretype"`
	Datastoreversion  string     `json:"datastoreversion"`
	Flavorref         string     `json:"flavorref"`
	Hamode            string     `json:"hamode,omitempty"`
	Hareplicationmode string     `json:"hareplicationmode,omitempty"`
	Port              string     `json:"port"`
	Password          string     `json:"password"`
	Region            string     `json:"region"`
	Subnet            string     `json:"subnet"`
	Securitygroup     string     `json:"securitygroup"`
	Users             []Users    `json:"users"`
	Volumetype        string     `json:"volumetype"`
	Volumesize        int        `json:"volumesize"`
	Vpc               string     `json:"vpc"`
}

type Users struct {
	Name       string   `json:"name"`
	Password   string   `json:"password"`
	Privileges []string `json:"privileges,omitempty"`
}

type Databases struct {
	Name string `json:"name"`
}

type RdsStatus struct {
	Id     string `json:"id"`
	Ip     string `json:"ip"`
	Logs   bool   `json:"logs"`
	Reboot bool   `json:"reboot"`
	Status string `json:"status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type RdsList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []Rds `json:"items"`
}

func (e *Rds) HasChanged(other *Rds) bool {
	//	return e.Spec != other.Spec || e.Status != other.Status
	return reflect.DeepEqual(e.Spec, other.Spec) || reflect.DeepEqual(e.Status, other.Status)
}
