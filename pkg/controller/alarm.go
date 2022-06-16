package controller

import (
	"fmt"
	"strings"

	golangsdk "github.com/opentelekomcloud/gophertelekomcloud"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack/cloudeyeservice/alarmrule"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack/smn/v2/subscriptions"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack/smn/v2/topics"
	"github.com/opentelekomcloud/gophertelekomcloud/pagination"
)

const (
	rootPath = "alarms"
)

func rootURL(c *golangsdk.ServiceClient) string {
	return c.ServiceURL(rootPath)
}

/*
type AlarmRule struct {
	AlarmName               string        `json:"alarm_name"`
	AlarmID                 string        `json:"alarm_id"`
	AlarmDescription        string        `json:"alarm_description"`
	AlarmType               string        `json:"alarm_type"`
	AlarmLevel              int           `json:"alarm_level"`
	Metric                  MetricInfo    `json:"metric"`
	Condition               ConditionInfo `json:"condition"`
	AlarmActions            []ActionInfo  `json:"alarm_actions"`
	InsufficientdataActions []ActionInfo  `json:"insufficientdata_actions"`
	OkActions               []ActionInfo  `json:"ok_actions"`
	AlarmEnabled            bool          `json:"alarm_enabled"`
	AlarmActionEnabled      bool          `json:"alarm_action_enabled"`
	UpdateTime              int64         `json:"update_time"`
	AlarmState              string        `json:"alarm_state"`
}

type ConditionInfo struct {
	Period             int    `json:"period"`
	Filter             string `json:"filter"`
	ComparisonOperator string `json:"comparison_operator"`
	Value              int    `json:"value"`
	Unit               string `json:"unit"`
	Count              int    `json:"count"`
}

type MetricInfo struct {
	Namespace  string          `json:"namespace"`
	MetricName string          `json:"metric_name"`
	Dimensions []DimensionInfo `json:"dimensions"`
}

type ActionInfo struct {
	Type             string   `json:"type"`
	NotificationList []string `json:"notificationList"`
}

type DimensionInfo struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}
*/
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
		// return fmt.Errorf("unable to create topic: %v", err)
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
		// return fmt.Errorf("error create subscription: %v", err)
	}

	// list all alarmrules
	pages, err := AlarmRuleList(ces, &ListAlarmRuleOpts{}).AllPages()
	if err != nil {
		return fmt.Errorf("ces list failed allpages: %v", err)
	}
	alarms, err := ExtractAlarmRules(pages)
	if err != nil {
		return fmt.Errorf("ces list extract failed: %v", err)
	}
	for _, alarm := range alarms {
		if alarm.AlarmName == nsRds+"-disc-util" {
			return fmt.Errorf("alarmrule exists for %s", nsRds)
		}
	}

	alarmDiscUtil := alarmrule.CreateOpts{
		AlarmName:        nsRds + "-disc-util",
		AlarmDescription: "RDS Operator Autopilot",
		AlarmLevel:       2,
		Metric: alarmrule.MetricOpts{
			Namespace:  "SYS.RDS",
			MetricName: "rds039_disk_util",
			Dimensions: []alarmrule.DimensionOpts{{
				Name:  "rds_instance_id",
				Value: instanceId,
			}},
		},
		Condition: alarmrule.ConditionOpts{
			Period:             300,
			SuppressDuration:   1800,
			Filter:             "average",
			ComparisonOperator: ">=",
			Value:              90,
			Unit:               "",
			Count:              3,
		},
		AlarmActions: []alarmrule.ActionOpts{{
			Type:             "notification",
			NotificationList: []string{topic.TopicUrn},
		}},
		AlarmEnabled:       true,
		AlarmActionEnabled: true,
	}

	alarmCpuUtil := alarmrule.CreateOpts{
		AlarmName:        nsRds + "-cpu-util",
		AlarmDescription: "RDS Operator Autopilot",
		AlarmLevel:       2,
		Metric: alarmrule.MetricOpts{
			Namespace:  "SYS.RDS",
			MetricName: "rds001_cpu_util",
			Dimensions: []alarmrule.DimensionOpts{{
				Name:  "rds_instance_id",
				Value: instanceId,
			}},
		},
		Condition: alarmrule.ConditionOpts{
			Period:             300,
			SuppressDuration:   1800,
			Filter:             "average",
			ComparisonOperator: ">=",
			Value:              90,
			Unit:               "",
			Count:              3,
		},
		AlarmActions: []alarmrule.ActionOpts{{
			Type:             "notification",
			NotificationList: []string{topic.TopicUrn},
		}},
		AlarmEnabled:       true,
		AlarmActionEnabled: true,
	}
	alarmMemUtil := alarmrule.CreateOpts{
		AlarmName:        nsRds + "-mem-util",
		AlarmDescription: "RDS Operator Autopilot",
		AlarmLevel:       2,
		Metric: alarmrule.MetricOpts{
			Namespace:  "SYS.RDS",
			MetricName: "rds002_mem_util",
			Dimensions: []alarmrule.DimensionOpts{{
				Name:  "rds_instance_id",
				Value: instanceId,
			}},
		},
		Condition: alarmrule.ConditionOpts{
			Period:             300,
			SuppressDuration:   1800,
			Filter:             "average",
			ComparisonOperator: ">=",
			Value:              90,
			Unit:               "",
			Count:              3,
		},
		AlarmActions: []alarmrule.ActionOpts{{
			Type:             "notification",
			NotificationList: []string{topic.TopicUrn},
		}},
		AlarmEnabled:       true,
		AlarmActionEnabled: true,
	}
	alarmDiscUtilResult, err := alarmrule.Create(ces, alarmDiscUtil).Extract()
	c.logger.Info(alarmDiscUtilResult.AlarmID)
	if err != nil {
		c.logger.Debug("error creating alarmrule alarmDiscUtil: %v", err)
		// return fmt.Errorf("error creating alarmrule alarmDiscUtil: %v", err)
	}
	alarmCpuUtilResult, err := alarmrule.Create(ces, alarmCpuUtil).Extract()
	c.logger.Info(alarmCpuUtilResult.AlarmID)
	if err != nil {
		c.logger.Debug("error creating alarmrule alarmCpuUtil: %v", err)
		// return fmt.Errorf("error creating alarmrule alarmCpuUtil: %v", err)
	}

	alarmMemUtilResult, err := alarmrule.Create(ces, alarmMemUtil).Extract()
	if err != nil {
		c.logger.Debug("error creating alarmrule alarmMemUtil: %v", err)
		// return fmt.Errorf("error creating alarmrule alarmMemUtil: %v", err)
	}
	c.logger.Info(alarmMemUtilResult.AlarmID)
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
	pages, err := AlarmRuleList(ces, &ListAlarmRuleOpts{}).AllPages()
	if err != nil {
		return fmt.Errorf("ces list failed allpages: %v", err)
	}
	alarms, err := ExtractAlarmRules(pages)
	if err != nil {
		return fmt.Errorf("ces list extract failed: %v", err)
	}
	for _, alarm := range alarms {
		if strings.Contains(alarm.AlarmName, nsRds) {
			alarmDeleteResult := alarmrule.Delete(ces, alarm.AlarmID)
			c.logger.Debug("ALARM Rule Delete: ", alarmDeleteResult.ErrResult)
		}
	}

	// delete topic/smn
	tl, err := topics.List(smn).Extract()
	if err != nil {
		return fmt.Errorf("unable to get topic list: %v", err)
	}
	for _, tc := range tl {
		if tc.Name == nsRds {
			// smnDeleteResult,err := subscriptions.Delete(smn, tc.TopicUrn).ExtractJobResponse()
			topicDeleteResult := topics.Delete(smn, tc.TopicUrn)
			c.logger.Debug("ALARM Topic Delete: ", topicDeleteResult.ErrResult)
			// c.logger.Debug("ALARM SMN Delete: ", smnDeleteResult)
			//c.logger.Debug("topic exists for ", nsRds)
		}
	}
	return nil
}
