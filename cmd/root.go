package cmd

import (
	"os"
	"strings"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	logLevel string
)

var rootCmd = &cobra.Command{
	Use:   "mcp",
	Short: "MCP server for Rancher AI agent",
	Long: `The MCP server allows the Rancher AI agent to securely retrieve 
or update Kubernetes and Rancher resources across local and downstream clusters.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		initLogger()
	},
}

// Execute runs the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "", "Set the log level (debug, info, warn, error)")
}

func initLogger() {
	if strings.ToLower(logLevel) == "debug" {
		zap.ReplaceGlobals(zap.Must(zap.NewDevelopment()))
	} else {
		config := zap.NewProductionConfig()
		// remove the "caller" key from the log output
		config.EncoderConfig.CallerKey = zapcore.OmitKey
		zap.ReplaceGlobals(zap.Must(config.Build()))
	}
}
