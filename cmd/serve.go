package cmd

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net/http"

	"mcp/pkg/client"
	"mcp/pkg/toolsets"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rancher/dynamiclistener"
	"github.com/rancher/dynamiclistener/server"
	"github.com/rancher/wrangler/pkg/generated/controllers/core"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"k8s.io/client-go/rest"
)

const (
	tlsName       = "rancher-mcp-server.cattle-ai-agent-system.svc"
	certNamespace = "cattle-ai-agent-system"
	certName      = "cattle-mcp-tls"
	caName        = "cattle-mcp-ca"
)

var (
	port     int
	insecure bool
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the MCP server",
	Long:  `Start the MCP server to handle requests from the Rancher AI agent`,
	Run:   runServe,
}

func init() {
	rootCmd.AddCommand(serveCmd)

	serveCmd.Flags().IntVar(&port, "port", 9092, "Port to listen on")
	serveCmd.Flags().BoolVar(&insecure, "insecure", false, "Skip TLS verification (uses INSECURE_SKIP_TLS env var if not set)")
}

func runServe(cmd *cobra.Command, args []string) {
	mcpServer := mcp.NewServer(&mcp.Implementation{Name: "rancher mcp server", Version: "v1.0.0"}, nil)
	client := client.NewClient(insecure)

	toolsets := toolsets.NewToolSetsWithAllTools(client)
	toolsets.AddTools(mcpServer)

	handler := mcp.NewStreamableHTTPHandler(func(request *http.Request) *mcp.Server {
		return mcpServer
	}, &mcp.StreamableHTTPOptions{})

	if insecure {
		startInsecureServer(handler)
	} else {
		startTLSServer(handler)
	}
}

func startInsecureServer(handler http.Handler) {
	zap.L().Info("MCP Server started!", zap.Int("port", port), zap.Bool("insecure", true))

	addr := fmt.Sprintf(":%d", port)
	log.Fatal(http.ListenAndServe(addr, handler))
}

func startTLSServer(handler http.Handler) {
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Fatalf("error creating in-cluster config: %v", err)
	}
	factory, err := core.NewFactoryFromConfig(config)
	if err != nil {
		log.Fatalf("error creating factory: %v", err)
	}

	ctx := context.Background()
	err = server.ListenAndServe(ctx, port, 0, handler, &server.ListenOpts{
		Secrets:       factory.Core().V1().Secret(),
		CertNamespace: certNamespace,
		CertName:      certName,
		CAName:        caName,
		TLSListenerConfig: dynamiclistener.Config{
			SANs: []string{
				tlsName,
			},
			FilterCN: dynamiclistener.OnlyAllow(tlsName),
			TLSConfig: &tls.Config{
				MinVersion: tls.VersionTLS12,
				CipherSuites: []uint16{
					tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
					tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
					tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
					tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
					tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
					tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
				},
				ClientAuth: tls.RequestClientCert,
			},
		},
	})
	if err != nil {
		log.Fatalf("error creating tls server: %v", err)
	}

	zap.L().Info("MCP Server with TLS started!", zap.Int("port", port))
	<-ctx.Done()
}
