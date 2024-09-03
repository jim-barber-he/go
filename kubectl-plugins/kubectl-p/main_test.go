package main

import "testing"

func TestTabTitleRow(t *testing.T) {
	t.Parallel()

	tests := []struct {
		row    tableRow
		result string
	}{
		{
			row:    tableRow{},
			result: "NAME	READY	STATUS	RESTARTS	AGE	IP	NODE	SPOT",
		},
		{
			row: tableRow{
				Namespace: "default",
				Name:      "pod1",
				Ready:     "1/1",
				Status:    "Running",
				Restarts:  "0",
				Age:       "1d",
				IP:        "10.1.1.1",
				Node:      "node1",
				Spot:      tick,
				AZ:        "ap-southeast-2a",
			},
			result: "NAMESPACE	NAME	READY	STATUS	RESTARTS	AGE	IP	NODE	SPOT	AZ",
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
				Name:     "pod1",
				Ready:    "1/1",
				Status:   "Running",
				Restarts: "0",
				Age:      "1d",
				IP:       "10.1.1.1",
				Node:     "node1",
				Spot:     "x",
				AZ:       "ap-southeast-2a",
			},
			result: "pod1	1/1	Running	0	1d	10.1.1.1	node1	x	ap-southeast-2a",
		},
		{
			row: tableRow{
				Namespace: "default",
				Name:      "pod1",
				Ready:     "1/1",
				Status:    "Running",
				Restarts:  "0",
				Age:       "1d",
				IP:        "10.1.1.1",
				Node:      "node1",
				Spot:      "x",
				AZ:        "ap-southeast-2a",
			},
			result: "default	pod1	1/1	Running	0	1d	10.1.1.1	node1	x	ap-southeast-2a",
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
