/*
Package k8s provides common Kubernetes functions to be used by other packages.
*/
package k8s

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/jim-barber-he/go/util"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	errGettingNamespace = errors.New("error getting namespace")
	errGettingNode      = errors.New("error getting node")
	errGettingNodes     = errors.New("error getting nodes")
	errGettingPods      = errors.New("error getting pods")
)

func newContextNotFoundError(context string) error {
	return &util.Error{
		Msg:   "context ",
		Param: context + " not found in kubeconfig",
	}
}

// buildConfigFromFlags creates a Kubernetes client configuration from the provided kubeconfig path and context.
// Based on clientcmd.BuildConfigFromFlags from the kubernetes go-client but with the added `context` parameter to set
// `CurrentContext`, and with the unneeded masterUrl parameter removed.
func buildConfigFromFlags(kubeconfigPath, context string) (*rest.Config, error) {
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfigPath},
		&clientcmd.ConfigOverrides{
			CurrentContext: context,
		}).ClientConfig()
}

// Client returns a Kubernetes client.
func Client(kubeContext string) *kubernetes.Clientset {
	config, err := buildConfigFromFlags(KubeConfig(), kubeContext)
	if err != nil {
		panic(fmt.Errorf("failed to build config from flags: %w", err))
	}

	// Create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(fmt.Errorf("failed to create Kubernetes clientset: %w", err))
	}

	return clientset
}

// GetNamespace returns a namespace.
func GetNamespace(ctx context.Context, client kubernetes.Interface, name string) (*v1.Namespace, error) {
	ptr, err := client.CoreV1().Namespaces().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		err = fmt.Errorf("%w: failed to get namespace %q: %w", errGettingNamespace, name, err)

		return nil, err
	}

	return ptr, nil
}

// GetNode returns a node.
func GetNode(ctx context.Context, client kubernetes.Interface, name string) (*v1.Node, error) {
	ptr, err := client.CoreV1().Nodes().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("%w: failed to get node %q: %w", errGettingNode, name, err)
	}

	return ptr, nil
}

// hasPodReadyCondition returns true if the pod has a condition type of "Ready" with a status of "True".
func hasPodReadyCondition(conditions []v1.PodCondition) bool {
	for _, condition := range conditions {
		if condition.Type == v1.PodReady && condition.Status == v1.ConditionTrue {
			return true
		}
	}

	return false
}

// isPodInitializedConditionTrue returns true if the pod has a condition type of "Initialized" with a status of "True".
func isPodInitializedConditionTrue(status *v1.PodStatus) bool {
	for _, condition := range status.Conditions {
		if condition.Type == v1.PodInitialized && condition.Status == v1.ConditionTrue {
			return true
		}
	}

	return false
}

// isRestartableInitContainer returns true if an init container has its RestartPolicy set to "Always".
func isRestartableInitContainer(initContainer *v1.Container) bool {
	if initContainer == nil || initContainer.RestartPolicy == nil {
		return false
	}

	return *initContainer.RestartPolicy == v1.ContainerRestartPolicyAlways
}

// KubeConfig returns the user's kube config file.
func KubeConfig() string {
	configAccess := clientcmd.NewDefaultPathOptions()

	return configAccess.GetDefaultFilename()
}

// ListNodes returns a list of Kubernetes nodes.
func ListNodes(client kubernetes.Interface) (*v1.NodeList, error) {
	nodes, err := client.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errGettingNodes, err)
	}

	return nodes, nil
}

// ListPods returns a list of Kubernetes pods.
// If namespace is an empty string then pods from all namespaces are returned.
func ListPods(client kubernetes.Interface, namespace, labelSelector string) (*v1.PodList, error) {
	listOptions := metav1.ListOptions{}
	if labelSelector != "" {
		listOptions.LabelSelector = labelSelector
	}

	pods, err := client.CoreV1().Pods(namespace).List(context.Background(), listOptions)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errGettingPods, err)
	}

	return pods, nil
}

// Namespace returns the namespace name that is selected (or "default" if it is not set) for a context in kubeconfig.
// If the context that is passed in is an empty string, fall back to the selected context in kubeconfig.
// If that's not set either, then just return the "default" namespace.
func Namespace(kubeContext string) string {
	config, err := clientcmd.LoadFromFile(KubeConfig())
	if err != nil {
		panic(fmt.Errorf("failed to load kubeconfig: %w", err))
	}

	if kubeContext == "" {
		if config.CurrentContext == "" {
			return "default"
		}

		kubeContext = config.CurrentContext
	}

	context, exists := config.Contexts[kubeContext]
	if !exists {
		panic(newContextNotFoundError(kubeContext))
	}

	ns := context.Namespace
	if ns == "" {
		ns = "default"
	}

	return ns
}

