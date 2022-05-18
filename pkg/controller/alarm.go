package controller

import (
	"fmt"

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
	nsRds := namespace + "-" + rdsName
	// initial provider
	provider, err := getProvider()
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
		return fmt.Errorf("unable to initialize ces client: %v", err)
	}

	// check if topic for rds exists
	tl, err := topics.List(smn).Extract()
	if err != nil {
		return fmt.Errorf("unable to get topic list: %v", err)
	}
	for _, tc := range tl {
		if tc.Name == nsRds {
			c.logger.Debug("topic exists for %s", nsRds)
			return nil
		}
	}

	// create topic
	newTopic := topics.CreateOps{
		Name:        nsRds,
		DisplayName: nsRds,
	}
	topic, err := topics.Create(smn, newTopic).Extract()
	if err != nil {
		return fmt.Errorf("unable to create topic: %v", err)
	}

	// check of subscription exists for rds
	smnList, err := subscriptions.List(smn).Extract()
	if err != nil {
		return fmt.Errorf("error extracting subscription list: %v", err)
	}
	for _, sl := range smnList {
		if sl.Endpoint == smnEndpoint {
			return fmt.Errorf("subscription exists for %s", nsRds)
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
		return fmt.Errorf("error create subscription: %v", err)
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
			Filter:             "average",
			ComparisonOperator: ">=",
			Value:              12,
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
	if err != nil {
		return fmt.Errorf("error creating alarmrule: %v", err)
	}
	fmt.Println(alarmDiscUtilResult.AlarmID)

	alarmCpuUtilResult, err := alarmrule.Create(ces, alarmCpuUtil).Extract()
	if err != nil {
		return fmt.Errorf("error creating alarmrule: %v", err)
	}
	fmt.Println(alarmCpuUtilResult.AlarmID)

	alarmMemUtilResult, err := alarmrule.Create(ces, alarmMemUtil).Extract()
	if err != nil {
		return fmt.Errorf("error creating alarmrule: %v", err)
	}
	fmt.Println(alarmMemUtilResult.AlarmID)
	return nil
}
