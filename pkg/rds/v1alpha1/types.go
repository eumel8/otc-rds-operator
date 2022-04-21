package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type Rds struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Events            RdsEvents `json:"events"`
	Spec              RdsSpec   `json:"spec"`
	Status            RdsStatus `json:"status"`
}

type RdsEvents struct {
	Errorlog string `json:"errorlog"`
	Slowlog  string `json:"slowlog"`
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
	Region            string `json:"region"`
	Subnet            string `json:"subnet"`
	Securitygroup     string `json:"securitygroup"`
	Volumetype        string `json:"volumetype"`
	Volumesize        int    `json:"volumesize"`
	Vpc               string `json:"vpc"`
}

type RdsStatus struct {
	Id     string `json:"id"`
	Ip     string `json:"ip"`
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
	return e.Spec != other.Spec || e.Status != other.Status
	//	return e.Spec.Flavorref != other.Spec.Flavorref || e.Spec.Volumesize != other.Spec.Volumesize || e.Status.Reboot != other.Status.Reboot || e.Spec.Backuprestoretime != other.Spec.Backuprestoretime
}
