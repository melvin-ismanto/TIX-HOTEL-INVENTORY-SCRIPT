package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

func Execute() {
	// rootCmd represents the base command when called without any subcommands
	rootCmd := &cobra.Command{
		Use:     "TIX-HOTEL-INVENTORY-SCRIPT",
		Short:   "List of script for Inventory team.",
		Example: "roomFiltering rollout:sync",
		RunE: func(cmd *cobra.Command, args []string) error {

			cobra.OnInitialize(initConfig)

			return nil
		}}

	rootCmd.AddCommand(cmdStart())

	err := rootCmd.ExecuteContext(context.Background())

	if err != nil {
		fmt.Printf("An error occured on root: %v", err)
		os.Exit(1)
	}
}

func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find start directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in start directory with name ".TIX-HOTEL-INVENTORY-SCRIPT" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".TIX-HOTEL-INVENTORY-SCRIPT")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}
