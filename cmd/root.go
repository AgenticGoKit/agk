// Package cmd implements the command-line interface for AGK.
package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/agenticgokit/agk/internal/utils"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
	verbose bool
	debug   bool
	logger  *zerolog.Logger
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
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		// No cleanup required for zerolog
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

	// Bind flags to viper
	_ = viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
	_ = viper.BindPFlag("debug", rootCmd.PersistentFlags().Lookup("debug"))
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

	viper.AutomaticEnv()

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
