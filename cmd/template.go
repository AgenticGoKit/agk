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
		ref := args[0]

		cm, err := registry.NewCacheManager("")
		if err != nil {
			return err
		}

		// Prefer lookup by manifest name or source (so `agk template remove rag-agent`
		// works, not just the full source path).
		if n, err := cm.RemoveByName(ref); err == nil {
			color.Green("Removed template: %s (%d cached version(s))", ref, n)
			return nil
		}

		// Fall back to treating the argument as a source path and removing all versions.
		if err := cm.Remove(ref, ""); err != nil {
			return fmt.Errorf("template %q not found in cache", ref)
		}

		color.Green("Removed template: %s", ref)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(templateCmd)
	templateCmd.AddCommand(templateListCmd)
	templateCmd.AddCommand(templateAddCmd)
	templateCmd.AddCommand(templateRemoveCmd)
}
