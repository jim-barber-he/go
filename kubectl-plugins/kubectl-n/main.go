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
	tableFlags    = 0
	tableMinWidth = 0
	tablePadChar  = ' '
	tablePadding  = 2
	tableTabWidth = 8
	tick          = "\u2713"
)

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

// Returns a table row with the field values populated via the struct tag called 'title'.
// If the table row item is unset and the title tag is set to 'omitempty' then do not include it.
func (r *tableRow) getTitleRow() (row tableRow) {
	// The following can also be written as follows but which is better?
	//   v := reflect.ValueOf(r).Elem()
	v := reflect.ValueOf(*r)
	for i, sf := range reflect.VisibleFields(v.Type()) {
		titleArray := strings.Split(sf.Tag.Get("title"), ",")
		if len(titleArray) > 1 && titleArray[1] == "omitempty" && v.Field(i).String() == "" {
			continue
		}
		reflect.ValueOf(&row).Elem().Field(i).SetString(titleArray[0])
	}

	return row
}

// Output the values of the tableRow struct separated by tabs. Empty fields are ignored.
func (r *tableRow) tabValues() string {
	var s []string
	v := reflect.ValueOf(*r)
	for i := 0; i < v.NumField(); i++ {
		if str := strings.TrimSpace(v.Field(i).String()); str != "" {
			s = append(s, str)
		}
	}
	return strings.Join(s, "\t")
}

type table struct {
	rows []tableRow
}

func (t *table) append(r tableRow) {
	t.rows = append(t.rows, r)
}

func (t *table) write() {
	// Sort function to sort the rows slice by InstanceGroup, then AZ, then Name when iterating through it.
	slices.SortFunc(t.rows, func(a, b tableRow) int {
		return cmp.Or(
			cmp.Compare(a.InstanceGroup, b.InstanceGroup),
			cmp.Compare(a.AZ, b.AZ),
			cmp.Compare(a.Name, b.Name),
		)
	})

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
	// Bail if no nodes were found.
	if len(nodes.Items) == 0 {
		panic("No nodes found")
	}

	var tbl table
	warnings := make(map[string][]string)
	for _, node := range nodes.Items {
		var r tableRow
		// Just keep the hostname and strip off any domain name.
		r.Name = strings.Split(node.Name, ".")[0]

		// Handle any conditions for the node to determine if it is healthy.
		// Keep track of any warning messages for the node.
		status, messages := getNodeStatus(node.Status.Conditions)
		warnings[node.Name] = messages
		if node.Spec.Unschedulable {
			status += " *"
			warnings[node.Name] = append(warnings[node.Name], "Scheduling Disabled")
		}
		r.Ok = status

		r.Age = formatAge(node.CreationTimestamp.Time)
		r.Version = node.Status.NodeInfo.KubeletVersion

		runtime := strings.Split(node.Status.NodeInfo.ContainerRuntimeVersion, "/")
		r.Runtime = runtime[len(runtime)-1]

		// Additional columns for AWS EC2 instances are from this point on.

		r.Type = node.Labels["node.kubernetes.io/instance-type"]

		if node.Labels["node-role.kubernetes.io/spot-worker"] != "" {
			r.Spot = tick
		} else {
			r.Spot = "x"
		}

		az := strings.Split(node.Labels["topology.kubernetes.io/zone"], "")
		if len(az) > 0 {
			r.AZ = az[len(az)-1]
		}

		// The external AWS controller manager sets the node names to the Instance ID,
		// while the old AWS code in k8s sets it to the DNS name that contains the IP address.
		// Depending on which one is used will determine if the InstanceID or IP value is set.
		if strings.HasPrefix(node.Name, "ip-") {
			providerID := strings.Split(node.Spec.ProviderID, "/")
			if len(providerID) > 0 {
				r.InstanceID = providerID[len(providerID)-1]
			}
		} else {
			r.IP = node.Annotations["alpha.kubernetes.io/provided-node-ip"]
		}

		// Handle getting a node group for both EKS and kOps.
		r.InstanceGroup = cmp.Or(
			node.Labels["kops.k8s.io/instancegroup"],
			node.Labels["eks.amazonaws.com/nodegroup"],
		)

		tbl.append(r)
	}

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
func formatAge(timestamp time.Time) string {
	var weeks, days, hours, minutes, seconds int

	duration := time.Since(timestamp).Round(time.Second)

	seconds = int(duration.Seconds())

	weeks = seconds / (60 * 60 * 24 * 7)
	seconds -= weeks * 60 * 60 * 24 * 7

	days = seconds / (60 * 60 * 24)
	seconds -= days * 60 * 60 * 24

	hours = seconds / (60 * 60)
	seconds -= hours * 60 * 60

	minutes = seconds / 60
	seconds -= minutes * 60

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
	goodStatuses := map[v1.NodeConditionType]v1.ConditionStatus{
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

	var messages []string
	status := tick

	for _, condition := range conditions {
		if _, ok := goodStatuses[condition.Type]; !ok {
			fmt.Printf("Warning, we haven't covered all conditions - Please add %s to goodStatuses", condition.Type)
		}
		if condition.Status != goodStatuses[condition.Type] {
			messages = append(messages, fmt.Sprintf("%s=%s", condition.Type, condition.Message))
		}
		if len(messages) > 0 {
			status = "x"
		}
	}

	return status, messages
}