// PodDetails returns details on pods as you would see in the READY, STATUS, and RESTARTS columns of kubectl output.
// The READY would be built up via "readyContainers/totalContainers".
// Based on: printPod() function in kubernetes/pkg/printers/internalversion/printers.go of kubernetes source code.
func PodDetails(pod *v1.Pod) (readyContainers, totalContainers int, status, restarts string) {
	restartCount := 0
	restartableInitContainerRestarts := 0
	totalContainers = len(pod.Spec.Containers)
	readyContainers = 0
	lastRestartDate := time.Time{}
	lastRestartableInitContainerRestartDate := time.Time{}

	podPhase := string(pod.Status.Phase)

	status = podPhase
	if pod.Status.Reason != "" {
		status = pod.Status.Reason
	}

	// If the Pod carries {type:PodScheduled, reason:SchedulingGated}, set status to 'SchedulingGated'.
	for _, condition := range pod.Status.Conditions {
		if condition.Type == v1.PodScheduled && condition.Reason == "SchedulingGated" {
			status = "SchedulingGated"
		}
	}

	initContainers := make(map[string]*v1.Container)
	for i := range pod.Spec.InitContainers {
		initContainers[pod.Spec.InitContainers[i].Name] = &pod.Spec.InitContainers[i]
		if isRestartableInitContainer(&pod.Spec.InitContainers[i]) {
			totalContainers++
		}
	}

	initializing := false

	for idx, icStatus := range pod.Status.InitContainerStatuses {
		restartCount += int(icStatus.RestartCount)
		if icStatus.LastTerminationState.Terminated != nil {
			terminatedDate := icStatus.LastTerminationState.Terminated.FinishedAt.Time
			if lastRestartDate.Before(terminatedDate) {
				lastRestartDate = terminatedDate
			}
		}

		if isRestartableInitContainer(initContainers[icStatus.Name]) {
			restartableInitContainerRestarts += int(icStatus.RestartCount)
			if icStatus.LastTerminationState.Terminated != nil {
				terminatedDate := icStatus.LastTerminationState.Terminated.FinishedAt.Time
				if lastRestartableInitContainerRestartDate.Before(terminatedDate) {
					lastRestartableInitContainerRestartDate = terminatedDate
				}
			}
		}

		switch {
		case icStatus.State.Terminated != nil && icStatus.State.Terminated.ExitCode == 0:
			continue
		case isRestartableInitContainer(initContainers[icStatus.Name]) && icStatus.Started != nil && *icStatus.Started:
			if icStatus.Ready {
				readyContainers++
			}

			continue
		case icStatus.State.Terminated != nil:
			// Initialization has failed
			if len(icStatus.State.Terminated.Reason) == 0 {
				if icStatus.State.Terminated.Signal != 0 {
					status = fmt.Sprintf("Init:Signal:%d", icStatus.State.Terminated.Signal)
				} else {
					status = fmt.Sprintf("Init:ExitCode:%d", icStatus.State.Terminated.ExitCode)
				}
			} else {
				status = "Init:" + icStatus.State.Terminated.Reason
			}

			initializing = true
		case icStatus.State.Waiting != nil &&
			len(icStatus.State.Waiting.Reason) > 0 &&
			icStatus.State.Waiting.Reason != "PodInitializing":
			status = "Init:" + icStatus.State.Waiting.Reason
			initializing = true
		default:
			status = fmt.Sprintf("Init:%d/%d", idx, len(pod.Spec.InitContainers))
			initializing = true
		}

		break
	}

	if !initializing || isPodInitializedConditionTrue(&pod.Status) {
		restartCount = restartableInitContainerRestarts
		lastRestartDate = lastRestartableInitContainerRestartDate
		hasRunning := false

		for i := len(pod.Status.ContainerStatuses) - 1; i >= 0; i-- {
			cStatus := pod.Status.ContainerStatuses[i]

			restartCount += int(cStatus.RestartCount)
			if cStatus.LastTerminationState.Terminated != nil {
				terminatedDate := cStatus.LastTerminationState.Terminated.FinishedAt.Time
				if lastRestartDate.Before(terminatedDate) {
					lastRestartDate = terminatedDate
				}
			}

			switch {
			case cStatus.State.Waiting != nil && cStatus.State.Waiting.Reason != "":
				status = cStatus.State.Waiting.Reason
			case cStatus.State.Terminated != nil:
				if cStatus.State.Terminated.Reason != "" {
					status = cStatus.State.Terminated.Reason
				} else {
					if cStatus.State.Terminated.Signal != 0 {
						status = fmt.Sprintf("Signal:%d", cStatus.State.Terminated.Signal)
					} else {
						status = fmt.Sprintf("ExitCode:%d", cStatus.State.Terminated.ExitCode)
					}
				}
			case cStatus.Ready && cStatus.State.Running != nil:
				hasRunning = true
				readyContainers++
			}
		}

		// Change pod status back to "Running" if there is at least one container still reporting as "Running" status.
		if status == "Completed" && hasRunning {
			if hasPodReadyCondition(pod.Status.Conditions) {
				status = "Running"
			} else {
				status = "NotReady"
			}
		}
	}

	if pod.DeletionTimestamp != nil {
		if pod.Status.Reason == "NodeLost" {
			status = "Unknown"
		} else if podPhase != "Failed" && podPhase != "Succeeded" {
			status = "Terminating"
		}
	}

	restarts = strconv.Itoa(restartCount)
	if restartCount != 0 && !lastRestartDate.IsZero() {
		restarts += fmt.Sprintf(" (%s ago)", util.FormatAge(lastRestartDate))
	}

	return readyContainers, totalContainers, status, restarts
}
