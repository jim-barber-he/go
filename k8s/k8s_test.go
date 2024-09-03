package k8s

import (
	"context"
	"testing"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestGetNamespace(t *testing.T) {
	t.Parallel()

	// Create a fake client
	client := fake.NewSimpleClientset()

	// Create a fake namespace
	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
		},
	}
	_, err := client.CoreV1().Namespaces().Create(context.Background(), ns, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("error creating namespace: %v", err)
	}

	// Get the namespace
	ptr, err := GetNamespace(client, "test")
	if err != nil {
		t.Fatalf("error getting namespace: %v", err)
	}

	// Verify the namespace
	if ptr.Name != "test" {
		t.Fatalf("expected namespace name to be 'test', got '%s'", ptr.Name)
	}
}

func TestGetNode(t *testing.T) {
	t.Parallel()

	// Create a fake client
	client := fake.NewSimpleClientset()

	// Create a fake node
	node := &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
		},
	}
	_, err := client.CoreV1().Nodes().Create(context.Background(), node, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("error creating node: %v", err)
	}

	// Get the node
	ptr, err := GetNode(client, "test")
	if err != nil {
		t.Fatalf("error getting node: %v", err)
	}

	// Verify the node
	if ptr.Name != "test" {
		t.Fatalf("expected node name to be 'test', got '%s'", ptr.Name)
	}
}

func TestHasPodReadyCondition(t *testing.T) {
	t.Parallel()

	// Create a fake pod with a ready condition
	podReady := &v1.Pod{
		Status: v1.PodStatus{
			Conditions: []v1.PodCondition{
				{
					Type:   v1.PodReady,
					Status: v1.ConditionTrue,
				},
			},
		},
	}

	// Create a fake pod with a not ready condition
	podNotReady := &v1.Pod{
		Status: v1.PodStatus{
			Conditions: []v1.PodCondition{
				{
					Type:   v1.PodReady,
					Status: v1.ConditionFalse,
				},
			},
		},
	}

	// Verify the pod has a ready condition
	if !hasPodReadyCondition(podReady.Status.Conditions) {
		t.Fatalf("expected pod to have a ready condition")
	}

	// Verify the pod does not have a ready condition
	if hasPodReadyCondition(podNotReady.Status.Conditions) {
		t.Fatalf("expected pod to have a not ready condition")
	}
}

func TestIsPodInitializedConditionTrue(t *testing.T) {
	t.Parallel()

	// Create a fake pod with an initialized condition
	podInitialized := &v1.Pod{
		Status: v1.PodStatus{
			Conditions: []v1.PodCondition{
				{
					Type:   v1.PodInitialized,
					Status: v1.ConditionTrue,
				},
			},
		},
	}

	// Create a fake pod with a not initialized condition
	podNotInitialized := &v1.Pod{
		Status: v1.PodStatus{
			Conditions: []v1.PodCondition{
				{
					Type:   v1.PodInitialized,
					Status: v1.ConditionFalse,
				},
			},
		},
	}

	// Verify the pod has an initialized condition
	if !isPodInitializedConditionTrue(&podInitialized.Status) {
		t.Fatalf("expected pod to have an initialized condition")
	}

	// Verify the pod does not have an initialized condition
	if isPodInitializedConditionTrue(&podNotInitialized.Status) {
		t.Fatalf("expected pod to have a not initialized condition")
	}
}

func TestIsRestartableInitContainer(t *testing.T) {
	t.Parallel()

	// Create a fake pod with a restartable init container
	containerRestartPolicyAlways := v1.ContainerRestartPolicyAlways

	podRestartable := &v1.Pod{
		Spec: v1.PodSpec{
			InitContainers: []v1.Container{
				{
					Name:          "init",
					Image:         "busybox",
					RestartPolicy: &containerRestartPolicyAlways,
				},
			},
		},
	}

	// Create a fake pod with a non-restartable init container
	podNonRestartable := &v1.Pod{
		Spec: v1.PodSpec{
			InitContainers: []v1.Container{
				{
					Name:  "init",
					Image: "busybox",
				},
			},
		},
	}

	// Verify the pod has a restartable init container

	if !isRestartableInitContainer(&podRestartable.Spec.InitContainers[0]) {
		t.Fatalf("expected pod to have a restartable init container")
	}

	// Verify the pod does not have a restartable init container
	if isRestartableInitContainer(&podNonRestartable.Spec.InitContainers[0]) {
		t.Fatalf("expected pod to have a non-restartable init container")
	}
}

func TestListNodes(t *testing.T) {
	t.Parallel()

	// Create a fake client
	client := fake.NewSimpleClientset()

	// Create a fake node
	node := &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
		},
	}
	_, err := client.CoreV1().Nodes().Create(context.Background(), node, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("error creating node: %v", err)
	}

	// List the nodes
	nodes, err := ListNodes(client)
	if err != nil {
		t.Fatalf("error listing nodes: %v", err)
	}

	// Verify the node
	if len(nodes.Items) != 1 {
		t.Fatalf("expected 1 node, got %d", len(nodes.Items))
	}
	if nodes.Items[0].Name != "test" {
		t.Fatalf("expected node name to be 'test', got '%s'", nodes.Items[0].Name)
	}
}

func TestListPods(t *testing.T) {
	t.Parallel()

	// Create a fake client
	client := fake.NewSimpleClientset()

	// Create a fake pod
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
	}
	_, err := client.CoreV1().Pods("default").Create(context.Background(), pod, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("error creating pod: %v", err)
	}

	// List the pods
	pods, err := ListPods(client, "default", "")
	if err != nil {
		t.Fatalf("error listing pods: %v", err)
	}

	// Verify the pod
	if len(pods.Items) != 1 {
		t.Fatalf("expected 1 pod, got %d", len(pods.Items))
	}
	if pods.Items[0].Name != "test" {
		t.Fatalf("expected pod name to be 'test', got '%s'", pods.Items[0].Name)
	}
}

/* TODO: Need to set up the status on the mocked pod.
func TestPodDetails(t *testing.T) {
	t.Parallel()

	// Create a fake client
	client := fake.NewSimpleClientset()

	// Create a fake pod
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
	}
	_, err := client.CoreV1().Pods("default").Create(context.Background(), pod, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("error creating pod: %v", err)
	}

	// Get the pod details
	ready, total, status, restarts := PodDetails(pod)
	t.Fatalf("[%d] [%d] [%s] [%s]", ready, total, status, restarts)
}
*/
