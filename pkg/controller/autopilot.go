package controller

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	//"github.com/davecgh/go-spew/spew"
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
	MessageType        string       `json:"message_type"`
	AlarmId            string       `json:"alarm_id"`
	AlarmName          string       `json:"alarm_name"`
	AlarmStatus        string       `json:"alarm_status"`
	Time               int64        `json:"time"`
	Namespace          string       `json:"namespace"`
	MetriyName         string       `json:"metric_name"`
	Dimension          string       `json:"dimension"`
	Period             int          `json:"period"`
	Filter             string       `json:"filter"`
	ComparisonOperator string       `json:"comparison_operator"`
	Value              int          `json:"value"`
	Unit               string       `json:"unit"`
	Count              int          `json:"count"`
	AlarmValue         []AlarmValue `json:"alarmValue"`
	SmSContent         string       `json:"sms_content"`
}
type AlarmValue struct {
	Time  int64   `json:"time"`
	Value float64 `json:"value"`
}

/* sms_content struct
IsAlarm bool `json:"IsAlarm"`
	IsCycleTrigger bool `json:"IsCycleTrigger"`
	AlarmLevel string `json:"AlarmLevel"`
	Region string `json:"Region"`
	ResourceId string `json:"ResourceId"`
	AlarmRule string `json:"AlarmRule"`
	CurrentDate string `json:"CurrentData"`
	AlarmTime string `json:"AlarmTime"`
	DataPoint string `json:"DataPoint"`
	DataPointTime string `json:"DataPointTime"`
	AlarmRuleName string `json:"AlarmRuleName"`
	AlarmId: string `json:"AlarmId"`
	AlarmDesc: string `json:"AlarmDesc"`
	MonitoringRange: string `json:"MonitoringRange"`
	IsOriginalValue: bool `json:"IsOriginalValue"`
	Period: string `json:"Period"`
	Filter: string `json:"Filter"`
	ComparionOperator: string `json:"ComparisonOperator"`
	Value":"12.00%","
	Unit":"%","
	Count":1,"
	EventContent":"","
	IsIEC":false}}{"level":"info","msg":"
	Event request: urn:smn:eu-de:7c3ec0b3db5f476990043258670caf82:my-rds-ha",
	"node":"otc-rds-operator-c76687d8b-x69mg",
	"service":"otc-rds-operator",
	"time":"2022-05-08T08:26:26Z",
	"type":"controller"}
*/

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
				//c.logger.Info("Event message: ", strings.Split(subscriber.Message, ","))

				cleanMessage := strings.Replace(subscriber.Message, "\\", "", -1)
				fmt.Println(cleanMessage)
				var mySubscriberMessage SubscriberMessage
				err := json.Unmarshal([]byte(cleanMessage), &mySubscriberMessage)
				if err != nil {
					fmt.Println(err)
				}
				fmt.Println("AlarmName")
				fmt.Println(mySubscriberMessage.AlarmName)

				//spew.Dump(subscriber)
				/*
					for _, sm := range subscriber.Message {
						if err != nil {
							fmt.Println(err)
						}
						fmt.Printf("message range: %s", sm.AlarmName)
						// fmt.Printf("event alarm_name: %s", string(sm.AlarmName))
					}
				*/
				/*
					{"message_type":"alarm","alarm_id":"al1651967846367MVO1yKvWy","alarm_name":"my-rds-ha-disc-util","alarm_status":"alarm","time":1651998061184,"namespace":"SYS.RDS","metric_name":"rds039_disk_util","dimension":"rds_instance_id:9a22c728f48142f88339dc5bfa06d592no01","period":300,"filter":"average","comparison_operator":"\u003e=","value":12,"unit":"","count":1,"alarmValue":[{"time":1651998000000,"value":19.96}],"sms_content":"[eu-de][Major Alarm]Dear customer: The Storage Space Usage of Relational Database Service-MySQL Instances \"my-rds-ha_node0\" (ID: 9a22c728f48142f88339dc5bfa06d592no01) Avg. \u003e= 12.00% for 1 consecutive periods of 5 minutes, at 05 08, 2022 10:21:01 GMT+02:00 triggered an alarm, You can log in to the Cloud Eye console to view details.","template_variable":{"AccountName":"customer","Namespace":"Relational Database Service","DimensionName":"MySQL Instances","ResourceName":"my-rds-ha_node0","MetricName":"Storage Space Usage","IsAlarm":true,"IsCycleTrigger":false,"AlarmLevel":"Major","Region":"eu-de","ResourceId":"9a22c728f48142f88339dc5bfa06d592no01","AlarmRule":"","CurrentData":"19.96%","AlarmTime":"05 08, 2022 10:21:01 GMT+02:00","DataPoint":{"05 08, 2022 10:20:00 GMT+02:00":"19.96%"},"DataPointTime":["05 08, 2022 10:20:00 GMT+02:00"],"AlarmRuleName":"my-rds-ha-disc-util","AlarmId":"al1651967846367MVO1yKvWy","AlarmDesc":"RDS Operator Autopilot","MonitoringRange":"Specific resources","IsOriginalValue":false,"Period":"5 minutes","Filter":"Avg.","ComparisonOperator":"\u003e=","Value":"12.00%","Unit":"%","Count":1,"EventContent":"","IsIEC":false}}{"level":"info","msg":"Event request: urn:smn:eu-de:7c3ec0b3db5f476990043258670caf82:my-rds-ha","node":"otc-rds-operator-c76687d8b-x69mg","service":"otc-rds-operator","time":"2022-05-08T08:26:26Z","type":"controller"}
				*/
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
