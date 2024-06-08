/*
A kubectl plugin to implement the 'kubectl n' command when placed in your PATH.

This is like 'kubectl get nodes' but shows some additional columns.
It also summarises any warnings/errors for the nodes at the end of the list.
*/
package main

import (
	"cmp"
	"fmt"
	"os"
	"reflect"
	"slices"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/jim-barber-he/go/k8s"
	flag "github.com/spf13/pflag"
	v1 "k8s.io/api/core/v1"
)

const (
	numSecondsPerWeek   = 60 * 60 * 24 * 7
	numSecondsPerDay    = 60 * 60 * 24
	numSecondsPerHour   = 60 * 60
	numSecondsPerMinute = 60

	tableFlags    = 0
	tableMinWidth = 0
	tablePadChar  = ' '
	tablePadding  = 2
	tableTabWidth = 8

	tick = "\u2713"
)

// Is there a better way than defining this as a global var?
var goodStatuses = map[v1.NodeConditionType]v1.ConditionStatus{
	"CorruptDockerOverlay2": "False",
	"DiskPressure":          "False",
	"KernelDeadlock":        "False",
	"MemoryPressure":        "False",
	"NetworkUnavailable":    "False",
	"OutOfDisk":             "False",
	"PIDPressure":           "False",
	"ReadonlyFilesystem":    "False",
	"Ready":                 "True",
}

// Interface that a table row struct needs to implement for the table.write() method to be able to use it.
// Both of these methods need to return a string containing tab separated row values for the tabwriter module to use.
type tableFormatter interface {
	getTitleRow() tableFormatter
	tabValues() string
}

type tableRow struct {
	Name          string `title:"NAME"`
	Ok            string `title:"OK"`
	Age           string `title:"AGE"`
	Version       string `title:"VERSION"`
	Runtime       string `title:"RUNTIME"`
	Type          string `title:"TYPE,omitempty"`
	Spot          string `title:"SPOT,omitempty"`
	AZ            string `title:"AZ,omitempty"`
	InstanceID    string `title:"INSTANCE-ID,omitempty"`
	IP            string `title:"IP-ADDRESS,omitempty"`
	InstanceGroup string `title:"INSTANCE-GROUP,omitempty"`
}

// Returns a new table row with the field values populated via the struct tag called 'title'.
// If the table row field is unset and the title tag is set to 'omitempty' then do not include it.
func (r *tableRow) getTitleRow() tableFormatter {
	var row tableRow
	rowElem := reflect.ValueOf(&row).Elem()

	v := reflect.ValueOf(*r)
	for i, sf := range reflect.VisibleFields(v.Type()) {
		titleArray := strings.Split(sf.Tag.Get("title"), ",")
		if len(titleArray) > 1 && titleArray[1] == "omitempty" && v.Field(i).String() == "" {
			continue
		}
		rowElem.Field(i).SetString(titleArray[0])
	}
	return &row
}

// Output the field values of the tableRow struct separated by tabs. Empty fields are ignored.
func (r *tableRow) tabValues() string {
	var s []string
	v := reflect.ValueOf(*r)
	for i := range v.NumField() {
		if str := strings.TrimSpace(v.Field(i).String()); str != "" {
			s = append(s, str)
		}
	}
	return strings.Join(s, "\t")
}

type table[R tableFormatter] struct {
	rows []R
}

func (t *table[R]) append(r R) {
	t.rows = append(t.rows, r)
}

func (t *table[R]) write() {
	tw := tabwriter.NewWriter(os.Stdout, tableMinWidth, tableTabWidth, tablePadding, tablePadChar, tableFlags)
	titles := t.rows[0].getTitleRow()
	fmt.Fprintln(tw, titles.tabValues())
	for _, row := range t.rows {
		fmt.Fprintln(tw, row.tabValues())
	}
	tw.Flush()
}

