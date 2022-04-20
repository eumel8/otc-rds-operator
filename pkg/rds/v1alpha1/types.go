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
	Availabilityzone  string `json:"availabilityzone"`
	Backuprestoretime string `json:"backuprestoretime"`
	Backupstarttime   string `json:"backupstarttime"`
	Backupkeepdays    int    `json:"backupkeepdays"`
	Datastoretype     string `json:"datastoretype"`
	Datastoreversion  string `json:"datastoreversion"`
	Flavorref         string `json:"flavorref"`
	Hamode            string `json:"hamode"`
	Hareplicationmode string `json:"hareplicationmode"`
	Port              string `json:"port"`
	Password          string `json:"password"`
	Reboot            bool   `json:"reboot"`
	Region            string `json:"region"`
	Subnet            string `json:"subnet"`
	Securitygroup     string `json:"securitygroup"`
	Volumetype        string `json:"volumetype"`
	Volumesize        int    `json:"volumesize"`
	Vpc               string `json:"vpc"`
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
