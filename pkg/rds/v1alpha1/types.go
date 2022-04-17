package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type Rds struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              RdsSpec   `json:"spec"`
	Status            RdsStatus `json:"status"`
}

type RdsStatus struct {
	Id     string `json:"id"`
	Ip     string `json:"ip"`
	Status string `json:"status"`
}

type RdsSpec struct {
	Datastoretype     string `json:"datastoretype"`
	Datastoreversion  string `json:"datastoreversion"`
	Volumetype        string `json:"volumetype"`
	Volumesize        int    `json:"volumesize"`
	Hamode            string `json:"hamode"`
	Hareplicationmode string `json:"hareplicationmode"`
	Id                string `json:"id"`
	Port              string `json:"port"`
	Password          string `json:"password"`
	Backupstarttime   string `json:"backupstarttime"`
	Backupkeepdays    int    `json:"backupkeepdays"`
	Flavorref         string `json:"flavorref"`
	Region            string `json:"region"`
	Availabilityzone  string `json:"availabilityzone"`
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

func (e *Rds) HasChanged(other *Rds) bool {
	return e.Spec.Flavorref != other.Spec.Flavorref || e.Spec.Volumesize != other.Spec.Volumesize
}
