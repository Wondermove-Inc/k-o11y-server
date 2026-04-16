package preferencetypes

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeAlertBaseURL(t *testing.T) {
	testCases := []struct {
		name          string
		input         any
		expectedValue string
		expectError   bool
	}{
		{
			name:          "ValidHTTPURL",
			input:         "http://localhost:8080",
			expectedValue: "http://localhost:8080",
		},
		{
			name:          "ValidHTTPSURLWithTrailingSlash",
			input:         "https://example.com/",
			expectedValue: "https://example.com",
		},
		{
			name:        "InvalidScheme",
			input:       "ftp://example.com",
			expectError: true,
		},
		{
			name:        "MissingScheme",
			input:       "example.com",
			expectError: true,
		},
		{
			name:        "PathNotAllowed",
			input:       "https://example.com/alerts",
			expectError: true,
		},
		{
			name:        "QueryNotAllowed",
			input:       "https://example.com?x=1",
			expectError: true,
		},
		{
			name:        "EmptyValue",
			input:       "",
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			value, err := NormalizeAlertBaseURL(tc.input)
			if tc.expectError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.expectedValue, value)
		})
	}
}