func main() {
	var kubeContext string
	flag.StringVar(&kubeContext, "context", "", "The name of the kubeconfig context to use")
	flag.Parse()

	clientset := k8s.Client(kubeContext)

	nodes, err := k8s.ListNodes(clientset)
	if err != nil {
		panic(err.Error())
	}
	if len(nodes.Items) == 0 {
		panic("No nodes found")
	}

	var tbl table[*tableRow]
	warnings := make(map[string][]string)
	for _, node := range nodes.Items {
		var row tableRow
		// Just keep the hostname and strip off any domain name.
		row.Name = strings.Split(node.Name, ".")[0]

		// Keep track of any warning messages for the node and a status to reflect if there are problems.
		status, messages := getNodeStatus(node.Status.Conditions)
		warnings[node.Name] = messages
		if node.Spec.Unschedulable {
			status += " *"
			warnings[node.Name] = append(warnings[node.Name], "Scheduling Disabled")
		}
		row.Ok = status

		row.Age = formatAge(node.CreationTimestamp.Time)
		row.Version = node.Status.NodeInfo.KubeletVersion
		row.Runtime = lastSplitItem(node.Status.NodeInfo.ContainerRuntimeVersion, "/")

		// Additional columns for AWS EC2 instances are from this point on.

		row.Type = node.Labels["node.kubernetes.io/instance-type"]

		if node.Labels["node-role.kubernetes.io/spot-worker"] != "" {
			row.Spot = tick
		} else {
			row.Spot = "x"
		}

		row.AZ = lastSplitItem(node.Labels["topology.kubernetes.io/zone"], "")

		// The external AWS controller manager sets the node names to the Instance ID,
		// while the old AWS code in k8s sets it to the DNS name that contains the IP address.
		// Depending on which one is used will determine if the InstanceID or IP value is set.
		if strings.HasPrefix(node.Name, "ip-") {
			row.InstanceID = lastSplitItem(node.Spec.ProviderID, "/")
		} else {
			row.IP = node.Annotations["alpha.kubernetes.io/provided-node-ip"]
		}

		// Handle getting a node group for both EKS and kOps.
		row.InstanceGroup = cmp.Or(
			node.Labels["kops.k8s.io/instancegroup"],
			node.Labels["eks.amazonaws.com/nodegroup"],
		)

		tbl.append(&row)
	}

	// Sort function to sort the rows slice by InstanceGroup, then AZ, then Name when iterating through it.
	slices.SortFunc(tbl.rows, func(a, b *tableRow) int {
		return cmp.Or(
			cmp.Compare(a.InstanceGroup, b.InstanceGroup),
			cmp.Compare(a.AZ, b.AZ),
			cmp.Compare(a.Name, b.Name),
		)
	})

	// Display the table.
	tbl.write()

	// Display any warning messages for the nodes.
	for nodeName, warnArray := range warnings {
		if len(warnArray) > 0 {
			fmt.Println()
			for _, message := range warnArray {
				fmt.Printf("%s: %s\n", nodeName, message)
			}
		}
	}
}

// Return the age in a human readable format of the first 2 non-zero time units from weeks to seconds,
// or just the seconds if no higher time unit was above 0.
// This differs from duration.String() in that it also handles weeks and days.
func formatAge(timestamp time.Time) string {
	var weeks, days, hours, minutes, seconds int

	duration := time.Since(timestamp).Round(time.Second)

	seconds = int(duration.Seconds())

	weeks = seconds / numSecondsPerWeek
	seconds -= weeks * numSecondsPerWeek

	days = seconds / numSecondsPerDay
	seconds -= days * numSecondsPerDay

	hours = seconds / numSecondsPerHour
	seconds -= hours * numSecondsPerHour

	minutes = seconds / numSecondsPerMinute
	seconds -= minutes * numSecondsPerMinute

	var dateStr string
	// When set to true, return as soon as the next non-zero time unit is set.
	var retNext bool

	if weeks > 0 {
		dateStr = fmt.Sprintf("%dw", weeks)
		retNext = true
	}
	if days > 0 {
		dateStr = fmt.Sprintf("%s%dd", dateStr, days)
		if retNext {
			return dateStr
		}
		retNext = true
	}
	if hours > 0 {
		dateStr = fmt.Sprintf("%s%dh", dateStr, hours)
		if retNext {
			return dateStr
		}
		retNext = true
	}
	if minutes > 0 {
		dateStr = fmt.Sprintf("%s%dm", dateStr, minutes)
		if retNext {
			return dateStr
		}
	}
	return fmt.Sprintf("%s%ds", dateStr, seconds)
}

// Looks at the conditions of a node and returns the node's status and any warning messages associated with it.
func getNodeStatus(conditions []v1.NodeCondition) (string, []string) {
	var messages []string

	status := tick
	for _, condition := range conditions {
		if _, ok := goodStatuses[condition.Type]; !ok {
			fmt.Printf("Warning, we haven't covered all conditions - Please add %s to goodStatuses", condition.Type)
			continue
		}
		if condition.Status != goodStatuses[condition.Type] {
			messages = append(messages, fmt.Sprintf("%s=%s", condition.Type, condition.Message))
		}
	}
	if len(messages) > 0 {
		status = "x"
	}
	return status, messages
}

// lastSplitItem() splits a string into a slice based on a split character and returns the last item.
func lastSplitItem(str, splitChar string) string {
	result := strings.Split(str, splitChar)
	if len(result) > 0 {
		return result[len(result)-1]
	}
	return ""
}
