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

var errGettingNodes = errors.New("error getting nodes")

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
	configAccess := clientcmd.NewDefaultPathOptions()
	kubeconfig := configAccess.GetDefaultFilename()

	config, err := buildConfigFromFlags(kubeconfig, kubeContext)
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

// ListNodes returns a list of Kubernetes nodes.
func ListNodes(client kubernetes.Interface) (*v1.NodeList, error) {
	nodes, err := client.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		err = fmt.Errorf("%w: %w", errGettingNodes, err)
		return nil, err
	}
	return nodes, nil
}
