package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/agenticgokit/agk/pkg/registry"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var templateCmd = &cobra.Command{
	Use:   "template",
	Short: "Manage project templates",
	Long:  `Manage local and remote templates for AGK projects.`,
}

var templateListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available templates",
	RunE: func(cmd *cobra.Command, args []string) error {
		cm, err := registry.NewCacheManager("")
		if err != nil {
			return err
		}

		templates, err := cm.List()
		if err != nil {
			return err
		}

		if len(templates) == 0 {
			fmt.Println("No templates found in cache. Add one with 'agk template add'.")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		_, _ = fmt.Fprintln(w, "NAME\tVERSION\tSOURCE\tDESCRIPTION")
		for _, t := range templates {
			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", t.Name, t.Version, t.Source, t.Description)
		}
		_ = w.Flush()
		return nil
	},
}

var templateAddCmd = &cobra.Command{
	Use:   "add [source]",
	Short: "Add a template to the cache",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		source := args[0]

		fmt.Printf("Fetching template from %s...\n", source)

		cm, err := registry.NewCacheManager("")
		if err != nil {
			return err
		}

		resolver := registry.NewResolver(cm)

		// Use context.Background for now
		tmpl, err := resolver.Resolve(cmd.Context(), source)
		if err != nil {
			return err
		}

		color.Green("Successfully added template: %s (%s)", tmpl.Name, tmpl.Version)
		return nil
	},
}

var templateRemoveCmd = &cobra.Command{
	Use:   "remove [name|source]",
	Short: "Remove a template from the cache",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		source := args[0]

		cm, err := registry.NewCacheManager("")
		if err != nil {
			return err
		}

		// Try to remove by exact source match first, then maybe by name?
		// CacheManager.Remove takes source.
		// If user passes "rag-agent" (name) but source is "github.com/...", Remove might fail.
		// TODO: Implement lookup by name in CacheManager to support removing by name.
		// For now, assume source.

		if err := cm.Remove(source, ""); err != nil {
			return err
		}

		color.Green("Removed template: %s", source)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(templateCmd)
	templateCmd.AddCommand(templateListCmd)
	templateCmd.AddCommand(templateAddCmd)
	templateCmd.AddCommand(templateRemoveCmd)
}
