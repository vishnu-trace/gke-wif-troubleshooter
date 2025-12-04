/*
Copyright 2025 Vishnu Udaikumar

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is a "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/spf13/cobra"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var (
	workloadNamespace string
	workloadType      string
)

// workloadCmd represents the workload command
var workloadCmd = &cobra.Command{
	Use:   "workload <workload-name>",
	Short: "Checks the Workload Identity configuration for a given Kubernetes workload.",
	Long: `Analyzes a Kubernetes workload (e.g., Deployment, StatefulSet, CronJob) to verify its Workload Identity setup.

	It performs the following checks:
		- Identifies the Kubernetes Service Account (KSA) used by the workload and then performs all the necessary checks on that KSA.
		- Checks for known configuration issues
		`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		workloadName := args[0]
		ctx := context.Background()

		gkeClient, err := newGKEClient(ctx)

		if err != nil {
			log.Fatalf("❌ Failed to create GKE client: %v", err)
		}
		defer gkeClient.Close()

		cluster, err := getGKECluster(ctx, gkeClient, projectID, location, clusterName)
		if err != nil {
			log.Fatalf("❌ Failed to get GKE cluster details: %v", err)
		}

		clientset, err := getK8sClientset(cluster)
		if err != nil {
			log.Fatalf("❌ Failed to create Kubernetes clientset: %v", err)
		}

		ksaName, err := getKsaFromWorkload(ctx, clientset, workloadNamespace, workloadName, workloadType)
		if err != nil {
			log.Fatalf("❌ Failed to get KSA from workload: %v", err)
		}

		fmt.Printf("ℹ️ Workload '%s/%s' is using Kubernetes Service Account '%s'.\n\n", workloadNamespace, workloadName, ksaName)

		if err := performKsaCheck(ctx, workloadNamespace, ksaName, cluster, clientset); err != nil {
			log.Fatalf("❌ Check failed for KSA '%s': %v", ksaName, err)
		}
	},
}

func getKsaFromWorkload(ctx context.Context, clientset kubernetes.Interface, namespace, name, wType string) (string, error) {
	var serviceAccountName string
	var err error

	switch strings.ToLower(wType) {
	case "deployment", "deploy":
		var workload *appsv1.Deployment
		workload, err = clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
		if err == nil {
			serviceAccountName = workload.Spec.Template.Spec.ServiceAccountName
		}
	case "statefulset", "sts":
		var workload *appsv1.StatefulSet
		workload, err = clientset.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
		if err == nil {
			serviceAccountName = workload.Spec.Template.Spec.ServiceAccountName
		}
	case "daemonset", "ds":
		var workload *appsv1.DaemonSet
		workload, err = clientset.AppsV1().DaemonSets(namespace).Get(ctx, name, metav1.GetOptions{})
		if err == nil {
			serviceAccountName = workload.Spec.Template.Spec.ServiceAccountName
		}
	case "job":
		var workload *batchv1.Job
		workload, err = clientset.BatchV1().Jobs(namespace).Get(ctx, name, metav1.GetOptions{})
		if err == nil {
			serviceAccountName = workload.Spec.Template.Spec.ServiceAccountName
		}
	case "cronjob", "cj":
		var workload *batchv1.CronJob
		workload, err = clientset.BatchV1().CronJobs(namespace).Get(ctx, name, metav1.GetOptions{})
		if err == nil {
			serviceAccountName = workload.Spec.JobTemplate.Spec.Template.Spec.ServiceAccountName
		}
	default:
		return "", fmt.Errorf("unsupported workload type '%s'", wType)
	}

	if err != nil {
		return "", fmt.Errorf("could not get workload '%s/%s' of type '%s': %w", namespace, name, wType, err)
	}

	// If the service account is not specified in the pod spec, it defaults to "default".
	if serviceAccountName == "" {
		return "default", nil
	}

	return serviceAccountName, nil
}

func init() {
	checkCmd.AddCommand(workloadCmd)
	workloadCmd.Flags().StringVarP(&workloadNamespace, "namespace", "n", "default", "Kubernetes namespace of the workload")
	workloadCmd.Flags().StringVarP(&workloadType, "type", "t", "deployment", "Type of the workload (deployment, statefulset, daemonset, job, cronjob)")
}
