/*
Copyright 2025 Vishnu Udaikumar

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
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
