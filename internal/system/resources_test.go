package system

import "testing"

func TestEstimateInstances(t *testing.T) {
	tests := []struct {
		ramMB        uint64
		wantStartup  uint64
		wantEfficient uint64
	}{
		{0, 0, 0},
		{255, 0, 0},
		{256, 1, 0},
		{511, 1, 0},
		{512, 2, 1},
		{1024, 4, 2},
	}
	for _, tt := range tests {
		gotStartup, gotEfficient := EstimateInstances(tt.ramMB)
		if gotStartup != tt.wantStartup || gotEfficient != tt.wantEfficient {
			t.Errorf("EstimateInstances(%d) = (%d, %d), want (%d, %d)", tt.ramMB, gotStartup, gotEfficient, tt.wantStartup, tt.wantEfficient)
		}
	}
}

