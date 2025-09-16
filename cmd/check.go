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
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	container "cloud.google.com/go/container/apiv1"
	"github.com/spf13/cobra"
	"google.golang.org/api/cloudresourcemanager/v1"
	"google.golang.org/api/iam/v1"
	containerpb "google.golang.org/genproto/googleapis/container/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/exec"
	"k8s.io/client-go/rest"
	clientcmd "k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

var (
	projectID      string
	location       string
	clusterName    string
	kubeconfigpath string
	kubeconfig     map[string]interface{}
)

// checkCmd represents the check command
var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Checks the Workload Identity configuration for various resources.",
	Long: `Performs a series of checks to validate the Workload Identity setup
for resources like Kubernetes Service Accounts (KSA) and workloads (Deployments, etc.).`,
	// This is a parent command, so it doesn't have a Run function.
}

func init() {
	rootCmd.AddCommand(checkCmd)

	checkCmd.PersistentFlags().StringVar(&projectID, "project", "", "GCP project ID (required)")
	checkCmd.PersistentFlags().StringVar(&location, "location", "", "GKE cluster location (region or zone) (required)")
	checkCmd.PersistentFlags().StringVar(&clusterName, "cluster", "", "GKE cluster name (required)")
	checkCmd.PersistentFlags().BoolFunc("local-kubeconfig", "Use local GKE cluster kubeconfig (optional)", getKubeconfig)

	checkCmd.MarkPersistentFlagRequired("project")
	checkCmd.MarkPersistentFlagRequired("location")
	checkCmd.MarkPersistentFlagRequired("cluster")
}

// generate kubeconfig path if --local-kubeconfig flag used
func getKubeconfig(string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Errorf("❌ Failed to retrieve home directory: %v", err)
		return err
	}
	kubeconfigpath = filepath.Join(homeDir, ".kube", "config")

	_, err = os.Stat(kubeconfigpath)
	if err != nil {
		fmt.Errorf("❌ Failed to find local kubeconfig at: %v/n. More details %w", kubeconfigpath, err)
		return err
	}
	return nil
}

// getGKECluster retrieves GKE cluster details.
func getGKECluster(ctx context.Context, client *container.ClusterManagerClient, project, location, cluster string) (*containerpb.Cluster, error) {
	req := &containerpb.GetClusterRequest{
		Name: fmt.Sprintf("projects/%s/locations/%s/clusters/%s", project, location, cluster),
	}
	return client.GetCluster(ctx, req)
}

// getK8sClientset creates a Kubernetes clientset from GKE cluster data.
func getK8sClientset(cluster *containerpb.Cluster) (*kubernetes.Clientset, error) {
	var config *rest.Config
	if kubeconfigpath == "" {
		caDec, err := base64.StdEncoding.DecodeString(cluster.MasterAuth.ClusterCaCertificate)
		if err != nil {
			return nil, fmt.Errorf("failed to decode cluster CA certificate: %w", err)
		}

		config = &rest.Config{
			Host: "https://" + cluster.Endpoint,
			TLSClientConfig: rest.TLSClientConfig{
				CAData: caDec,
			},
			ExecProvider: &clientcmdapi.ExecConfig{
				APIVersion:         "client.authentication.k8s.io/v1beta1",
				Command:            "gke-gcloud-auth-plugin",
				ProvideClusterInfo: true,
				InteractiveMode:    "Never",
			},
		}
	} else {
		var err error
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfigpath)
		if err != nil {
			return nil, fmt.Errorf("failed to build config from kubeconfig: %w", err)
		}
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create clientset from config: %w", err)
	}
	return clientset, nil
}

