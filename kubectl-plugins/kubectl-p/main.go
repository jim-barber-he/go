/*
A kubectl plugin to implement the 'kubectl p' command when placed in your PATH.

This is a bit like 'kubectl get pods -o wide' but shows columns with more detail on the node and AWS availability zone.
*/
package main

import (
	"cmp"
	"fmt"
	"log"
	"os"
	"runtime"
	"runtime/pprof"
	"slices"

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
	allNamespaces := flag.BoolP("all-namespaces", "A", false, "List the pods across all namespaces")
	cpuProfile := flag.String("cpuprofile", "", "Produce pprof cpu profiling output in supplied file")
	kubeContext := flag.String("context", "", "The name of the kubeconfig context to use")
	labelSelector := flag.StringP("selector", "l", "", "Selector (label query) to filter on")
	memProfile := flag.String("memprofile", "", "Produce pprof memory profiling output in supplied file")

	flag.Parse()

	// CPU profiling.
	if *cpuProfile != "" {
		f, err := os.Create(*cpuProfile)
		if err != nil {
			log.Println(err)
			return
		}
		defer f.Close()
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Println(err)
			return
		}
		defer pprof.StopCPUProfile()
	}

	clientset := k8s.Client(*kubeContext)

	namespace := ""
	if !*allNamespaces {
		namespace = k8s.Namespace(*kubeContext)
	}

	nodeList, err := k8s.ListNodes(clientset)
	if err != nil {
		log.Println(err)
		return
	}
	nodes := make(map[string]*v1.Node)
	for i, node := range nodeList.Items {
		nodes[node.Name] = &nodeList.Items[i]
	}

	pods, err := k8s.ListPods(clientset, namespace, *labelSelector)
	if err != nil {
		log.Println(err)
		return
	}
	if len(pods.Items) == 0 {
		log.Println("No pods found")
		return
	}

	var tbl texttable.Table[*tableRow]
	for i, pod := range pods.Items {
		var row tableRow

		// Get details about the containers in the pod.
		readyContainers, totalContainers, status, restarts := k8s.PodDetails(&pods.Items[i])

		// Build up the table contents.
		if *allNamespaces {
			row.Namespace = pod.Namespace
		}
		row.Name = pod.Name
		row.Ready = fmt.Sprintf("%d/%d", readyContainers, totalContainers)
		row.Status = status
		row.Restarts = restarts
		row.Age = util.FormatAge(pod.CreationTimestamp.Time)
		row.IP = pod.Status.PodIP
		if row.IP == "" {
			row.IP = "?"
		}
		node := pod.Spec.NodeName
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

	// Memory profiling.
	if *memProfile != "" {
		f, err := os.Create(*memProfile)
		if err != nil {
			log.Println(err)
			return
		}
		defer f.Close()
		// Get up-to-date statistics.
		runtime.GC()
		if err := pprof.WriteHeapProfile(f); err != nil {
			log.Println(err)
		}
	}
}
