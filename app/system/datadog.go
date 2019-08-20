package system

import (
	"fmt"
	"time"

	d "github.com/zorkian/go-datadog-api"
)

//Datadog takes in a datadog struct named data, creates a metric to post to datadog endpoint
func Datadog(data DatadogStruct) {

	/*
	   Datadog authentication
	   apikey and appkey are exported from config.yml, accesses gitlab secret variables to connect to datadog
	*/

	apiKey := ""
	appKey := ""
	client := d.NewClient(apiKey, appKey)
	client.SetBaseUrl("https://nordstrom.datadoghq.com")
	metricValue := data.Value
	host := "bip0319t10.nordstrom.net"
	metric := "ltm"
	metric = metric + string(".") + string(data.Name)
	timeNow := float64(time.Now().Unix())

	/*
		postData contains the datapoint along with host name and tags associated with the datapoint
	*/

	postdata := d.Metric{
		Metric: &metric,
		Points: []d.DataPoint{
			d.DataPoint{&timeNow, &metricValue},
		},
		Type: &data.Type,
		Host: &host,
		Tags: []string{"f5_env:nightwing", "BIG_IP:testing", "from:intern"},
	}

	/*
		liftoff! Posting SNMP metric data to datadog.
	*/

	err := client.PostMetrics([]d.Metric{postdata})
	if err != nil {
		panic(fmt.Sprintf("[Error]: %s", err))
	}
}
