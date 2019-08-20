package system

import (
	"fmt"
	"log"
	"strings"
	"time"

	"strconv"

	g "github.com/soniah/gosnmp"
)

var walkOid string
var value float64
var pdus []g.SnmpPDU

// ConnectAndExportSNMP : Connects to BigIP, uses yamlParser to read the snmp.yml file and exports data to datadog
func ConnectAndExportSNMP() {

	/*
		params configured to connect to Big-IP
	*/

	params := &g.GoSNMP{
		Target:        "",
		Port:          161,
		Version:       g.Version3,
		Timeout:       time.Duration(100) * time.Second,
		SecurityModel: g.UserSecurityModel,
		MsgFlags:      g.AuthPriv,
		SecurityParameters: &g.UsmSecurityParameters{UserName: "",
			AuthenticationProtocol:   g.SHA,
			AuthenticationPassphrase: "",
			PrivacyProtocol:          g.AES,
			PrivacyPassphrase:        "",
		},
	}

	err := params.Connect()
	if err != nil {
		log.Fatalf("Connect() err: %v", err)
	}
	defer params.Conn.Close()

	/*
		c contains snmp data read from snmp.yml
	*/

	c := Yamlparser()
	for i := 0; i < len(c.BigIP.Metrics); i++ {

		/*
			logic to export non-indexed oid data from big-ip
		*/

		if len(c.BigIP.Metrics[i].Indexes) == 0 {
			walkOid = c.BigIP.Metrics[i].Oid
			structSNMPerr := params.BulkWalk(walkOid, exportDatanotIndexed)
			if structSNMPerr != nil {
				log.Fatalf("Get() struct err: %v", structSNMPerr)
			}
		}

	}

	/*
	 triggers logic to export indexed oid data from big-ip

	*/

	walkAll := c.Walk

	result := []g.SnmpPDU{}

	for _, subtree := range walkAll {
		result, err = params.BulkWalkAll(subtree)
	}

	pdus = append(pdus, result...)
	//fmt.Println("pdu", pdus)
	performLookup(c.BigIP.Metrics)
	test := buildMetricTree((c.BigIP.Metrics))

	for _, o := range len(test) {
		fmt.Println(test.children)
	}
}

/*
	Functions to help scrape indexed data
*/

// takes in the entire metric struct to create a metric tree, performs key value lookup for indexes

func performLookup(metrics []*Metric) {

	metricTree := buildMetricTree(metrics)
	oidToPdu := make(map[string]g.SnmpPDU, len(pdus))
PduLoop:
	for oid, pdu := range oidToPdu {
		head := metricTree
		oidList := oidToList(oid)
		for i, o := range oidList {
			var ok bool
			head, ok = head.children[o]
			if !ok {
				continue PduLoop
			}
			if head.metric != nil {
				// Found a match.
				pduToSamples(oidList[i+1:], pdu, head.metric, oidToPdu)
				break
			}
		}
	}
}

// Build a tree of metrics from the config, for fast lookup when there's lots of them.
func buildMetricTree(metrics []*Metric) *MetricNode {
	metricTree := &MetricNode{children: map[int]*MetricNode{}}
	for _, metric := range metrics {
		head := metricTree
		for _, o := range oidToList(metric.Oid) {
			_, ok := head.children[o]
			if !ok {
				head.children[o] = &MetricNode{children: map[int]*MetricNode{}}
			}
			head = head.children[o]
		}
		head.metric = metric
	}
	return metricTree
}

func getPduValue(pdu g.SnmpPDU) float64 {
	switch pdu.Type {
	case g.Counter64:
		return float64(g.ToBigInt(pdu.Value).Uint64())
	case g.OpaqueFloat:
		return float64(pdu.Value.(float32))
	case g.OpaqueDouble:
		return pdu.Value.(float64)
	default:
		return float64(g.ToBigInt(pdu.Value).Int64())
	}
}

// function to process indexed data

