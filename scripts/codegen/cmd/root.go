package cmd

import (
	"bytes"
	"fmt"
	"os"

	"github.com/pkg/errors"

	"github.com/launchdarkly/terraform-provider-launchdarkly/scripts/codegen/manifestgen"
	"github.com/spf13/cobra"
)

var OUTPUT_PATH string
var ACCESS_TOKEN string
var APP_HOST string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "codegen",
	Short: "This command is used to generate a structured Go map of LaunchDarkly auditlog events hooks configs",
	Long: `This command does the following:
	 1. Fetches integration manifests from https://app.launchdarkly.com/api/v2/integration-manifests
	 2. Unmarshals the output into a custom Go struct
	 3. Generates Go code at the specified output
	 `,

	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Generating LaunchDarkly integration code from manifests API...")
		if ACCESS_TOKEN == "" {
			return errors.New("LAUNCHDARKLY_ACCESS_TOKEN not set")
		}
		manifests, err := manifestgen.FetchManifests(APP_HOST, ACCESS_TOKEN)
		if err != nil {
			return err
		}
		buf := &bytes.Buffer{}
		err = manifestgen.Render(buf, manifests)
		if err != nil {
			panic(err)
		}
		err = os.WriteFile(OUTPUT_PATH, buf.Bytes(), 0644)
		if err != nil {
			panic(err)
		}

		fmt.Println("Done generating code.")
		return nil
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.
	rootCmd.Flags().StringVar(&APP_HOST, "host", "app.launchdarkly.com", "LaunchDarkly app host")
	rootCmd.Flags().StringVarP(&OUTPUT_PATH, "output-path", "o", "", "Output path (required)")
	_ = rootCmd.MarkFlagRequired("output-path")

	ACCESS_TOKEN = os.Getenv("LAUNCHDARKLY_ACCESS_TOKEN")
}
