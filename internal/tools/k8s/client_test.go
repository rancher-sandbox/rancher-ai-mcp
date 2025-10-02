package k8s

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
			token:         "my-secret-Token-123",
			url:           "https://my-rancher.example.com",
			expectErr:     false,
			expectedHost:  "https://my-rancher.example.com/k8s/clusters/local",
			expectedToken: "my-secret-Token-123",
		},
	}

	for name, test := range testCases {
		t.Run(name, func(t *testing.T) {
			restConfig, err := createRestConfig(test.token, test.url, "local")

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