func pduToSamples(indexOids []int, pdu g.SnmpPDU, metric *Metric, oidToPdu map[string]g.SnmpPDU) {

	// The part of the OID that is the indexes.
	labels := indexesToLabels(indexOids, metric, oidToPdu)

	//value := getPduValue(pdu)

	labelnames := make([]string, 0, len(labels)+1)
	labelvalues := make([]string, 0, len(labels)+1)
	//look out
	for k, v := range labels {
		labelnames = append(labelnames, k)
		labelvalues = append(labelvalues, v)
	}

	switch metric.Type {
	case "counter":

	case "gauge":

	case "Float", "Double":

	case "DateAndTime":

	default:
		// It's some form of string.

		value = 1.0
		metricType := metric.Type
		if _, ok := labels[metric.Name]; !ok {
			labelnames = append(labelnames, metric.Name)
			labelvalues = append(labelvalues, pduValueAsString(pdu, metricType))
		}
	}
}

func indexesToLabels(indexOids []int, metric *Metric, oidToPdu map[string]g.SnmpPDU) map[string]string {
	labels := map[string]string{}
	labelOids := map[string][]int{}

	// Covert indexes to useful strings.
	for _, index := range metric.Indexes {
		str, subOid, remainingOids := indexOidsAsString(indexOids, index.Type, index.FixedSize, index.Implied)
		// The labelvalue is the text form of the index oids.
		labels[index.Labelname] = str
		// Save its oid in case we need it for lookups.
		labelOids[index.Labelname] = subOid
		// For the next iteration.
		indexOids = remainingOids
	}

	// Perform lookups.
	for _, lookup := range metric.Lookups {
		if len(lookup.Labels) == 0 {
			delete(labels, lookup.Labelname)
			continue
		}
		oid := lookup.Oid
		for _, label := range lookup.Labels {
			oid = fmt.Sprintf("%s.%s", oid, listToOid(labelOids[label]))
		}
		if pdu, ok := oidToPdu[oid]; ok {
			labels[lookup.Labelname] = pduValueAsString(pdu, lookup.Type)
		} else {
			labels[lookup.Labelname] = ""
		}
	}

	return labels
}

// Convert oids to a string index value.
//
// Returns the string, the oids that were used and the oids left over.
func indexOidsAsString(indexOids []int, typ string, fixedSize int, implied bool) (string, []int, []int) {

	switch typ {
	case "Integer32", "Integer", "gauge", "counter":
		// Extract the oid for this index, and keep the remainder for the next index.
		subOid, indexOids := splitOid(indexOids, 1)
		return fmt.Sprintf("%d", subOid[0]), subOid, indexOids
	case "OctetString":
		var subOid []int
		// The length of fixed size indexes come from the MIB.
		// For varying size, we read it from the first oid.
		length := fixedSize
		if implied {
			length = len(indexOids)
		}
		if length == 0 {
			subOid, indexOids = splitOid(indexOids, 1)
			length = subOid[0]
		}
		content, indexOids := splitOid(indexOids, length)
		subOid = append(subOid, content...)
		parts := make([]byte, length)
		for i, o := range content {
			parts[i] = byte(o)
		}
		if len(parts) == 0 {
			return "", subOid, indexOids
		} else {
			return fmt.Sprintf("0x%X", string(parts)), subOid, indexOids
		}
	case "DisplayString":
		var subOid []int
		length := fixedSize
		if implied {
			length = len(indexOids)
		}
		if length == 0 {
			subOid, indexOids = splitOid(indexOids, 1)
			length = subOid[0]
		}
		content, indexOids := splitOid(indexOids, length)
		subOid = append(subOid, content...)
		parts := make([]byte, length)
		for i, o := range content {
			parts[i] = byte(o)
		}
		// ASCII, so can convert staight to utf-8.
		return string(parts), subOid, indexOids
	default:
		log.Fatalf("Unknown index type %s", typ)
		return "", nil, nil
	}
}

// Converting string of oid into array of ints

func oidToList(oid string) []int {
	result := []int{}
	for _, x := range strings.Split(oid, ".") {
		o, _ := strconv.Atoi(x)
		result = append(result, o)
	}
	return result
}

