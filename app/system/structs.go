package system

/*

Contains all the structs required to process data coming from f5 devices
Also contains struct that is used to post metrics to datadog

*/

// Config to export gitlab secrets
type Config struct {
	Datadog DatadogConfig `yaml:"datadog"`
	Bigip   BigIPConfig   `yaml:"bigip"`
}

// DatadogConfig to configure datadog
type DatadogConfig struct {
	Datadogappkey string `yaml:"datadogappkey"`
	Datadogapikey string `yaml:"datadogapikey"`
}

// BigIPConfig to configure gosnmp
type BigIPConfig struct {
	Username                 string `yaml:"username"`
	AuthenticationPassphrase string `yaml:"authenticationpassphrase"`
	PrivacyPassphrase        string `yaml:"privacypassphrase"`
}

// DatadogStruct contains data to be create the Metrics to be posted
type DatadogStruct struct {
	Oid   string
	Name  string
	Value float64
	Help  string
	Type  string
}

//MetricNode to create metric tree
type MetricNode struct {
	metric   *Metric
	children map[int]*MetricNode
}

// SNMPyaml is the highest level struct containing all data from snmp.yml
type SNMPyaml struct {
	BigIP `yaml:"f5bigipsmall"`
}

//BigIP contains Walk and Get oids, and following that metric values
type BigIP struct {
	Walk    `yaml:"walk,omitempty"`
	Get     `yaml:"get,omitempty"`
	Metrics []*Metric `yaml:"metrics"`
}

//Walk contains an array of oid strings to walk
type Walk []string

//Get contains an array of oid strings to get specific oid data values
type Get []string

//Metric contain all data pertaining to specific oid
type Metric struct {
	Name    string    `yaml:"name"`
	Oid     string    `yaml:"oid"`
	Type    string    `yaml:"type"`
	Help    string    `yaml:"help"`
	Indexes []*Index  `yaml:"indexes,omitempty"`
	Lookups []*Lookup `yaml:"lookups,omitempty"`
}

//Index contain labelname and the oid data type
type Index struct {
	Labelname string `yaml:"labelname"`
	Type      string `yaml:"type"`
	FixedSize int    `yaml:"fixed_size,omitempty"`
	Implied   bool   `yaml:"implied,omitempty"`
}

//Lookup struct helps scrape data from indexed oids
type Lookup struct {
	Labels    []string `yaml:"labels"`
	Labelname string   `yaml:"labelname"`
	Oid       string   `yaml:"oid,omitempty"`
	Type      string   `yaml:"type,omitempty"`
}
