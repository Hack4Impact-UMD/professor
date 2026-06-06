package playwright

import (
	"testing"
)

func TestParseTestName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    TestMeta
		wantErr bool
	}{
		{
			name:  "private test",
			input: "[5] - A 5 pt private test",
			want:  TestMeta{Name: "A 5 pt private test", Points: 5, Public: false},
		},
		{
			name:  "public test",
			input: "[10*] - A 10 pt public test",
			want:  TestMeta{Name: "A 10 pt public test", Points: 10, Public: true},
		},
		{
			name:  "zero points",
			input: "[0] - Zero point test",
			want:  TestMeta{Name: "Zero point test", Points: 0, Public: false},
		},
		{
			name:  "zero points public",
			input: "[0*] - Zero point public test",
			want:  TestMeta{Name: "Zero point public test", Points: 0, Public: true},
		},
		{
			name:  "large point value",
			input: "[100] - Big test",
			want:  TestMeta{Name: "Big test", Points: 100, Public: false},
		},
		{
			name:  "test name with special characters",
			input: "[5] - Test: checks x > 0 && y != nil",
			want:  TestMeta{Name: "Test: checks x > 0 && y != nil", Points: 5, Public: false},
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "missing brackets",
			input:   "5 - No brackets",
			wantErr: true,
		},
		{
			name:    "missing dash separator",
			input:   "[5] No dash",
			wantErr: true,
		},
		{
			name:    "non-numeric points",
			input:   "[abc] - Bad points",
			wantErr: true,
		},
		{
			name:  "empty test name",
			input: "[5] - ",
			want:  TestMeta{Name: "", Points: 5, Public: false},
		},
		{
			name:    "star in wrong position",
			input:   "[*5] - Bad star",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseTestName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseTestName(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if !tt.wantErr {
				if got.Name != tt.want.Name {
					t.Errorf("Name = %q, want %q", got.Name, tt.want.Name)
				}
				if got.Points != tt.want.Points {
					t.Errorf("Points = %d, want %d", got.Points, tt.want.Points)
				}
				if got.Public != tt.want.Public {
					t.Errorf("Public = %v, want %v", got.Public, tt.want.Public)
				}
			}
		})
	}
}
