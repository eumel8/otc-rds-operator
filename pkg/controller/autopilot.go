package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	//"github.com/davecgh/go-spew/spew"

	rdsv1alpha1clientset "github.com/eumel8/otc-rds-operator/pkg/rds/v1alpha1/apis/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
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

type SubscriberMessage struct {
	MessageType        string           `json:"message_type"`
	AlarmId            string           `json:"alarm_id"`
	AlarmName          string           `json:"alarm_name"`
	AlarmStatus        string           `json:"alarm_status"`
	Time               int64            `json:"time"`
	Namespace          string           `json:"namespace"`
	MetriyName         string           `json:"metric_name"`
	Dimension          string           `json:"dimension"`
	Period             int              `json:"period"`
	Filter             string           `json:"filter"`
	ComparisonOperator string           `json:"comparison_operator"`
	Value              int              `json:"value"`
	Unit               string           `json:"unit"`
	Count              int              `json:"count"`
	AlarmValue         []AlarmValue     `json:"alarmValue"`
	SmSContent         string           `json:"sms_content"`
	TemplateVariable   TemplateVariable `json:"template_variable"`
}
type AlarmValue struct {
	Time  int64   `json:"time"`
	Value float64 `json:"value"`
}

type TemplateVariable struct {
	IsAlarm           bool   `json:"IsAlarm"`
	IsCycleTrigger    bool   `json:"IsCycleTrigger"`
	AlarmLevel        string `json:"AlarmLevel"`
	Region            string `json:"Region"`
	ResourceId        string `json:"ResourceId"`
	AlarmRule         string `json:"AlarmRule"`
	CurrentDate       string `json:"CurrentData"`
	AlarmTime         string `json:"AlarmTime"`
	DataPoint         string `json:"DataPoint"`
	DataPointTime     string `json:"DataPointTime"`
	AlarmRuleName     string `json:"AlarmRuleName"`
	AlarmId           string `json:"AlarmId"`
	AlarmDesc         string `json:"AlarmDesc"`
	MonitoringRange   string `json:"MonitoringRange"`
	IsOriginalValue   bool   `json:"IsOriginalValue"`
	Period            string `json:"Period"`
	Filter            string `json:"Filter"`
	ComparionOperator string `json:"ComparisonOperator"`
	Value             string `json:"Value"`
	Unit              string `json:"Unit"`
	Count             int    `json:"Count"`
	EventContent      string `json:"EventComntent"`
	IsIEC             bool   `json:"IsIEC"`
}

func (c *Controller) SmnReceiver(ctx context.Context) error {
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
				c.logger.Error(err)
			}
			err = json.Unmarshal([]byte(req), &subscriber)
			if err != nil {
				c.logger.Error(err)
			}
			// subscribe to smn topic
			if subscriber.Subscribeurl != "" {
				c.logger.Info("Subscriber request: ", subscriber.Topicurn)
				_, err = http.Get(subscriber.Subscribeurl)
				if err != nil {
					c.logger.Error(err)
				}
			}
			// action on events
			if subscriber.Signature != "" {
				c.logger.Info("Event request: ", subscriber.Topicurn)
				rdsNsName := strings.Split(subscriber.Topicurn, ":")[4]
				namespace := strings.Split(rdsNsName, "_")[0]
				rdsName := strings.Split(rdsNsName, "_")[1]

				restConfig, err := rest.InClusterConfig()
				if err != nil {
					err := fmt.Errorf("error init in-cluster config: %v", err)
					c.logger.Error(err)
				}
				rdsclientset, err := rdsv1alpha1clientset.NewForConfig(restConfig)
				if err != nil {
					err := fmt.Errorf("error creating rdsclientset: %v", err)
					c.logger.Error(err)
				}
				returnRds, err := rdsclientset.McspsV1alpha1().Rdss(namespace).Get(ctx, rdsName, metav1.GetOptions{})
				if err != nil {
					err := fmt.Errorf("autopilot returnRds error: %v", err)
					c.logger.Error(err)
				}

				c.logger.Debug("SMN received message: ", subscriber.Message)
				if strings.Contains(subscriber.Message, "rds039_disk_util") {
					c.logger.Info("rds039_disk_util alarm ", rdsName, namespace)
					returnRds.Spec.Volumesize = returnRds.Spec.Volumesize + 10
					returnNewRds, err := rdsclientset.McspsV1alpha1().Rdss(namespace).Update(ctx, returnRds, metav1.UpdateOptions{})
					if returnRds.Spec.Volumesize != returnNewRds.Spec.Volumesize {
						err := fmt.Errorf("autopilot error update volumesize spec")
						c.logger.Error(err)
					}
					if err != nil {
						err := fmt.Errorf("error update rds rds039_disk_util alarm: %v", err)
						c.logger.Error(err)
					}

				}
				if strings.Contains(subscriber.Message, "rds001_cpu_util") {
					c.logger.Info("rds001_cpu_util alarm for ", rdsName, namespace)
					newFlavor, err := c.RdsFlavorLookup(returnRds, "cpu")
					if err != nil {
						err := fmt.Errorf("error lookup next flavor rds001_cpu_util alarm: %v", err)
						c.logger.Error(err)
					} else {
						returnRds.Spec.Flavorref = newFlavor
						_, err := rdsclientset.McspsV1alpha1().Rdss(namespace).Update(ctx, returnRds, metav1.UpdateOptions{})
						if err != nil {
							err := fmt.Errorf("error update rds for rds001_cpu_util alarm: %v", err)
							c.logger.Error(err)
						}
					}
				}
				if strings.Contains(subscriber.Message, "rds002_mem_util") {
					c.logger.Info("rds002_mem_util alarm for", rdsName, namespace)
					newFlavor, err := c.RdsFlavorLookup(returnRds, "mem")
					if err != nil {
						err := fmt.Errorf("error lookup next flavor rds002_mem_util alarm: %v", err)
						c.logger.Error(err)
					} else {
						returnRds.Spec.Flavorref = newFlavor
						_, err := rdsclientset.McspsV1alpha1().Rdss(namespace).Update(ctx, returnRds, metav1.UpdateOptions{})
						if err != nil {
							err := fmt.Errorf("error update rds for rds002_mem_util alarm: %v", err)
							c.logger.Error(err)
						}
					}
				}
			}
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
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
