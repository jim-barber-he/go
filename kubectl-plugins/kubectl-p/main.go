/*
A kubectl plugin to implement the 'kubectl p' command when placed in your PATH.

This is a bit like 'kubectl get pods -o wide' but shows columns with more detail on the node and AWS availability zone.
*/
package main

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"regexp"
	"runtime"
	"runtime/pprof"
	"slices"

	"github.com/jim-barber-he/go/k8s"
	"github.com/jim-barber-he/go/texttable"
	"github.com/jim-barber-he/go/util"
	flag "github.com/spf13/pflag"
	"golang.org/x/sync/errgroup"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

const tick = "\u2713"

var errNoPodsFound = errors.New("no pods found")

// tableRow represents a row in the output table.
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

// Commandline options.
type options struct {
	allNamespaces bool
	kubeContext   string
	labelSelector string
	match         string
	namespace     string
	node          string
	notMatch      string
	notStatus     string
	profileCPU    string
	profileMemory string
	status        string
	version       bool
}

func main() {
	var opts options

	flag.BoolVarP(
		&opts.allNamespaces,
		"all-namespaces",
		"A",
		false,
		"List the pods across all namespaces. Overrides --namespace / -n",
	)
	flag.StringVar(&opts.match, "match", "", "Limit output to pods with names matching this regex")
	flag.StringVar(&opts.kubeContext, "context", "", "The name of the kubeconfig context to use")
	flag.StringVarP(&opts.labelSelector, "selector", "l", "", "Selector (label query) to filter on")
	flag.StringVarP(&opts.namespace, "namespace", "n", "", "If present, the namespace scope for this CLI request")
	flag.StringVar(&opts.node, "node", "", "Limit output to pods running on nodes matching this regex")
	flag.StringVar(&opts.notMatch, "not-match", "", "Limit output to pods with names not matching this regex")
	flag.StringVar(&opts.notStatus, "not-status", "", "Limit output to pods with a status not matching this regex")
	flag.StringVar(&opts.profileCPU, "profile-cpu", "", "Produce pprof cpu profiling output in supplied file")
	flag.StringVar(&opts.profileMemory, "profile-mem", "", "Produce pprof memory profiling output in supplied file")
	flag.StringVar(&opts.status, "status", "", "Limit output to pods with a status matching this regex")
	flag.BoolVarP(&opts.version, "version", "v", false, "Print the version of this tool")
	flag.Parse()

	if opts.version {
		util.DisplayVersion("kubectl-p")

		return
	}

	// Have run() do the main work so that it can use defer statements,
	// while still giving us, the ability to use os.Exit(1) or log.Fatal*.
	err := run(opts)
	if err != nil {
		log.Fatalln(err)
	}
}

// run is the main part of the program.
func run(opts options) error {
	// CPU profiling.
	if opts.profileCPU != "" {
		fp, err := os.Create(opts.profileCPU)
		if err != nil {
			return fmt.Errorf("failed to create CPU profile file: %w", err)
		}
		defer func(fp *os.File) {
			err := fp.Close()
			if err != nil {
				log.Println(err)
			}
		}(fp)

		err = pprof.StartCPUProfile(fp)
		if err != nil {
			return fmt.Errorf("failed to start CPU profile: %w", err)
		}

		defer pprof.StopCPUProfile()
	}

	clientset := k8s.Client(opts.kubeContext)

	// Select the namespace to look at based on the command line options passed.
	namespace, err := selectNamespace(clientset, opts)
	if err != nil {
		return err
	}

	// Fetch the list of nodes and pods in parallel.
	nodes, pods, err := fetchNodesAndPods(clientset, namespace, opts.labelSelector)
	if err != nil {
		return err
	}

	// Remove pods that don't match the various filtering options.
	podItems := pods.Items

	// If the --match option was passed, then filter out the pod names that don't match.
	if opts.match != "" {
		re := regexp.MustCompile(opts.match)
		podItems = slices.DeleteFunc(podItems, func(pod v1.Pod) bool {
			return !re.MatchString(pod.Name)
		})
	}

	// If the --node option was passed, then filter out the pods that aren't on nodes whos names don't match the
	// regex.
	if opts.node != "" {
		re := regexp.MustCompile(opts.node)
		podItems = slices.DeleteFunc(podItems, func(pod v1.Pod) bool {
			return !re.MatchString(pod.Spec.NodeName)
		})
	}

	// If the --not-match option was passed, then filter out the pod names that match.
	if opts.notMatch != "" {
		re := regexp.MustCompile(opts.notMatch)
		podItems = slices.DeleteFunc(podItems, func(pod v1.Pod) bool {
			return re.MatchString(pod.Name)
		})
	}

	// If the --not-status option was passed, then filter out the pods that match.
	if opts.notStatus != "" {
		re := regexp.MustCompile(opts.notStatus)
		podItems = slices.DeleteFunc(podItems, func(pod v1.Pod) bool {
			return re.MatchString(k8s.PodDetails(&pod).Status)
		})
	}

	// If the --status option was passed, then filter out the pods that don't match.
	if opts.status != "" {
		re := regexp.MustCompile(opts.status)
		podItems = slices.DeleteFunc(podItems, func(pod v1.Pod) bool {
			return !re.MatchString(k8s.PodDetails(&pod).Status)
		})
	}

	if len(podItems) == 0 {
		return util.NewError("no pods found", "No pods found matching the options passed")
	}

	pods.Items = podItems

	// Build and display the table for each pod.
	buildAndDisplayTable(pods, nodes, opts.allNamespaces)

	// Memory profiling.
	if opts.profileMemory != "" {
		fp, err := os.Create(opts.profileMemory)
		if err != nil {
			return fmt.Errorf("failed to create memory profile file: %w", err)
		}
		defer func(fp *os.File) {
			err := fp.Close()
			if err != nil {
				log.Println(err)
			}
		}(fp)
		// Get up-to-date statistics.
		runtime.GC()

		err = pprof.WriteHeapProfile(fp)
		if err != nil {
			return fmt.Errorf("failed to write memory profile: %w", err)
		}
	}

	return nil
}

