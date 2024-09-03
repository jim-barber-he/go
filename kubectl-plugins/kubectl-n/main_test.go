package main

import (
	"reflect"
	"testing"

	v1 "k8s.io/api/core/v1"
)

func TestTabTitleRow(t *testing.T) {
	t.Parallel()

	tests := []struct {
		row    tableRow
		result string
	}{
		{
			row:    tableRow{},
			result: "NAME	OK	AGE	VERSION	RUNTIME",
		},
		{
			row: tableRow{
				Name:          "node1",
				Ok:            tick,
				Age:           "1d",
				Version:       "v1.30.0",
				Runtime:       "containerd",
				Type:          "t3.medium",
				Spot:          "true",
				AZ:            "ap-southeast-2a",
				InstanceID:    "i-1234567890abcdef0",
				IP:            "10.1.1.1",
				InstanceGroup: "nodes",
			},
			result: "NAME	OK	AGE	VERSION	RUNTIME	TYPE	SPOT	AZ	INSTANCE-ID	IP-ADDRESS	INSTANCE-GROUP",
		},
	}

	for _, tt := range tests {
		t.Run("TabTitleRow", func(t *testing.T) {
			t.Parallel()
			result := tt.row.TabTitleRow()
			if result != tt.result {
				t.Errorf("got %s, want %s", result, tt.result)
			}
		})
	}
}

func TestTabValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		row    tableRow
		result string
	}{
		{
			row:    tableRow{},
			result: "",
		},
		{
			row: tableRow{
				Name:          "node1",
				Ok:            tick,
				Age:           "1d",
				Version:       "v1.30.0",
				Runtime:       "containerd",
				Type:          "t3.medium",
				Spot:          "true",
				AZ:            "ap-southeast-2a",
				InstanceID:    "i-1234567890abcdef0",
				IP:            "10.1.1.1",
				InstanceGroup: "nodes",
			},
			result: "node1	✓	1d	v1.30.0	containerd	t3.medium	true	ap-southeast-2a	i-1234567890abcdef0	10.1.1.1	nodes",
		},
	}

	for _, tt := range tests {
		t.Run("TabValues", func(t *testing.T) {
			t.Parallel()
			result := tt.row.TabValues()
			if result != tt.result {
				t.Errorf("got %s, want %s", result, tt.result)
			}
		})
	}
}

func TestGetNodeStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		conditions []v1.NodeCondition
		status     string
		messages   []string
	}{
		{
			conditions: []v1.NodeCondition{
				{Type: "Ready", Status: v1.ConditionTrue, Message: "Kubelet is healthy"},
			},
			status:   tick,
			messages: nil,
		},
		{
			conditions: []v1.NodeCondition{
				{Type: "Ready", Status: v1.ConditionFalse, Message: "Kubelet is unhealthy"},
			},
			status:   "x",
			messages: []string{"Node condition Ready is now: False, message: \"Kubelet is unhealthy\""},
		},
		{
			conditions: []v1.NodeCondition{
				{Type: "Ready", Status: v1.ConditionTrue, Message: "Kubelet is healthy"},
				{Type: "DiskPressure", Status: v1.ConditionTrue, Message: "Disk is full"},
			},
			status:   "x",
			messages: []string{"Node condition DiskPressure is now: True, message: \"Disk is full\""},
		},
		{
			conditions: []v1.NodeCondition{
				{Type: "Ready", Status: v1.ConditionTrue, Message: "Kubelet is healthy"},
				{Type: "CPUPressure", Status: v1.ConditionTrue, Message: "CPU is overloaded"},
			},
			status:   "x",
			messages: []string{"Node condition CPUPressure is now: True, message: \"CPU is overloaded\""},
		},
		{
			conditions: []v1.NodeCondition{
				{Type: "Ready", Status: v1.ConditionTrue, Message: "Kubelet is healthy"},
				{Type: "ContainerRuntimeUnhealthy", Status: v1.ConditionTrue, Message: "Container runtime is unhealthy"},
				{Type: "CorruptDockerOverlay2", Status: v1.ConditionTrue, Message: "Docker overlay is corrupt"},
				{Type: "CPUPressure", Status: v1.ConditionTrue, Message: "CPU is overloaded"},
				{Type: "DiskPressure", Status: v1.ConditionTrue, Message: "Disk is full"},
				{Type: "FrequentContainerdRestart", Status: v1.ConditionTrue, Message: "Containerd is restarting"},
				{Type: "FrequentDockerRestart", Status: v1.ConditionTrue, Message: "Docker is restarting"},
				{Type: "FrequentKubeletRestart", Status: v1.ConditionTrue, Message: "Kubelet is restarting"},
				{Type: "FrequentUnregisterNetDevice", Status: v1.ConditionTrue, Message: "Network device is unregistering"},
				{Type: "KernelDeadlock", Status: v1.ConditionTrue, Message: "Kernel is dead"},
				{Type: "KubeletUnhealthy", Status: v1.ConditionTrue, Message: "Kubelet is unhealthy"},
				{Type: "MemoryPressure", Status: v1.ConditionTrue, Message: "Memory is full"},
				{Type: "NetworkUnavailable", Status: v1.ConditionTrue, Message: "Network is down"},
				{Type: "OutOfDisk", Status: v1.ConditionTrue, Message: "Disk is full"},
				{Type: "PIDPressure", Status: v1.ConditionTrue, Message: "PID is full"},
				{Type: "ReadonlyFilesystem", Status: v1.ConditionTrue, Message: "Filesystem is read-only"},
			},
			status: "x",
			messages: []string{
				"Node condition ContainerRuntimeUnhealthy is now: True, message: \"Container runtime is unhealthy\"",
				"Node condition CorruptDockerOverlay2 is now: True, message: \"Docker overlay is corrupt\"",
				"Node condition CPUPressure is now: True, message: \"CPU is overloaded\"",
				"Node condition DiskPressure is now: True, message: \"Disk is full\"",
				"Node condition FrequentContainerdRestart is now: True, message: \"Containerd is restarting\"",
				"Node condition FrequentDockerRestart is now: True, message: \"Docker is restarting\"",
				"Node condition FrequentKubeletRestart is now: True, message: \"Kubelet is restarting\"",
				"Node condition FrequentUnregisterNetDevice is now: True, message: \"Network device is unregistering\"",
				"Node condition KernelDeadlock is now: True, message: \"Kernel is dead\"",
				"Node condition KubeletUnhealthy is now: True, message: \"Kubelet is unhealthy\"",
				"Node condition MemoryPressure is now: True, message: \"Memory is full\"",
				"Node condition NetworkUnavailable is now: True, message: \"Network is down\"",
				"Node condition OutOfDisk is now: True, message: \"Disk is full\"",
				"Node condition PIDPressure is now: True, message: \"PID is full\"",
				"Node condition ReadonlyFilesystem is now: True, message: \"Filesystem is read-only\"",
			},
		},
	}

	for _, tt := range tests {
		t.Run("getNodeStatus", func(t *testing.T) {
			t.Parallel()
			status, messages := getNodeStatus(tt.conditions)
			if status != tt.status {
				t.Errorf("got %s, want %s", status, tt.status)
			}
			if !reflect.DeepEqual(messages, tt.messages) {
				t.Errorf("got %v, want %v", messages, tt.messages)
			}
		})
	}
}
