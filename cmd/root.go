// Package cmd implements the command-line interface for AGK.
package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/agenticgokit/agenticgokit/observability"
	"github.com/agenticgokit/agk/internal/utils"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile        string
	verbose        bool
	debug          bool
	trace          bool
	traceExporter  string
	traceEndpoint  string
	traceSample    float64
	storePrompts   bool
	tracerShutdown func(context.Context) error
	logger         *zerolog.Logger
)

// rootCmd represents the base command
var rootCmd = &cobra.Command{
	Use:   "agk",
	Short: "AgenticGoKit Developer CLI",
	Long: `AGK is a comprehensive developer tool for AgenticGoKit framework.

Create, manage, test, and debug multi-agent AI systems with ease.

Features:
  • Project scaffolding and initialization
  • Workflow execution and testing
  • MCP server management
  • Debugging and trace visualization
  • Memory and knowledge base management

Get started with: agk init my-project`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Initialize zerolog
		var err error
		logger, err = utils.NewLogger(debug)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
			os.Exit(1)
		}
		// Set a global level as well for libraries using zerolog's package logger
		if debug {
			zerolog.SetGlobalLevel(zerolog.DebugLevel)
		} else {
			zerolog.SetGlobalLevel(zerolog.InfoLevel)
		}
		// Ensure timestamps are enabled
		*logger = logger.With().Timestamp().Logger()
		// Use RFC3339 time format consistently
		zerolog.TimeFieldFormat = time.RFC3339

		// Initialize tracing if enabled
		trace = viper.GetBool("trace")
		traceExporter = viper.GetString("trace_exporter")
		traceEndpoint = viper.GetString("trace_endpoint")
		traceSample = viper.GetFloat64("trace_sample")

		if trace {
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}

			runID := generateRunID()
			ctx = observability.WithRunID(ctx, runID)
			ctx = observability.WithLogger(ctx, logger)
			cmd.SetContext(ctx)

			cfg := observability.TracerConfig{
				ServiceName:    "agk-cli",
				ServiceVersion: Version,
				Environment:    viper.GetString("environment"),
				Endpoint:       traceEndpoint,
				Exporter:       traceExporter,
				SampleRate:     traceSample,
				Debug:          debug,
				FilePath:       traceEndpoint,
			}

			tracerShutdown, err = observability.SetupTracer(ctx, cfg)
			if err != nil {
				logger.Error().Err(err).Msg("failed to set up tracer")
			}
		}
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		if tracerShutdown != nil {
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}
			_ = tracerShutdown(ctx)
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.agk.toml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "debug mode")
	rootCmd.PersistentFlags().BoolVar(&trace, "trace", false, "enable tracing")
	rootCmd.PersistentFlags().StringVar(&traceExporter, "trace-exporter", "console", "trace exporter: console|otlp|file")
	rootCmd.PersistentFlags().StringVar(&traceEndpoint, "trace-endpoint", "", "OTLP endpoint URL or file path (for file exporter)")
	rootCmd.PersistentFlags().Float64Var(&traceSample, "trace-sample", 1.0, "trace sample rate (0.0-1.0)")
	rootCmd.PersistentFlags().BoolVar(&storePrompts, "store-prompts", false, "store prompts for debugging (if supported by commands)")

	// Bind flags to viper
	_ = viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
	_ = viper.BindPFlag("debug", rootCmd.PersistentFlags().Lookup("debug"))
	_ = viper.BindPFlag("trace", rootCmd.PersistentFlags().Lookup("trace"))
	_ = viper.BindPFlag("trace_exporter", rootCmd.PersistentFlags().Lookup("trace-exporter"))
	_ = viper.BindPFlag("trace_endpoint", rootCmd.PersistentFlags().Lookup("trace-endpoint"))
	_ = viper.BindPFlag("trace_sample", rootCmd.PersistentFlags().Lookup("trace-sample"))
	_ = viper.BindPFlag("store_prompts", rootCmd.PersistentFlags().Lookup("store-prompts"))
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		viper.AddConfigPath(home)
		viper.SetConfigType("toml")
		viper.SetConfigName(".agk")
	}

	viper.SetEnvPrefix("AGK")
	viper.AutomaticEnv()

	viper.SetDefault("trace_exporter", "console")
	viper.SetDefault("trace_sample", 1.0)
	viper.SetDefault("environment", "dev")

	if err := viper.ReadInConfig(); err == nil && verbose {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}

// GetLogger returns the configured logger
func GetLogger() *zerolog.Logger {
	if logger == nil {
		if l, err := utils.NewLogger(false); err == nil {
			logger = l
		} else {
			// Fallback to a basic stderr logger
			l := zerolog.New(os.Stderr).With().Timestamp().Logger()
			logger = &l
		}
	}
	return logger
}

func generateRunID() string {
	return fmt.Sprintf("run-%d", time.Now().UnixNano())
}