// performKsaCheck carries out the actual validation for a given KSA.
func performKsaCheck(ctx context.Context, ksaNamespace, ksaName string, cluster *containerpb.Cluster, clientset *kubernetes.Clientset) error {
	fmt.Printf("🔎 Starting GKE Workload Identity analysis for KSA: %s/%s\n", ksaNamespace, ksaName)
	fmt.Println("-------------------------------------------------------------")

	// 1. Check GKE cluster for Workload Identity
	fmt.Printf("1. Checking cluster '%s' in '%s'...\n", cluster.Name, cluster.Location)

	if cluster.WorkloadIdentityConfig == nil || cluster.WorkloadIdentityConfig.WorkloadPool == "" {
		return fmt.Errorf("Workload Identity is not enabled on cluster '%s'", cluster.Name)
	}
	workloadPool := cluster.WorkloadIdentityConfig.WorkloadPool
	fmt.Printf("   ✅ Workload Identity is enabled. Workload Pool: %s\n", workloadPool)

	// 2. Check K8s Service Account and annotation
	fmt.Printf("\n2. Checking K8s Service Account '%s/%s'...\n", ksaNamespace, ksaName)

	ksa, err := clientset.CoreV1().ServiceAccounts(ksaNamespace).Get(ctx, ksaName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get Kubernetes Service Account '%s' in namespace '%s': %w", ksaName, ksaNamespace, err)
	}
	fmt.Printf("   ✅ Found KSA '%s/%s'.\n", ksaNamespace, ksaName)

	gsaAnnotation := "iam.gke.io/gcp-service-account"
	gsaEmail, ok := ksa.Annotations[gsaAnnotation]

	iamService, err := iam.NewService(ctx)
	if err != nil {
		return fmt.Errorf("failed to create IAM client: %w", err)
	}

	legacySyntax := fmt.Sprintf("serviceAccount:%s.svc.id.goog[%s/%s]", projectID, ksaNamespace, ksaName)
	principalSchema := fmt.Sprintf("%s.svc.id.goog/subject/ns/%s/sa/%s", projectID, ksaNamespace, ksaName)

	// 3. Check IAM binding

	if !ok || gsaEmail == "" {
		fmt.Printf("   ℹ️  KSA '%s/%s' is missing the '%s' annotation.\n", ksaNamespace, ksaName, gsaAnnotation)
		fmt.Println("   ℹ️  This is not necessarily an error. Checking for direct IAM role bindings on the KSA principal...")

		fmt.Println("\n3. Checking for direct IAM bindings for KSA principal at the project level...")
		crmService, err := cloudresourcemanager.NewService(ctx)
		if err != nil {
			return fmt.Errorf("failed to create Cloud Resource Manager client: %w", err)
		}

		policy, err := crmService.Projects.GetIamPolicy(projectID, &cloudresourcemanager.GetIamPolicyRequest{}).Do()
		if err != nil {
			return fmt.Errorf("failed to get IAM policy for project '%s': %w", projectID, err)
		}

		foundMember := ""
		for _, binding := range policy.Bindings {
			for _, m := range binding.Members {
				if strings.Contains(m, principalSchema) || strings.Contains(m, legacySyntax) {
					foundMember = m
					break
				}
			}
		}

		if foundMember == "" {
			fmt.Printf("   ❌ No direct IAM bindings found for KSA principal at the project level ('%s').\n", projectID)
			fmt.Println("   ℹ️  This is not necessarily an error if the principal is assigned role directly on the product.")
			fmt.Println("   ℹ️  If your workload needs permissions at the project level, you should either:")
			fmt.Println("	  1. Grant IAM roles directly to the KSA principal on the project level (recommended).\n		The principal syntax could be found at https://cloud.google.com/kubernetes-engine/docs/concepts/workload-identity#kubernetes-resources-iam-policies")
			fmt.Printf("	  2. Annotate the KSA '%s/%s' to impersonate a GSA .\n", ksaNamespace, ksaName)

		} else {
			fmt.Printf("   ✅ Found direct IAM bindings for KSA principal '%s' at the project level.\n", foundMember)
			fmt.Println("\n🎉 Checks passed! The KSA has direct IAM role bindings at the project level.")
			fmt.Println("   Please ensure these roles provide the necessary permissions for your workload to function.")
		}
	} else {
		fmt.Printf("   ✅ KSA is annotated with GSA: %s\n", gsaEmail)

		fmt.Printf("\n3. Checking IAM binding for GSA '%s'...\n", gsaEmail)
		iamPolicy, err := iamService.Projects.ServiceAccounts.GetIamPolicy("projects/-/serviceAccounts/" + gsaEmail).Do()
		if err != nil {
			return fmt.Errorf("failed to get IAM policy for GSA '%s' (does it exist?): %w", gsaEmail, err)
		}

		role := "roles/iam.workloadIdentityUser"
		bindingFound := false
		for _, binding := range iamPolicy.Bindings {
			if binding.Role == role {
				for _, m := range binding.Members {
					if m == legacySyntax {
						bindingFound = true
						break
					}
				}
			}
			if bindingFound {
				break
			}
		}

		if !bindingFound {
			return fmt.Errorf("IAM binding not found. Run the following command to fix:\n\ngcloud iam service-accounts add-iam-policy-binding %s \\\n  --role=roles/iam.workloadIdentityUser \\\n  --member=\"serviceAccount:%s.svc.id.goog[%s/%s]\"", gsaEmail, projectID, ksaNamespace, ksaName)
		}
		fmt.Printf("   ✅ Found IAM binding for member '%s' with role '%s'.\n", legacySyntax, role)

		fmt.Println("-------------------------------------------------------------")
		fmt.Println("🎉 All checks passed! Your Workload Identity setup seems correct for this KSA.")
	}
	return nil
}
