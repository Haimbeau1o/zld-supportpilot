package ticket

import "testing"

func TestTicketStatusTransitions(t *testing.T) {
	if InitialTicketStatus() != TicketStatusOpen {
		t.Fatalf("expected initial ticket status %q, got %q", TicketStatusOpen, InitialTicketStatus())
	}

	testCases := []struct {
		name        string
		from        TicketStatus
		to          TicketStatus
		expectError bool
	}{
		{
			name:        "open to in_progress",
			from:        TicketStatusOpen,
			to:          TicketStatusInProgress,
			expectError: false,
		},
		{
			name:        "in_progress to resolved",
			from:        TicketStatusInProgress,
			to:          TicketStatusResolved,
			expectError: false,
		},
		{
			name:        "resolved to closed",
			from:        TicketStatusResolved,
			to:          TicketStatusClosed,
			expectError: false,
		},
		{
			name:        "closed to in_progress should fail",
			from:        TicketStatusClosed,
			to:          TicketStatusInProgress,
			expectError: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			err := ValidateStatusTransition(testCase.from, testCase.to)
			if testCase.expectError && err == nil {
				t.Fatalf("expected transition from %q to %q to fail", testCase.from, testCase.to)
			}

			if !testCase.expectError && err != nil {
				t.Fatalf("expected transition from %q to %q to succeed, got error: %v", testCase.from, testCase.to, err)
			}
		})
	}
}
