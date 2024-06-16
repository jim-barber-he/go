/*
Package k8s provides common Kubernetes functions to be used by other packages.
*/
package k8s

import (
	"context"
	"errors"
	"fmt"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	errGettingNode  = errors.New("error getting node")
	errGettingNodes = errors.New("error getting nodes")
	errGettingPods  = errors.New("error getting pods")
)

// Based on clientcmd.BuildConfigFromFlags from the kubernetes go-client but with the added `context` parameter to set
// `CurrentContext`, and with the unneeded masterUrl parameter removed.
func buildConfigFromFlags(kubeconfigPath, context string) (*restclient.Config, error) {
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
		panic(err.Error())
	}

	// Create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	return clientset
}

// GetNode returns a node.
func GetNode(client kubernetes.Interface, node string) (*v1.Node, error) {
	nodePtr, err := client.CoreV1().Nodes().Get(context.Background(), node, metav1.GetOptions{})
	if err != nil {
		err = fmt.Errorf("%w: %w", errGettingNode, err)
		return nil, err
	}
	return nodePtr, nil
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
		err = fmt.Errorf("%w: %w", errGettingNodes, err)
		return nil, err
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
		err = fmt.Errorf("%w: %w", errGettingPods, err)
		return nil, err
	}
	return pods, nil
}

// Namespace returns the namespace that is selected (or "default" if it is not set) for a context in kubeconfig.
// If the context that is passed in is an empty string, fall back to the selected context in kubeconfig.
// If that's not set either, then just return the "default" namespace.
func Namespace(kubeContext string) string {
	config, err := clientcmd.LoadFromFile(KubeConfig())
	if err != nil {
		panic(err.Error())
	}

	if kubeContext == "" {
		if config.CurrentContext == "" {
			return "default"
		}
		kubeContext = config.CurrentContext
	}

	ns := config.Contexts[kubeContext].Namespace
	if len(ns) == 0 {
		ns = "default"
	}
	return ns
}
