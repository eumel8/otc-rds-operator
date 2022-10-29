package controller

import (
	"fmt"
	"strings"

	golangsdk "github.com/opentelekomcloud/gophertelekomcloud"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack"

	// "github.com/opentelekomcloud/gophertelekomcloud/openstack/cloudeyeservice/alarmrule"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack/ces/v1/alarms"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack/smn/v2/subscriptions"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack/smn/v2/topics"
	// "github.com/opentelekomcloud/gophertelekomcloud/pagination"
)

// alarmrule has no list function
/*
const (
	rootPath = "alarms"
)

func rootURL(c *golangsdk.ServiceClient) string {
	return c.ServiceURL(rootPath)
}

type AlarmRulePage struct {
	pagination.SinglePageBase
}

type ListAlarmRuleOpts struct {
	AlarmName    string   `q:"alarm_name"`
	MetricAlarms []string `q:"metric_alarms"`
	MetaData     MetaData `q:"meta_data"`
}

type MetaData struct {
	Count  int    `q:"count"`
	Marker string `q:"marker"`
	Total  int    `q:"total"`
}

type ListAlarmRuleBuilder interface {
	ToAlarmRuleListDetailQuery() (string, error)
}

func (opts ListAlarmRuleOpts) ToAlarmRuleListDetailQuery() (string, error) {
	q, err := golangsdk.BuildQueryString(opts)
	if err != nil {
		return "", err
	}
	return q.String(), err
}

func AlarmRuleList(client *golangsdk.ServiceClient, opts ListAlarmRuleBuilder) pagination.Pager {
	url := rootURL(client)
	if opts != nil {
		query, err := opts.ToAlarmRuleListDetailQuery()

		if err != nil {
			return pagination.Pager{Err: err}
		}
		url += query
	}

	pageAlarmRuleList := pagination.NewPager(client, url, func(r pagination.PageResult) pagination.Page {
		return AlarmRulePage{pagination.SinglePageBase(r)}
	})

	alarmruleheader := map[string]string{"Content-Type": "application/json"}
	pageAlarmRuleList.Headers = alarmruleheader
	return pageAlarmRuleList
}

func ExtractAlarmRules(r pagination.Page) ([]alarmrule.AlarmRule, error) {
	var s []alarmrule.AlarmRule
	err := ExtractAlarmRulesInto(r, &s)
	return s, err
}

func ExtractAlarmRulesInto(r pagination.Page, v interface{}) error {
	return r.(AlarmRulePage).Result.ExtractIntoSlicePtr(v, "metric_alarms")
}
*/
func (c *Controller) CreateAlarm(instanceId string, smnEndpoint string, rdsName string, namespace string) error {
	nsRds := namespace + "_" + rdsName
	// initial provider
	provider, err := GetProvider()
	if err != nil {
		return fmt.Errorf("unable to initialize provider: %v", err)
	}
	// inital service clients for CES and SMN
	ces, err := openstack.NewCESClient(provider, golangsdk.EndpointOpts{})
	if err != nil {
		return fmt.Errorf("unable to initialize ces client: %v", err)
	}

	smn, err := openstack.NewSMNV2(provider, golangsdk.EndpointOpts{})
	if err != nil {
		return fmt.Errorf("unable to initialize smn client: %v", err)
	}
	// TODO separate return existing error
	// check if topic for rds exists

	tl, err := topics.List(smn).Extract()
	if err != nil {
		return fmt.Errorf("unable to get topic list: %v", err)
	}
	for _, tc := range tl {
		if tc.Name == nsRds {
			c.logger.Debug("topic exists for ", nsRds)
		}
	}

	// create topic
	newTopic := topics.CreateOps{
		Name:        nsRds,
		DisplayName: nsRds,
	}
	topic, err := topics.Create(smn, newTopic).Extract()
	if err != nil {
		c.logger.Debug("unable to create topic: %v", err)
	}

	// check of subscription exists for rds
	smnList, err := subscriptions.List(smn).Extract()
	if err != nil {
		return fmt.Errorf("error extracting subscription list: %v", err)
	}
	for _, sl := range smnList {
		if sl.Endpoint == smnEndpoint {
			c.logger.Error("subscription exists for ", nsRds)
		}
	}

	// create smn subscription
	newSmn := subscriptions.CreateOpts{
		Endpoint: smnEndpoint,
		Protocol: "https",
		Remark:   "RDS Operator",
	}
	_, err = subscriptions.Create(smn, newSmn, topic.TopicUrn).Extract()
	if err != nil {
		c.logger.Debug("error create subscription: %v", err)
	}

	// list all alarmrules
	//ListAlarmOpts :=
	alarmList, err := alarms.ListAlarms(ces, alarms.ListAlarmsOpts{})
	if err != nil {
		return fmt.Errorf("ces list alarm failed: %v", err)
	}
	//alarms, err := ExtractAlarmRules(pages)
	//if err != nil {
	//	return fmt.Errorf("ces list extract failed: %v", err)
	//}
	t := true
	for _, alarm := range alarmList.MetricAlarms {
		if alarm.AlarmName == nsRds+"-disc-util" {
			return fmt.Errorf("alarmrule exists for %s", nsRds)
		}
	}
	alarmDiscUtil := alarms.CreateAlarmOpts{
		AlarmName:        nsRds + "-disc-util",
		AlarmDescription: "RDS Operator Autopilot",
		AlarmLevel:       2,
		Metric: alarms.MetricForAlarm{
			Namespace:  "SYS.RDS",
			MetricName: "rds039_disk_util",
			Dimensions: []alarms.MetricsDimension{{
				Name:  "rds_instance_id",
				Value: instanceId,
			}},
		},
		Condition: alarms.Condition{
			Period: 300,
			//SuppressDuration:   1800,
			Filter:             "average",
			ComparisonOperator: ">=",
			Value:              90,
			Unit:               "",
			Count:              3,
		},
		AlarmActions: []alarms.AlarmActions{{
			Type:             "notification",
			NotificationList: []string{topic.TopicUrn},
		}},
		AlarmEnabled:       &t,
		AlarmActionEnabled: &t,
	}
	alarmCpuUtil := alarms.CreateAlarmOpts{
		AlarmName:        nsRds + "-cpu-util",
		AlarmDescription: "RDS Operator Autopilot",
		AlarmLevel:       2,
		Metric: alarms.MetricForAlarm{
			Namespace:  "SYS.RDS",
			MetricName: "rds001_cpu_util",
			Dimensions: []alarms.MetricsDimension{{
				Name:  "rds_instance_id",
				Value: instanceId,
			}},
		},
		Condition: alarms.Condition{
			Period: 300,
			// SuppressDuration:   1800,
			Filter:             "average",
			ComparisonOperator: ">=",
			Value:              90,
			Unit:               "",
			Count:              3,
		},
		AlarmActions: []alarms.AlarmActions{{
			Type:             "notification",
			NotificationList: []string{topic.TopicUrn},
		}},
		AlarmEnabled:       &t,
		AlarmActionEnabled: &t,
	}
	alarmMemUtil := alarms.CreateAlarmOpts{
		AlarmName:        nsRds + "-mem-util",
		AlarmDescription: "RDS Operator Autopilot",
		AlarmLevel:       2,
		Metric: alarms.MetricForAlarm{
			Namespace:  "SYS.RDS",
			MetricName: "rds002_mem_util",
			Dimensions: []alarms.MetricsDimension{{
				Name:  "rds_instance_id",
				Value: instanceId,
			}},
		},
		Condition: alarms.Condition{
			Period: 300,
			// SuppressDuration:   1800,
			Filter:             "average",
			ComparisonOperator: ">=",
			Value:              90,
			Unit:               "",
			Count:              3,
		},
		AlarmActions: []alarms.AlarmActions{{
			Type:             "notification",
			NotificationList: []string{topic.TopicUrn},
		}},
		AlarmEnabled:       &t,
		AlarmActionEnabled: &t,
	}
	alarmIdDiscUtil, err := alarms.CreateAlarm(ces, alarmDiscUtil)
	c.logger.Info(alarmIdDiscUtil)
	if err != nil {
		c.logger.Debug("error creating alarmrule alarmDiscUtil: %v", err)
	}
	alarmIdCpuUtil, err := alarms.CreateAlarm(ces, alarmCpuUtil)
	c.logger.Info(alarmIdCpuUtil)
	if err != nil {
		c.logger.Debug("error creating alarmrule alarmCpuUtil: %v", err)
	}

	alarmIdMemUtil, err := alarms.CreateAlarm(ces, alarmMemUtil)
	if err != nil {
		c.logger.Debug("error creating alarmrule alarmMemUtil: %v", err)
	}
	c.logger.Info(alarmIdMemUtil)
	return nil
}

