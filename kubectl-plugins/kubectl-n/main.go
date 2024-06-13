/*
A kubectl plugin to implement the 'kubectl n' command when placed in your PATH.

This is like 'kubectl get nodes' but shows some additional columns.
It also summarises any warnings/errors for the nodes at the end of the list.
*/
package main

import (
	"cmp"
	"fmt"
	"slices"
	"strings"

	"github.com/jim-barber-he/go/k8s"
	"github.com/jim-barber-he/go/texttable"
	"github.com/jim-barber-he/go/util"

	flag "github.com/spf13/pflag"
	v1 "k8s.io/api/core/v1"
)

const tick = "\u2713"

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

// Implement the texttab.tableFormatter interface.
func (tr *tableRow) TabTitleRow() string {
	return texttable.ReflectedTitleRow(tr)
}

// Implement the texttab.tableFormatter interface.
func (tr *tableRow) TabValues() string {
	return texttable.ReflectedTabValues(tr)
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

	var tbl texttable.Table[*tableRow]
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

		row.Age = util.FormatAge(node.CreationTimestamp.Time)
		row.Version = node.Status.NodeInfo.KubeletVersion
		row.Runtime = util.LastSplitItem(node.Status.NodeInfo.ContainerRuntimeVersion, "/")

		// Additional columns for AWS EC2 instances are from this point on.

		row.Type = node.Labels["node.kubernetes.io/instance-type"]

		if node.Labels["node-role.kubernetes.io/spot-worker"] != "" {
			row.Spot = tick
		} else {
			row.Spot = "x"
		}

		row.AZ = util.LastSplitItem(node.Labels["topology.kubernetes.io/zone"], "")

		// The external AWS controller manager sets the node names to the Instance ID,
		// while the old AWS code in k8s sets it to the DNS name that contains the IP address.
		// Depending on which one is used will determine if the InstanceID or IP value is set.
		if strings.HasPrefix(node.Name, "ip-") {
			row.InstanceID = util.LastSplitItem(node.Spec.ProviderID, "/")
		} else {
			row.IP = node.Annotations["alpha.kubernetes.io/provided-node-ip"]
		}

		// Handle getting a node group for both EKS and kOps.
		row.InstanceGroup = cmp.Or(
			node.Labels["kops.k8s.io/instancegroup"],
			node.Labels["eks.amazonaws.com/nodegroup"],
		)

		tbl.Append(&row)
	}

	// Sort function to sort the rows slice by InstanceGroup, then AZ, then Name when iterating through it.
	slices.SortFunc(tbl.Rows, func(a, b *tableRow) int {
		return cmp.Or(
			cmp.Compare(a.InstanceGroup, b.InstanceGroup),
			cmp.Compare(a.AZ, b.AZ),
			cmp.Compare(a.Name, b.Name),
		)
	})

	// Display the table.
	tbl.Write()

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
