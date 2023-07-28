package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "cookiecutter",
	Short: "cookiecutter",
	Long:  `cookiecutter`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("USAGE: cookiecutter [command] [options]")
		fmt.Println("commands: strfrydeploy, haproxydeploy")
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	// examples of flags
	//viper.BindPFlag("pubkey", rootCmd.PersistentFlags().Lookup("pubkey"))
	//viper.BindPFlag("file", rootCmd.PersistentFlags().Lookup("file"))
}
