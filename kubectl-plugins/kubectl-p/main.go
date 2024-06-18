/*
A kubectl plugin to implement the 'kubectl p' command when placed in your PATH.

This is a bit like 'kubectl get pods -o wide' but shows columns with more detail on the node and AWS availability zone.
*/
package main

import (
	"cmp"
	"fmt"
	"os"
	"slices"
	"strconv"

	"github.com/jim-barber-he/go/k8s"
	"github.com/jim-barber-he/go/texttable"
	"github.com/jim-barber-he/go/util"

	flag "github.com/spf13/pflag"
	v1 "k8s.io/api/core/v1"
)

const tick = "\u2713"

type tableRow struct {
	Namespace string `title:"NAMESPACE,omitempty"`
	Name      string `title:"NAME"`
	Ready     string `title:"READY"`
	Status    string `title:"STATUS"`
	Restarts  string `title:"RESTARTS"`
	Age       string `title:"AGE"`
	IP        string `title:"IP"`
	Node      string `title:"NODE"`
	Spot      string `title:"SPOT"`
	AZ        string `title:"AZ,omitempty"`
}

// Implement the texttab.TableFormatter interface.
func (tr *tableRow) TabTitleRow() string {
	return texttable.ReflectedTitleRow(tr)
}

// Implement the texttab.TableFormatter interface.
func (tr *tableRow) TabValues() string {
	return texttable.ReflectedTabValues(tr)
}

func main() {
	var allNamespaces bool
	var kubeContext string
	var labelSelector string

	flag.BoolVarP(&allNamespaces, "all-namespaces", "A", false, "List the pods across all namespaces")
	flag.StringVar(&kubeContext, "context", "", "The name of the kubeconfig context to use")
	flag.StringVarP(&labelSelector, "selector", "l", "", "Selector (label query) to filter on")

	flag.Parse()

	clientset := k8s.Client(kubeContext)

	namespace := ""
	if !allNamespaces {
		namespace = k8s.Namespace(kubeContext)
	}

	pods, err := k8s.ListPods(clientset, namespace, labelSelector)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
	if len(pods.Items) == 0 {
		fmt.Fprintln(os.Stderr, "No pods found")
		os.Exit(1)
	}

	var tbl texttable.Table[*tableRow]
	nodes := make(map[string]*v1.Node)
	for _, pod := range pods.Items {
		var row tableRow

		// Try to find out what node the pod is on and get its details if we haven't already.
		node := pod.Spec.NodeName
		if node != "" {
			if _, ok := nodes[node]; !ok {
				nodePtr, err := k8s.GetNode(clientset, node)
				if err == nil {
					nodes[node] = nodePtr
				}
			}
		}
		numContainers := len(pod.Status.ContainerStatuses)
		numReady := 0
		restarts := 0
		for _, containerStatus := range pod.Status.ContainerStatuses {
			if containerStatus.Ready {
				numReady++
			}
			if containerStatus.RestartCount > 0 {
				restarts += int(containerStatus.RestartCount)
			}
		}

		if allNamespaces {
			row.Namespace = pod.Namespace
		}
		row.Name = pod.Name
		row.Ready = fmt.Sprintf("%d/%d", numReady, numContainers)
		row.Status = string(pod.Status.Phase)
		row.Restarts = strconv.Itoa(restarts)
		row.Age = util.FormatAge(pod.CreationTimestamp.Time)
		row.IP = pod.Status.PodIP
		if row.IP == "" {
			row.IP = "?"
		}
		if node != "" {
			row.Node = node
			if _, ok := nodes[node]; ok {
				if nodes[node].Labels["node-role.kubernetes.io/spot-worker"] != "" {
					row.Spot = tick
				} else {
					row.Spot = "x"
				}
				row.AZ = util.LastSplitItem(nodes[node].Labels["topology.kubernetes.io/zone"], "")
			} else {
				row.Node += " (gone)"
			}
		}

		tbl.Append(&row)
	}

	// Sort function to sort the rows slice by Namespace and then Name when iterating through it.
	slices.SortFunc(tbl.Rows, func(a, b *tableRow) int {
		return cmp.Or(
			cmp.Compare(a.Namespace, b.Namespace),
			cmp.Compare(a.Name, b.Name),
		)
	})

	// Display the table.
	tbl.Write()
}
