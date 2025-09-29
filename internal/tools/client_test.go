package tools

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateRestConfig(t *testing.T) {
	testCases := map[string]struct {
		token         string
		url           string
		expectErr     bool
		expectedHost  string
		expectedToken string
	}{
		"valid inputs": {
			token:         "my-secret-token-123",
			url:           "https://my-kubernetes-api.example.com",
			expectErr:     false,
			expectedHost:  "https://my-kubernetes-api.example.com",
			expectedToken: "my-secret-token-123",
		},
		"failure case with empty URL": {
			token:     "a-valid-token",
			url:       "",
			expectErr: true,
		},
	}

	for name, test := range testCases {
		t.Run(name, func(t *testing.T) {
			restConfig, err := createRestConfig(test.token, test.url)

			if test.expectErr {
				require.Error(t, err)
				assert.Nil(t, restConfig)
			} else {
				require.NoError(t, err)
				require.NotNil(t, restConfig)
				assert.Equal(t, test.expectedHost, restConfig.Host)
				assert.Equal(t, test.expectedToken, restConfig.BearerToken)
			}
		})
	}
}
