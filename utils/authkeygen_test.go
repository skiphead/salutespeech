package utils

import "testing"

func TestGenerateBasicAuthKey(t *testing.T) {
	tests := []struct {
		name     string
		clientID string
		secret   string
		want     string
		wantErr  bool
	}{
		{
			name:     "valid credentials",
			clientID: "myapp",
			secret:   "mysecret",
			want:     "bXlhcHA6bXlzZWNyZXQ=",
			wantErr:  false,
		},
		{
			name:     "empty clientID",
			clientID: "",
			secret:   "secret",
			want:     "",
			wantErr:  true,
		},
		{
			name:     "special characters",
			clientID: "user@domain.com",
			secret:   "p@ss:w0rd!",
			want:     "dXNlckBkb21haW4uY29tOnBAc3M6dzByZCE=",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GenerateBasicAuthKey(tt.clientID, tt.secret)
			if (err != nil) != tt.wantErr {
				t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("got = %q, want %q", got, tt.want)
			}
		})
	}
}
