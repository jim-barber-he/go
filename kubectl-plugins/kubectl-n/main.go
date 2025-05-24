/*
A kubectl plugin to implement the 'kubectl n' command when placed in your PATH.

This is like 'kubectl get nodes' but shows some additional columns.
It also summarises any warnings/errors for the nodes at the end of the list.
*/
package main

import (
	"cmp"
	"fmt"
	"log"
	"slices"
	"strings"

	"github.com/jim-barber-he/go/k8s"
	"github.com/jim-barber-he/go/texttable"
	"github.com/jim-barber-he/go/util"
	flag "github.com/spf13/pflag"
	v1 "k8s.io/api/core/v1"
)

const tick = "\u2713"

var goodStatuses = map[v1.NodeConditionType]v1.ConditionStatus{
	"ContainerRuntimeUnhealthy":   "False",
	"CorruptDockerOverlay2":       "False",
	"CPUPressure":                 "False",
	"DiskPressure":                "False",
	"FrequentContainerdRestart":   "False",
	"FrequentDockerRestart":       "False",
	"FrequentKubeletRestart":      "False",
	"FrequentUnregisterNetDevice": "False",
	"KernelDeadlock":              "False",
	"KubeletUnhealthy":            "False",
	"MemoryPressure":              "False",
	"NetworkUnavailable":          "False",
	"OutOfDisk":                   "False",
	"PIDPressure":                 "False",
	"ReadonlyFilesystem":          "False",
	"Ready":                       "True",
}

// tableRow represents a row in the output table.
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
	KernelVersion string `title:"KERNEL-VERSION,omitempty"`
	OSImage       string `title:"OS-IMAGE,omitempty"`
	Architecture  string `title:"ARCH,omitempty"`
}

func main() {
	kubeContext := flag.String("context", "", "The name of the kubeconfig context to use")
	wide := flag.BoolP("wide", "w", false, "Add KernelVersion, OSImage, and Architecture columns")
	flag.Parse()

	clientset := k8s.Client(*kubeContext)

	nodes, err := k8s.ListNodes(clientset)
	if err != nil {
		log.Fatalf("Error listing nodes: %v", err)
	}
	if len(nodes.Items) == 0 {
		log.Fatal("No nodes found")
	}

	var tbl texttable.Table[*tableRow]
	warnings := make(map[string][]string)

	for _, node := range nodes.Items {
		row := createTableRow(&node, *wide)

		// Keep track of any warning messages for the node and a status to reflect if there are problems.
		status, messages := getNodeStatus(node.Status.Conditions)
		warnings[node.Name] = messages
		if node.Spec.Unschedulable {
			status += " *"
			warnings[node.Name] = append(warnings[node.Name], "Scheduling Disabled")
		}
		row.Ok = status

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
	printWarnings(warnings)
}

// createTableRow creates a tableRow struct from a v1.Node struct.
func createTableRow(node *v1.Node, wide bool) tableRow {
	var row tableRow

	// Just keep the hostname and strip off any domain name.
	row.Name = strings.Split(node.Name, ".")[0]

	row.Age = util.FormatAge(node.CreationTimestamp.Time)
	row.Version = node.Status.NodeInfo.KubeletVersion
	if wide {
		row.Runtime = node.Status.NodeInfo.ContainerRuntimeVersion
		row.KernelVersion = node.Status.NodeInfo.KernelVersion
		row.OSImage = node.Status.NodeInfo.OSImage
		row.Architecture = node.Status.NodeInfo.Architecture
	} else {
		row.Runtime = util.LastSplitItem(node.Status.NodeInfo.ContainerRuntimeVersion, "/")
	}

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

	return row
}

// getNodeStatus looks at the conditions of a node and returns the node's status and any associated warning messages.
func getNodeStatus(conditions []v1.NodeCondition) (string, []string) {
	var messages []string

	status := tick

	for _, condition := range conditions {
		expectedStatus, ok := goodStatuses[condition.Type]
		if !ok {
			log.Printf(
				"Warning, we haven't covered all conditions - Please add %s to goodStatuses",
				condition.Type,
			)
			continue
		}
		if condition.Status != expectedStatus {
			messages = append(messages, fmt.Sprintf(
				"Node condition %s is now: %s, message: \"%s\"",
				condition.Type, condition.Status, condition.Message,
			))
		}
	}

	if len(messages) > 0 {
		status = "x"
	}

	return status, messages
}

// printWarnings displays any warning messages that were collected for the nodes.
func printWarnings(warnings map[string][]string) {
	for nodeName, messages := range warnings {
		if len(messages) > 0 {
			fmt.Println()
			for _, message := range messages {
				fmt.Printf("%s: %s\n", nodeName, message)
			}
		}
	}
}