// buildAndDisplayTable builds the table from the pods (with some node details for the pod) and displays it.
func buildAndDisplayTable(pods *v1.PodList, nodes map[string]*v1.Node, allNamespaces bool) {
	var tbl texttable.Table[*tableRow]

	for i := range pods.Items {
		row := createTableRow(&pods.Items[i], nodes, allNamespaces)
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

// createTableRow creates a tableRow from a pod and node information.
func createTableRow(pod *v1.Pod, nodes map[string]*v1.Node, allNamespaces bool) tableRow {
	var row tableRow

	// Get details about the containers in the pod.
	details := k8s.PodDetails(pod)

	// Build up the table contents.
	if allNamespaces {
		row.Namespace = pod.Namespace
	}

	row.Name = pod.Name
	row.Ready = fmt.Sprintf("%d/%d", details.ReadyContainers, details.TotalContainers)
	row.Status = details.Status
	row.Restarts = details.Restarts
	row.Age = util.FormatAge(pod.CreationTimestamp.Time)
	row.IP = pod.Status.PodIP

	if row.IP == "" {
		row.IP = "?"
	}

	node := pod.Spec.NodeName
	if node != "" {
		row.Node = node
		if nodeInfo, ok := nodes[node]; ok {
			row.Spot = spotStatus(nodeInfo)
			row.AZ = util.LastSplitItem(nodes[node].Labels["topology.kubernetes.io/zone"], "")
		} else {
			row.Node += " (gone)"
		}
	}

	return row
}

// fetchNodesAndPods fetches the list of nodes and pods in parallel.
func fetchNodesAndPods(
	clientset *kubernetes.Clientset, namespace string, labelSelector string,
) (map[string]*v1.Node, *v1.PodList, error) {
	grp := new(errgroup.Group)

	nodes := make(map[string]*v1.Node)

	grp.Go(func() error {
		nodeList, err := k8s.ListNodes(clientset)
		if err != nil {
			return fmt.Errorf("failed to list nodes: %w", err)
		}

		for i, node := range nodeList.Items {
			nodes[node.Name] = &nodeList.Items[i]
		}

		return nil
	})

	pods := &v1.PodList{}

	grp.Go(func() error {
		listPods, err := k8s.ListPods(clientset, namespace, labelSelector)
		if err != nil {
			return fmt.Errorf("failed to list pods: %w", err)
		}

		if len(listPods.Items) == 0 {
			return errNoPodsFound
		}

		*pods = *listPods

		return nil
	})

	err := grp.Wait()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch nodes and/or pods: %w", err)
	}

	return nodes, pods, nil
}

// selectNamespace returns the namespace to use based on the command line options.
// An empty string means all namespaces.
func selectNamespace(clientset *kubernetes.Clientset, opts options) (string, error) {
	if opts.allNamespaces {
		return "", nil
	}

	if opts.namespace != "" {
		// Verify that the supplied namespace is valid.
		_, err := k8s.GetNamespace(context.Background(), clientset, opts.namespace)
		if err != nil {
			return "", fmt.Errorf("invalid namespace: %w", err)
		}

		return opts.namespace, nil
	}

	return k8s.Namespace(opts.kubeContext), nil
}

// spotStatus returns a tick if the node is a spot instance, otherwise an x.
func spotStatus(node *v1.Node) string {
	if node.Labels["node-role.kubernetes.io/spot-worker"] != "" {
		return tick
	}

	return "x"
}
