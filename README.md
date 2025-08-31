# GKE Workload Identity Troubleshooter

`gke-wif-troubleshooter` is a command-line tool designed to diagnose and resolve common configuration issues with Workload Identity Federation on Google Kubernetes Engine (GKE).

It helps you verify that your GKE clusters, Kubernetes Service Accounts (KSAs), and Google Service Accounts (GSAs) are correctly configured to allow your GKE workloads to securely access Google Cloud services.

## Prerequisites

Before using this tool, ensure you have the following:

1.  **Go:** Version 1.23 or later installed.
2.  **Google Cloud SDK (`gcloud`):** Installed and authenticated. You will need to be logged in for both user and application-default credentials.
    ```bash
    gcloud auth login
    gcloud auth application-default login
    ```
3.  **GKE Auth Plugin:** The tool requires the `gke-gcloud-auth-plugin` to authenticate with your GKE cluster. This is typically installed with the `gcloud` CLI.

## Installation

You can install the tool using `go install`:

```bash
go install github.com/vishnu-trace/gke-wif-troubleshooter@latest
```

Alternatively, you can clone the repository and build it from source:

```bash
git clone https://github.com/vishnu-trace/gke-wif-troubleshooter.git
cd gke-wif-troubleshooter
go build .
```

## Usage

The tool provides commands to check the Workload Identity configuration for Kubernetes Service Accounts and various workloads.

All `check` subcommands require the GKE cluster details:
*   `--project`: Your GCP Project ID.
*   `--location`: The region or zone of your GKE cluster.
*   `--cluster`: The name of your GKE cluster.

### Check a Kubernetes Service Account (KSA)

This command analyzes a specific KSA to verify its Workload Identity setup.

```bash
gke-wif-troubleshooter check ksa <KSA_NAME> \
  --namespace <NAMESPACE> \
  --project <PROJECT_ID> \
  --location <CLUSTER_LOCATION> \
  --cluster <CLUSTER_NAME>
```

**Example:**

```bash
gke-wif-troubleshooter check ksa my-app-ksa \
  --namespace my-app-ns \
  --project my-gcp-project \
  --location us-central1 \
  --cluster my-gke-cluster
```

### Check a Kubernetes Workload

This command analyzes a Kubernetes workload (like a Deployment, StatefulSet, etc.) to find its KSA and then performs the same checks.

```bash
gke-wif-troubleshooter check workload <WORKLOAD_NAME> \
  --type <WORKLOAD_TYPE> \
  --namespace <NAMESPACE> \
  --project <PROJECT_ID> \
  --location <CLUSTER_LOCATION> \
  --cluster <CLUSTER_NAME>
```

Supported workload types: `deployment`, `statefulset`, `daemonset`, `job`, `cronjob`.

**Example:**

```bash
gke-wif-troubleshooter check workload my-deployment \
  --type deployment \
  --namespace my-app-ns \
  --project my-gcp-project \
  --location us-central1 \
  --cluster my-gke-cluster
```

## What It Checks

The troubleshooter performs a series of validations:

1.  **Cluster Configuration:**
    *   Verifies that Workload Identity is enabled on the specified GKE cluster.

2.  **Kubernetes Service Account (KSA):**
    *   Confirms that the KSA exists in the specified namespace.
    *   Checks for the `iam.gke.io/gcp-service-account` annotation, which links the KSA to a Google Service Account (GSA).

3.  **IAM Bindings:**
    *   **If the KSA is annotated:** It verifies that the GSA has an IAM policy binding with the `roles/iam.workloadIdentityUser` role for the KSA's principal.
    *   **If the KSA is NOT annotated:** It checks if the KSA's principal has been granted IAM roles directly at the project level.

The tool provides clear success messages or actionable error messages to help you fix any detected issues.