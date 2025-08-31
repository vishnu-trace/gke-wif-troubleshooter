/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "gke-wif-troubleshooter",
	Short: "A CLI tool to troubleshoot Workload Identity Federation for GKE.",
	Long: `gke-wif-troubleshooter is a command-line tool designed to diagnose and
resolve common configuration issues with Workload Identity Federation on Google
Kubernetes Engine (GKE).

It helps you verify that your GKE clusters, Kubernetes Service Accounts, and
Google Service Accounts are correctly configured to allow your GKE workloads to
securely access Google Cloud services.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
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

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.gke-wif-troubleshooter.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
}