func (c *Controller) DeleteAlarm(rdsName string, namespace string) error {
	nsRds := namespace + "_" + rdsName
	// initial provider
	provider, err := GetProvider()
	if err != nil {
		return fmt.Errorf("unable to initialize provider: %v", err)
	}
	// inital service clients for CES and SMN
	ces, err := openstack.NewCESClient(provider, golangsdk.EndpointOpts{})
	if err != nil {
		return fmt.Errorf("unable to initialize ces client: %v", err)
	}

	smn, err := openstack.NewSMNV2(provider, golangsdk.EndpointOpts{})
	if err != nil {
		return fmt.Errorf("unable to initialize smn client: %v", err)
	}

	// delete alarm rules

	alarmList, err := alarms.ListAlarms(ces, alarms.ListAlarmsOpts{})

	for _, alarm := range alarmList.MetricAlarms {
		if strings.Contains(alarm.AlarmName, nsRds) {
			alarmDeleteResult := alarms.DeleteAlarm(ces, alarm.AlarmId)
			c.logger.Debug("ALARM Rule Delete: ", alarmDeleteResult)
		}
	}

	// delete topic/smn
	tl, err := topics.List(smn).Extract()
	if err != nil {
		return fmt.Errorf("unable to get topic list: %v", err)
	}
	for _, tc := range tl {
		if tc.Name == nsRds {
			topicDeleteResult := topics.Delete(smn, tc.TopicUrn)
			c.logger.Debug("ALARM Topic Delete: ", topicDeleteResult.ErrResult)
		}
	}
	return nil
}
