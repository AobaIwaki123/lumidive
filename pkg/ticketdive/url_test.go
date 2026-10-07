package ticketdive

import (
	"testing"
)

func TestNormalizeInput(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		expectedID   string
		expectedCan  string
		expectedShrt string
		wantErr      bool
	}{
		{
			name:         "Bare event ID",
			input:        "plkt1022",
			expectedID:   "plkt1022",
			expectedCan:  "https://ticketdive.com/event/plkt1022",
			expectedShrt: "https://t-dv.com/plkt1022",
			wantErr:      false,
		},
		{
			name:         "Full canonical URL",
			input:        "https://ticketdive.com/event/plkt1022",
			expectedID:   "plkt1022",
			expectedCan:  "https://ticketdive.com/event/plkt1022",
			expectedShrt: "https://t-dv.com/plkt1022",
			wantErr:      false,
		},
		{
			name:         "Shortened URL",
			input:        "https://t-dv.com/plkt1022",
			expectedID:   "plkt1022",
			expectedCan:  "https://ticketdive.com/event/plkt1022",
			expectedShrt: "https://t-dv.com/plkt1022",
			wantErr:      false,
		},
		{
			name:         "FC event URL",
			input:        "https://ticketdive.com/event/fc/fc_event_123",
			expectedID:   "fc_event_123",
			expectedCan:  "https://ticketdive.com/event/fc_event_123",
			expectedShrt: "https://t-dv.com/fc_event_123",
			wantErr:      false,
		},
		{
			name:         "Artist URL",
			input:        "https://ticketdive.com/artist/yoruami",
			expectedID:   "yoruami",
			expectedCan:  "https://ticketdive.com/artist/yoruami",
			expectedShrt: "https://ticketdive.com/artist/yoruami",
			wantErr:      false,
		},
		{
			name:    "Empty input",
			input:   "   ",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeInput(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("NormalizeInput() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}

			if got.EventID != tt.expectedID {
				t.Errorf("EventID = %v, want %v", got.EventID, tt.expectedID)
			}
			if got.CanonicalURL != tt.expectedCan {
				t.Errorf("CanonicalURL = %v, want %v", got.CanonicalURL, tt.expectedCan)
			}
			if got.ShortURL != tt.expectedShrt {
				t.Errorf("ShortURL = %v, want %v", got.ShortURL, tt.expectedShrt)
			}
		})
	}
}