//converting array of ints into a string of oid
func listToOid(l []int) string {
	var result []string
	for _, o := range l {
		result = append(result, strconv.Itoa(o))
	}
	return strings.Join(result, ".")
}

// Right pad oid with zeros, and split at the given point.
// Some routers exclude trailing 0s in responses.
func splitOid(oid []int, count int) ([]int, []int) {
	head := make([]int, count)
	tail := []int{}
	for i, v := range oid {
		if i < count {
			head[i] = v
		} else {
			tail = append(tail, v)
		}
	}
	return head, tail
}

// This mirrors decodeValue in gosnmp's helper.go.
func pduValueAsString(pdu g.SnmpPDU, typ string) string {
	switch pdu.Value.(type) {
	case int:
		return strconv.Itoa(pdu.Value.(int))
	case uint:
		return strconv.FormatUint(uint64(pdu.Value.(uint)), 10)
	case uint64:
		return strconv.FormatUint(pdu.Value.(uint64), 10)
	case float32:
		return strconv.FormatFloat(float64(pdu.Value.(float32)), 'f', -1, 32)
	case float64:
		return strconv.FormatFloat(pdu.Value.(float64), 'f', -1, 64)
	case string:
		if pdu.Type == g.ObjectIdentifier {
			// Trim leading period.
			return pdu.Value.(string)[1:]
		}
		// DisplayString.
		return pdu.Value.(string)
	case []byte:
		return ""
	case nil:
		return ""
	default:
		// This shouldn't happen.
		log.Fatalf("Got PDU with unexpected type: Name: %s Value: '%s', Go Type: %T SNMP Type: %v", pdu.Name, pdu.Value, pdu.Value, pdu.Type)
		return fmt.Sprintf("%s", pdu.Value)
	}
}

/*
	Functions that helps process non-indexed data
*/

//exporting non-indexed data to datadog
func exportDatanotIndexed(pdu g.SnmpPDU) error {
	d := lookupOidnotIndexed(pdu)
	switch pdu.Type {
	case g.OctetString:
		b := pdu.Value.([]byte)
		fmt.Printf("STRING: %s\n", string(b))
	case g.TimeTicks:
		d.Value = unixToTime(pdu)
	default:
		b := g.ToBigInt(pdu.Value)
		bigstr := b.String()
		f, _ := strconv.ParseFloat(bigstr, 64)
		d.Value = f
	}
	//	Datadog(d)
	return nil
}

/*
   uses pdu.Oid and cross references to find Metric
*/

func lookupOidnotIndexed(pdu g.SnmpPDU) DatadogStruct {
	c := Yamlparser()
	m := c.Metrics
	var d DatadogStruct
	for i := 0; i < len(m); i++ {
		metricName := m[i].Name
		metricOid := m[i].Oid
		if walkOid == metricOid {
			d.Oid = metricOid
			d.Name = metricName
			d.Type = m[i].Type
			d.Help = m[i].Help
		}
	}
	return d
}

// converts unix time to regular seconds
func unixToTime(pdu g.SnmpPDU) float64 {
	timeNow := uint(time.Now().Unix())
	pduInt := pdu.Value.(uint)
	u := timeNow - pduInt
	uptime := float64(u)
	return uptime / 6000
}

/*
	For testing purposed functions
*/

func printValue(pdu g.SnmpPDU) error {
	fmt.Printf("%s = ", pdu.Name)
	switch pdu.Type {
	case g.OctetString:
		b := pdu.Value.([]byte)
		fmt.Printf("STRING: %s\n", string(b))
	default:
		fmt.Printf("TYPE %d: %d\n", pdu.Type, g.ToBigInt(pdu.Value))
	}
	return nil
}

// deletes the first '.' in each oid
func trimFirstRune(s string) string {
	for i := range s {
		if i > 0 {
			// The value i is the index in s of the second
			// rune.  Slice to remove the first rune.
			return s[i:]
		}
	}
	// There are 0 or 1 runes in the string.
	return ""
}
