package tools

/*
func TestGetKubernetesResource(t *testing.T) {
	podJSON := `{"apiVersion":"v1","kind":"Pod","metadata":{"name":"rancher"},"spec":{"containers":[{"name":"rancher-container","image":"rancher:latest"}]}}`
	podJSONWithManagedFields := `{"apiVersion":"v1","kind":"Pod","metadata":{"name":"rancher","managedFields":{"apiVersion": "v1","fieldsType":"FieldsV1"}},"spec":{"containers":[{"name":"rancher-container","image":"rancher:latest"}]}}`
	deploymentJSON := `{"apiVersion":"apps/v1","kind":"Deployment","metadata":{"name":"nginx-deployment"},"spec":{"replicas":1,"selector":{"matchLabels":{"app":"nginx"}},"template":{"metadata":{"labels":{"app":"nginx"}},"spec":{"containers":[{"name":"nginx","image":"nginx:1.14.2","ports":[{"containerPort":80}]}]}}}}`

	tests := map[string]struct {
		params         GetKubernetesResourceParams
		mockResponse   string
		expectedPath   string
		expectedResult string
		expectedError  string
	}{
		"pod": {
			params:         GetKubernetesResourceParams{Name: "rancher", Kind: "pod", Namespace: "default", Cluster: "local"},
			mockResponse:   podJSON,
			expectedPath:   "/k8s/clusters/local/v1/pod/default/rancher",
			expectedResult: podJSON,
		},
		"pod with managed fields": {
			params:         GetKubernetesResourceParams{Name: "rancher", Kind: "pod", Namespace: "default", Cluster: "local"},
			mockResponse:   podJSONWithManagedFields,
			expectedPath:   "/k8s/clusters/local/v1/pod/default/rancher",
			expectedResult: podJSON,
		},
		"deployment": {
			params:         GetKubernetesResourceParams{Name: "rancher", Kind: "deployment", Namespace: "default", Cluster: "local"},
			mockResponse:   deploymentJSON,
			expectedPath:   "/k8s/clusters/local/v1/apps.deployment/default/rancher",
			expectedResult: deploymentJSON,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, test.expectedPath, r.URL.Path)
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(test.mockResponse))
			}))
			defer mockServer.Close()
			tools := &Tools{}

			result, _, err := tools.GetResource(nil, &mcp.CallToolRequest{
				Extra: &mcp.RequestExtra{
					Header: map[string][]string{
						"R_url": {mockServer.URL},
					},
				},
			}, test.params)

			if test.expectedError != "" {
				assert.ErrorContains(t, err, test.expectedError)
			} else {
				assert.NoError(t, err)
				assert.JSONEq(t, test.expectedResult, result.Content[0].(*mcp.TextContent).Text)
			}
		})
	}
}

func TestUpdateKubernetesResource(t *testing.T) {
	tests := map[string]struct {
		params         UpdateKubernetesResourceParams
		obj            runtime.Object
		expectedError  string
		expectedResult string
	}{
		"update pod": {
			params: UpdateKubernetesResourceParams{
				Name:      "pod-1",
				Namespace: "dev",
				Kind:      "pod",
				Cluster:   "local",
				Patch: []interface{}{
					map[string]interface{}{
						"op":    "replace",
						"path":  "/spec/containers/0/image",
						"value": "rancher:2",
					},
				},
			},
			obj: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod-1",
					Namespace: "dev",
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "rancher-container",
							Image: "rancher:1",
						},
					},
				},
			},
			expectedResult: `{"apiVersion":"v1","kind":"Pod","metadata":{"name":"pod-1","namespace":"dev"},"spec":{"containers":[{"image":"rancher:2","name":"rancher-container","resources":{}}]},"status":{}}`,
		},
		"invalid patch": {
			params: UpdateKubernetesResourceParams{
				Name:      "pod-1",
				Namespace: "dev",
				Kind:      "pod",
				Cluster:   "local",
				Patch: []interface{}{
					map[string]interface{}{
						"op":    "invalid",
						"path":  "/spec/invalid",
						"value": "rancher:2",
					},
				},
			},
			obj: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod-1",
					Namespace: "dev",
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "rancher-container",
							Image: "rancher:1",
						},
					},
				},
			},
			expectedError: "failed to update resource pod-1 in namespace dev: Unexpected kind: invalid",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			fakeClient := createFakeDynamicClient(test.obj)
			tools := &Tools{
				createDynamicClientFunc: func(token string, url string) (dynamic.Interface, error) {
					return fakeClient, nil
				},
			}
			result, _, err := tools.UpdateKubernetesResource(nil, &mcp.CallToolRequest{
				Extra: &mcp.RequestExtra{
					Header: map[string][]string{
						"R_url":   {"https://localhost:8080"},
						"R_token": {"token-xxxx"},
					},
				},
			}, test.params)

			if test.expectedError != "" {
				assert.ErrorContains(t, err, test.expectedError)
			} else {
				assert.NoError(t, err)
				assert.JSONEq(t, test.expectedResult, result.Content[0].(*mcp.TextContent).Text)
			}
		})
	}
}

func TestListKubernetesResource(t *testing.T) {
	podsJSON := `["pod1", "pod2"]`
	deploymentsJSON := `["deployment1", "deployment2"]`

	tests := map[string]struct {
		params         ListKubernetesResourcesParams
		mockResponse   string
		expectedPath   string
		expectedResult string
		expectedError  string
	}{
		"pod": {
			params:         ListKubernetesResourcesParams{Kind: "pod", Namespace: "default", Cluster: "local"},
			mockResponse:   podsJSON,
			expectedPath:   "/k8s/clusters/local/v1/pod/default",
			expectedResult: podsJSON,
		},
		"deployment": {
			params:         ListKubernetesResourcesParams{Kind: "deployment", Namespace: "default", Cluster: "local"},
			mockResponse:   deploymentsJSON,
			expectedPath:   "/k8s/clusters/local/v1/apps.deployment/default",
			expectedResult: deploymentsJSON,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, test.expectedPath, r.URL.Path)
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(test.mockResponse))
			}))
			defer mockServer.Close()
			tools := &Tools{}

			result, _, err := tools.ListKubernetesResources(nil, &mcp.CallToolRequest{
				Extra: &mcp.RequestExtra{
					Header: map[string][]string{
						"R_url": {mockServer.URL},
					},
				},
			}, test.params)

			if test.expectedError != "" {
				assert.ErrorContains(t, err, test.expectedError)
			} else {
				assert.NoError(t, err)
				assert.JSONEq(t, test.expectedResult, result.Content[0].(*mcp.TextContent).Text)
			}
		})
	}
}

func TestGetNodes(t *testing.T) {
	nodes := `{"type":"collection","resourceType":"node","count":1,"data":[{"id":"k3d-test-server-0","type":"node","apiVersion":"v1","kind":"Node"}]}`
	nodesMetrics := `{"type":"collection","resourceType":"metrics.k8s.io.nodemetrics","count":1,"data":[{"id":"k3d-test-server-0","type":"metrics.k8s.io.nodemetrics","apiVersion":"metrics.k8s.io/v1beta1","kind":"NodeMetrics","name":"k3d-test-server-0","relationships":null,"state":{"error":false,"message":"Resourceiscurrent","name":"active","transitioning":false}},"timestamp":"2025-09-18T15:56:28Z","usage":{"cpu":"215886808n","memory":"2794176Ki"},"window":"20.028s"}]}`
	tests := map[string]struct {
		params                   GetNodesParams
		mockNodesResponse        string
		mockNodesMetricsResponse string
		expectedNodesPath        string
		expectedMetricsPath      string
		expectedResult           string
		expectedError            string
	}{
		"get nodes": {
			params:                   GetNodesParams{Cluster: "local"},
			mockNodesResponse:        nodes,
			mockNodesMetricsResponse: nodesMetrics,
			expectedNodesPath:        "/k8s/clusters/local/v1/nodes",
			expectedMetricsPath:      "/k8s/clusters/local/v1/metrics.k8s.io.nodes",
			expectedResult:           nodes + nodesMetrics,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			tools := &Tools{}
			mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case test.expectedNodesPath:
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(test.mockNodesResponse))
				case test.expectedMetricsPath:
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(test.mockNodesMetricsResponse))
				default:
					assert.Fail(t, fmt.Sprintf("unexpected path: %s", r.URL.Path))
				}
			}))
			defer mockServer.Close()

			result, _, err := tools.GetNodes(nil, &mcp.CallToolRequest{
				Extra: &mcp.RequestExtra{
					Header: map[string][]string{
						"R_url": {mockServer.URL},
					},
				},
			}, test.params)

			if test.expectedError != "" {
				assert.ErrorContains(t, err, test.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, test.expectedResult, result.Content[0].(*mcp.TextContent).Text)
			}
		})
	}
}

func TestGetPodLogs(t *testing.T) {
	fakeLogs := "fake logs" // this is hardcoded in the fake client. see fake_pod_expansion.go

	tests := map[string]struct {
		params         GetPodLogsParams
		mockClient     func(token string, url string) (kubernetes.Interface, error)
		expectedError  string
		expectedResult string
	}{
		"get logs": {
			params: GetPodLogsParams{
				Name:      "pod-1",
				Namespace: "dev",
				Cluster:   "local",
			},
			mockClient: func(token string, url string) (kubernetes.Interface, error) {
				pod := &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod-1",
						Namespace: "dev",
					},
				}

				return fake.NewClientset(pod), nil
			},
			expectedResult: fakeLogs,
		},
		"error creating client": {
			params: GetPodLogsParams{
				Name:      "pod-1",
				Namespace: "dev",
				Cluster:   "local",
			},
			mockClient: func(token string, url string) (kubernetes.Interface, error) {
				return nil, errors.New("fake error")
			},
			expectedError: "failed to create clientset: fake error",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			tools := &Tools{
				createClientSetFunc: test.mockClient,
			}
			result, _, err := tools.GetPodLogs(context.TODO(), &mcp.CallToolRequest{
				Extra: &mcp.RequestExtra{
					Header: map[string][]string{
						"R_url":   {"https://localhost:8080"},
						"R_token": {"token-xxxx"},
					},
				},
			}, test.params)

			if test.expectedError != "" {
				assert.ErrorContains(t, err, test.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, test.expectedResult, result.Content[0].(*mcp.TextContent).Text)
			}
		})
	}
}

// TODO testCreateResource
// TODO add human validation for create

func createFakeDynamicClient(objects ...runtime.Object) *fakedyn.FakeDynamicClient {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	dynClient := fakedyn.NewSimpleDynamicClient(scheme, objects...)

	return dynClient
}
*/
