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

func TestStrconvToUint64(t *testing.T) {
	tests := []struct {
		input   string
		want    uint64
		wantErr bool
	}{
		{"123", 123, false},
		{"0", 0, false},
		{"abc", 0, true},
		{"", 0, false},
	}
	for _, tt := range tests {
		got, err := strconvToUint64(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("strconvToUint64(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
		}
		if got != tt.want {
			t.Errorf("strconvToUint64(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}
