package marin3r

import (
	"errors"
	"fmt"
	monitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var (
	level1ApiUsageLowerThresholdPercent  = 80
	level1ApiUsageHigherThresholdPercent = 90
	level1ApiUsageCheckFrequencyMins     = 240
	level2ApiUsageLowerThresholdPercent  = 90
	level2ApiUsageHigherThresholdPercent = 95
	level2ApiUsageCheckFrequencyMins     = 120
	level3ApiUsageLowerThresholdPercent  = 95
	level3ApiUsageCheckFrequencyMins     = 30
	totalRequestsMetric                  = "ratelimit_service_rate_limit_apicast_ratelimit_generic_key_slowpath_total_hits"
)

func (r *Reconciler) newAlertsReconciler(rateLimitUnit string, rateLimitRequestsPerUnit uint32) (resources.AlertReconciler, error) {

	requestsAllowedPerSecond, err := getRateLimitInSeconds(rateLimitUnit, rateLimitRequestsPerUnit)
	if err != nil {
		return nil, err
	}

	level1Rule, err := getLevel1ApiUsageAlert(rateLimitUnit, rateLimitRequestsPerUnit, requestsAllowedPerSecond)
	if err != nil {
		return nil, err
	}
	level2Rule, err := getLevel2ApiUsageAlert(rateLimitUnit, rateLimitRequestsPerUnit, requestsAllowedPerSecond)
	if err != nil {
		return nil, err
	}
	level3Rule, err := getLevel3ApiUsageAlert(rateLimitUnit, rateLimitRequestsPerUnit, requestsAllowedPerSecond)
	if err != nil {
		return nil, err
	}

	return &resources.AlertReconcilerImpl{
		ProductName:  "3Scale",
		Installation: r.installation,
		Logger:       r.logger,
		Alerts: []resources.AlertConfiguration{
			{
				AlertName: "api-usage-alert-level1",
				Namespace: r.Config.GetNamespace(),
				GroupName: "api-usage.rules",
				Interval:  fmt.Sprintf("%dm", level1ApiUsageCheckFrequencyMins),
				Rules: []monitoringv1.Rule{
					*level1Rule,
				},
			},
			{
				AlertName: "api-usage-alert-level2",
				Namespace: r.Config.GetNamespace(),
				GroupName: "api-usage.rules",
				Interval:  fmt.Sprintf("%dm", level2ApiUsageCheckFrequencyMins),
				Rules: []monitoringv1.Rule{
					*level2Rule,
				},
			},
			{
				AlertName: "api-usage-alert-level3",
				Namespace: r.Config.GetNamespace(),
				GroupName: "api-usage.rules",
				Interval:  fmt.Sprintf("%dm", level3ApiUsageCheckFrequencyMins),
				Rules: []monitoringv1.Rule{
					*level3Rule,
				},
			},
		},
	}, nil
}

func getLevel1ApiUsageAlert(rateLimitUnit string, rateLimitRequestsPerUnit uint32, requestsAllowedPerSecond uint32) (*monitoringv1.Rule, error) {

	requestsAllowedOverTimePeriod := requestsAllowedPerSecond * uint32(level1ApiUsageCheckFrequencyMins) * 60

	return &monitoringv1.Rule{
		Alert: "Level1ThreeScaleApiUsageThresholdExceeded",
		Annotations: map[string]string{
			"message": fmt.Sprintf("3Scale API usage is between 80%% and 90%% of the allowable threshold, %d requests per %s, during the last 4 hours", rateLimitRequestsPerUnit, rateLimitUnit),
		},
		Expr: intstr.FromString(fmt.Sprintf("(increase(%s[%dm]) >= (%d / 100 * %d)) and (increase(%s[%dm]) <=  (%d / 100 * %d))",
			totalRequestsMetric, level1ApiUsageCheckFrequencyMins, requestsAllowedOverTimePeriod, level1ApiUsageLowerThresholdPercent, totalRequestsMetric, level1ApiUsageCheckFrequencyMins, requestsAllowedOverTimePeriod, level1ApiUsageHigherThresholdPercent)),
		Labels: map[string]string{"severity": "informational"},
	}, nil
}

func getLevel2ApiUsageAlert(rateLimitUnit string, rateLimitRequestsPerUnit uint32, requestsAllowedPerSecond uint32) (*monitoringv1.Rule, error) {

	requestsAllowedOverTimePeriod := requestsAllowedPerSecond * uint32(level2ApiUsageCheckFrequencyMins) * 60

	return &monitoringv1.Rule{
		Alert: "Level2ThreeScaleApiUsageThresholdExceeded",
		Annotations: map[string]string{
			"message": fmt.Sprintf("3Scale API usage is between 90%% and 95%% of the allowable threshold, %d requests per %s, during the last 2 hours", rateLimitRequestsPerUnit, rateLimitUnit),
		},
		Expr: intstr.FromString(fmt.Sprintf("(increase(%s[%dm]) >= (%d / 100 * %d)) and (increase(%s[%dm]) <=  (%d / 100 * %d))",
			totalRequestsMetric, level2ApiUsageCheckFrequencyMins, requestsAllowedOverTimePeriod, level2ApiUsageLowerThresholdPercent, totalRequestsMetric, level2ApiUsageCheckFrequencyMins, requestsAllowedOverTimePeriod, level2ApiUsageHigherThresholdPercent)),
		Labels: map[string]string{"severity": "informational"},
	}, nil
}

func getLevel3ApiUsageAlert(rateLimitUnit string, rateLimitRequestsPerUnit uint32, requestsAllowedPerSecond uint32) (*monitoringv1.Rule, error) {

	requestsAllowedOverTimePeriod := requestsAllowedPerSecond * uint32(level3ApiUsageCheckFrequencyMins) * 60

	return &monitoringv1.Rule{
		Alert: "Level3ThreeScaleApiUsageThresholdExceeded",
		Annotations: map[string]string{
			"message": fmt.Sprintf("3Scale API usage is above 95%% of the allowable threshold, %d requests per %s, during the last 30 minutes", rateLimitRequestsPerUnit, rateLimitUnit),
		},
		Expr: intstr.FromString(fmt.Sprintf("(increase(%s[%dm]) >= (%d / 100 * %d))",
			totalRequestsMetric, level3ApiUsageCheckFrequencyMins, requestsAllowedOverTimePeriod, level3ApiUsageLowerThresholdPercent)),
		Labels: map[string]string{"severity": "informational"},
	}, nil
}

func getRateLimitInSeconds(rateLimitUnit string, rateLimitRequestsPerUnit uint32) (uint32, error) {
	if rateLimitUnit == "seconds" {
		return rateLimitRequestsPerUnit, nil
	} else if rateLimitUnit == "minute" {
		return rateLimitRequestsPerUnit * 60, nil
	} else if rateLimitUnit == "hour" {
		return rateLimitRequestsPerUnit * 60 * 60, nil
	} else if rateLimitUnit == "day" {
		return rateLimitRequestsPerUnit * 60 * 60 * 24, nil
	} else {
		logrus.Errorf("Unexpected Rate Limit Unit %v, while creating 3scale api usage alerts", rateLimitUnit)
		return 0, errors.New(fmt.Sprintf("Unexpected Rate Limit Unit %v, while creating 3scale api usage alerts", rateLimitUnit))
	}
}
