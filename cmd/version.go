package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

var (
	// Set via ldflags during build
	Version   = "dev"
	GitCommit = "unknown"
	BuildDate = "unknown"
	GoVersion = runtime.Version()
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Long:  `Display version, build, and runtime information for AGK.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("AGK Developer CLI\n")
		fmt.Printf("Version:     %s\n", Version)
		fmt.Printf("Git Commit:  %s\n", GitCommit)
		fmt.Printf("Build Date:  %s\n", BuildDate)
		fmt.Printf("Go Version:  %s\n", GoVersion)
		fmt.Printf("OS/Arch:     %s/%s\n", runtime.GOOS, runtime.GOARCH)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
