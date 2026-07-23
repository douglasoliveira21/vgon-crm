package services

import "testing"

func TestShouldExecuteFlowNode(t *testing.T) {
	tests := []struct {
		name                string
		alreadyCompleted    bool
		resumedFromResponse bool
		step                int
		want                bool
	}{
		{
			name: "new node always executes",
			step: 1,
			want: true,
		},
		{
			name:             "durable retry does not duplicate completed node",
			alreadyCompleted: true,
			step:             1,
			want:             false,
		},
		{
			name:                "resume start does not duplicate completed node",
			alreadyCompleted:    true,
			resumedFromResponse: true,
			step:                1,
			want:                false,
		},
		{
			name:                "loop reached after response executes prompt again",
			alreadyCompleted:    true,
			resumedFromResponse: true,
			step:                4,
			want:                true,
		},
		{
			name:             "ordinary retry remains idempotent after first step",
			alreadyCompleted: true,
			step:             4,
			want:             false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := shouldExecuteFlowNode(test.alreadyCompleted, test.resumedFromResponse, test.step)
			if got != test.want {
				t.Fatalf("shouldExecuteFlowNode(%v, %v, %d) = %v, want %v",
					test.alreadyCompleted, test.resumedFromResponse, test.step, got, test.want)
			}
		})
	}
}
