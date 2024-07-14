/*
A kubectl plugin to implement the 'kubectl p' command when placed in your PATH.

This is a bit like 'kubectl get pods -o wide' but shows columns with more detail on the node and AWS availability zone.
*/
package main

import (
	"cmp"
	"errors"
	"fmt"
	"log"
	"os"
	"runtime"
	"runtime/pprof"
	"slices"
	"sync"

	"github.com/jim-barber-he/go/k8s"
	"github.com/jim-barber-he/go/texttable"
	"github.com/jim-barber-he/go/util"
	flag "github.com/spf13/pflag"
	v1 "k8s.io/api/core/v1"
)

const tick = "\u2713"

var errNoPodsFound = errors.New("no pods found")

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

// Commandline options.
type options struct {
	allNamespaces bool
	kubeContext   string
	labelSelector string
	namespace     string
	profileCPU    string
	profileMemory string
}

func main() {
	var opts options

	flag.BoolVarP(
		&opts.allNamespaces,
		"all-namespaces",
		"A",
		false,
		"List the pods across all namespace. Overrides --namespace / -n",
	)
	flag.StringVar(&opts.kubeContext, "context", "", "The name of the kubeconfig context to use")
	flag.StringVarP(&opts.labelSelector, "selector", "l", "", "Selector (label query) to filter on")
	flag.StringVarP(&opts.namespace, "namespace", "n", "", "If present, the namespace scope for this CLI request")
	flag.StringVar(&opts.profileCPU, "profile-cpu", "", "Produce pprof cpu profiling output in supplied file")
	flag.StringVar(&opts.profileMemory, "profile-mem", "", "Produce pprof memory profiling output in supplied file")
	flag.Parse()

	// Have run() do the main work so that it can use defer statements,
	// while still giving us, the ability to use os.Exit(1) or log.Fatal*.
	if err := run(opts); err != nil {
		log.Fatalln(err)
	}
}

// run is the main part of the program.
// Error handling isn't perfect here, and not sure how to do it better.
// If an error is returned early, then I guess any errors from the defer functions will be lost.
func run(opts options) error {
	// If the defer anonymous functions encounter an error, they can set this var to be returned to the calling function.
	var deferErrors error

	// CPU profiling.
	if opts.profileCPU != "" {
		fp, err := os.Create(opts.profileCPU)
		if err != nil {
			return err
		}
		defer func(fp *os.File) {
			if err := fp.Close(); err != nil {
				deferErrors = errors.Join(err, deferErrors)
			}
		}(fp)
		if err := pprof.StartCPUProfile(fp); err != nil {
			return err
		}
		defer pprof.StopCPUProfile()
	}

	clientset := k8s.Client(opts.kubeContext)

	// Choose the namespace to look at based on the command line options passed.
	var namespace string
	switch {
	case opts.allNamespaces:
		namespace = ""
	case opts.namespace != "":
		namespace = opts.namespace
	default:
		namespace = k8s.Namespace(opts.kubeContext)
	}

	// Fetch the log of nodes and pods in parallel.
	var wg sync.WaitGroup
	wg.Add(2)

	errGetNodes := make(chan error)
	defer close(errGetNodes)
	nodes := make(map[string]*v1.Node)
	go func(errChan chan<- error) {
		defer wg.Done()

		nodeList, err := k8s.ListNodes(clientset)
		if err != nil {
			errChan <- err
			return
		}
		for i, node := range nodeList.Items {
			nodes[node.Name] = &nodeList.Items[i]
		}
	}(errGetNodes)

	errGetPods := make(chan error)
	defer close(errGetPods)
	var pods *v1.PodList = &v1.PodList{}
	go func(pods *v1.PodList, errChan chan<- error) {
		defer wg.Done()

		listPods, err := k8s.ListPods(clientset, namespace, opts.labelSelector)
		if err != nil {
			errChan <- err
			return
		}
		if len(listPods.Items) == 0 {
			errChan <- errNoPodsFound
			return
		}
		*pods = *listPods
	}(pods, errGetPods)

	wg.Wait()

	select {
	case err := <-errGetNodes:
		return err
	case err := <-errGetPods:
		return err
	default:
		// No errors.
	}

	var tbl texttable.Table[*tableRow]
	for i, pod := range pods.Items {
		var row tableRow

		// Get details about the containers in the pod.
		readyContainers, totalContainers, status, restarts := k8s.PodDetails(&pods.Items[i])

		// Build up the table contents.
		if opts.allNamespaces {
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
	if opts.profileMemory != "" {
		fp, err := os.Create(opts.profileMemory)
		if err != nil {
			return err
		}
		defer func(fp *os.File) {
			if err := fp.Close(); err != nil {
				deferErrors = errors.Join(err, deferErrors)
			}
		}(fp)
		// Get up-to-date statistics.
		runtime.GC()
		if err := pprof.WriteHeapProfile(fp); err != nil {
			return err
		}
	}

	return deferErrors
}
