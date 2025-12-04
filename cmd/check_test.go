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
	"fmt"
	"net"
	"os"
	"path/filepath"
	"testing"

	"cloud.google.com/go/container/apiv1/containerpb"
	iampb "cloud.google.com/go/iam/apiv1/iampb"
	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	clientcmd "k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

func TestGetKubeconfig(t *testing.T) {
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)

	t.Run("Success", func(t *testing.T) {
		tempDir := t.TempDir()
		os.Setenv("HOME", tempDir)
		kubeDir := filepath.Join(tempDir, ".kube")
		os.Mkdir(kubeDir, 0755)
		configPath := filepath.Join(kubeDir, "config")
		os.WriteFile(configPath, []byte("test-config"), 0644)

		err := getKubeconfig("")
		assert.NoError(t, err)
		assert.Equal(t, configPath, kubeconfigpath)
	})

	t.Run("NoHomeDir", func(t *testing.T) {
		// Unsetting HOME might not reliably cause UserHomeDir to fail on all OSes
		// but it's a common way to test this scenario.
		os.Unsetenv("HOME")
		// To be more robust, one would need to patch os.UserHomeDir,
		// but we are avoiding code modifications.
		// This test case's effectiveness may vary.
		if _, err := os.UserHomeDir(); err != nil {
			err := getKubeconfig("")
			assert.Error(t, err)
		} else {
			t.Skip("Skipping test: unable to reliably trigger UserHomeDir error")
		}
	})

	t.Run("KubeconfigNotFound", func(t *testing.T) {
		tempDir := t.TempDir()
		os.Setenv("HOME", tempDir)

		err := getKubeconfig("")
		assert.Error(t, err)
	})
}

func TestGetK8sClientset(t *testing.T) {
	cluster := &containerpb.Cluster{
		Endpoint: "localhost:8080",
		MasterAuth: &containerpb.MasterAuth{
			ClusterCaCertificate: "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUVMVENDQXBXZ0F3SUJBZ0lSQU9OUVQ4QXh2OWk0WURpenFrVEJidEV3RFFZSktvWklodmNOQVFFTEJRQXcKTHpFdE1Dc0dBMVVFQXhNa1pHUmlOV0UwWkdRdE1EQTRNUzAwT1RZMUxXRXdNakF0TWpoaE9UazBPVEJpTnpZNApNQ0FYRFRJMU1EVXhNakF5TWpjMU1Gb1lEekl3TlRVd05UQTFNRE15TnpVd1dqQXZNUzB3S3dZRFZRUURFeVJrClpHSTFZVFJrWkMwd01EZ3hMVFE1TmpVdFlUQXlNQzB5T0dFNU9UUTVNR0kzTmpnd2dnR2lNQTBHQ1NxR1NJYjMKRFFFQkFRVUFBNElCandBd2dnR0tBb0lCZ1FEc1p4TllYR2NmN0dJOVA2eFVRanhlYTRMY0N0d01pcmNZUWtVOApRcG82bUFoMkZoKzJvQkZlNFhzb2Z3QWdmSnFWODdEdmlZdC9RV3BiYlovY1drMHBjcnNzaWNjUzV0QXQ5NGhmClBMY09FYkJ6QWJBY1ozbXFHeGtBNjZ5YWlmbnd4RGROYVkxSmRPakRMZnUyd0t5M3lZdXdtUHYycFFob0NQVmEKaUd3bzlIRWt2Uk0yWkdQZzVOdFJVZ2NGb2c5Ukk3a3BVK1NJUkZ6MlRaakNSc0Z1SVl1VERzR0dEaHlTazQxUwpvOVpUN05lMUxaY2k4dWpMNzZNbVRZbXpHblhjcjRsdFdqY3MxYkZCN3RnUFZkZnBVV1dJdDdkNm1BQVhBUjJVCmp5RThZNWF6cmJ3YTl3cVd3N2hHVG9WZkhtdUFvdTUzQ3F6VkxjSEdlYnJnN3J2MGU0MHQxcncySGtCaG55bGkKRFUwc3R4SlNEeEhoU1A0aWRHQVkwaG5WVUJpTU9HdUVWeWpKaFlqS3h4d2JOR21zN2loZ3RtdFBnb1ZMRmFDZQpjRlFiQ0pKcFVZbDZFZ01jMkQyMVJyRUZXR1V4SlFEbnZwQ0g1TTViTTQxRlBNUEJqMVc1TzRGMmY1Wjc1ajdMCnNhazU5Y0FIam9FNDNKczlwenMxbHZsTDc5MENBd0VBQWFOQ01FQXdEZ1lEVlIwUEFRSC9CQVFEQWdJRU1BOEcKQTFVZEV3RUIvd1FGTUFNQkFmOHdIUVlEVlIwT0JCWUVGRDhGQzhCOXlHdTdqeE5yZGlQQUZMSzlVSXhvTUEwRwpDU3FHU0liM0RRRUJDd1VBQTRJQmdRQklHbUZrTStkcDRqYXFCK3RMTkdPTHZLWUJpaDFpemdhVzZWenFoUkl3CnlJQ1ZNYno1MzY2aW1ySjkxMVdaQ3o1U3NrdXRGQVJ0VTVGd01pVjZBTjFxUExmbndyMHdmTkVkUFJEUGovWEsKYzNnbEl6RGFkNnZVYkxZRmxlVDBPMnk4TEVmaVhPMzNYNXNOTHNsQkxXNCttenBUWHYwTnRveGRRS3lxWDE1cwo5bWcxalEwQzFDRDZzdHJ3RU9PcGZqcGRpSUZBZHYzMkVKSExPbjN1d2xraURpOTlQUks1NTV6NE5sS3VrTG9HCk9mVUorNmdsaGltZ1A2LytSWnY4eE92M2R2bHZqckZKUndiUS8rTnNhUFRwTlRyMk5tenFqdDdUU2FubXBsMzgKYXJ4NEh5T1Z3YTZaWS92V0l1R2pROUt3d0dPU3IrTGtyU3l1Smw1Rm9TVll1MkZuYnNPQVJObEhXYXNlNC8rTwp5eFZVMHU2ckREVG4zRGtjc09KSUluOXNOU3dRUW9zbXB4MC9peGtIa2JnejI5bklwVlAyV0lacUxRSFFBeEsxClh3ZGZaZ3JCdTF5N0M0ejRNNlRhT3ZFT0I1YlR5YzE3TExJN2RhcHpJWCsyV09UN1dCTDBJNzRxVGJBSHN2TVQKOHZQZWk0cFNMK0hOMm5RQmxEK2JmWTQ9Ci0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K",
		},
	}

	t.Run("FromClusterConfig", func(t *testing.T) {
		kubeconfigpath = ""
		clientset, err := getK8sClientset(cluster)
		assert.NoError(t, err)
		assert.NotNil(t, clientset)
	})

	t.Run("FromKubeconfigPath", func(t *testing.T) {
		cfg := clientcmdapi.Config{
			Clusters: map[string]*clientcmdapi.Cluster{
				"test-cluster": {
					Server: "https://localhost",
				},
			},
			Contexts: map[string]*clientcmdapi.Context{
				"test-context": {
					Cluster: "test-cluster",
				},
			},
			CurrentContext: "test-context",
		}
		tmpfile, err := os.CreateTemp("", "kubeconfig")
		assert.NoError(t, err)
		defer os.Remove(tmpfile.Name())

		err = clientcmd.WriteToFile(cfg, tmpfile.Name())
		assert.NoError(t, err)

		kubeconfigpath = tmpfile.Name()
		clientset, err := getK8sClientset(cluster)
		assert.NoError(t, err)
		assert.NotNil(t, clientset)
	})

	t.Run("FromKubeconfigPathError", func(t *testing.T) {
		kubeconfigpath = "/path/to/non/existent/config"
		_, err := getK8sClientset(cluster)
		assert.Error(t, err)
	})

	t.Run("InvalidBase64Cert", func(t *testing.T) {
		kubeconfigpath = ""
		invalidCluster := &containerpb.Cluster{
			Endpoint: "localhost:8080",
			MasterAuth: &containerpb.MasterAuth{
				ClusterCaCertificate: "invalid-base64",
			},
		}
		_, err := getK8sClientset(invalidCluster)
		assert.Error(t, err)
	})
}

func TestPerformKsaCheck(t *testing.T) {
	ctx := context.Background()
	projectID = "test-project"
	ksaNamespace := "default"
	ksaName := "test-ksa"

	clusterWithWI := &containerpb.Cluster{
		Name:     "test-cluster",
		Location: "us-central1",
		WorkloadIdentityConfig: &containerpb.WorkloadIdentityConfig{
			WorkloadPool: "test-project.svc.id.goog",
		},
	}

	clusterWithoutWI := &containerpb.Cluster{
		Name:     "test-cluster-no-wi",
		Location: "us-central1",
	}

	// Mock IAM and ResourceManager clients are needed.
	// Since we cannot modify the source to inject mocks, we can't fully test these.
	// We will test the logic branches that don't require live clients.

	t.Run("WI not enabled", func(t *testing.T) {
		err := performKsaCheck(ctx, ksaNamespace, ksaName, clusterWithoutWI, fake.NewSimpleClientset())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Workload Identity is not enabled")
	})

	t.Run("KSA not found", func(t *testing.T) {
		clientset := fake.NewSimpleClientset()
		err := performKsaCheck(ctx, ksaNamespace, ksaName, clusterWithWI, clientset)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get Kubernetes Service Account")
	})
}

func TestGetTokenFromConfig(t *testing.T) {
	ctx := context.Background()

	t.Run("NoAccessToken", func(t *testing.T) {
		accessToken = ""
		tokenSource := getTokenFromConfig(ctx)
		assert.Nil(t, tokenSource)
	})

	t.Run("WithAccessToken", func(t *testing.T) {
		testToken := "test-access-token"
		accessToken = testToken
		defer func() { accessToken = "" }()

		tokenSource := getTokenFromConfig(ctx)
		assert.NotNil(t, tokenSource)

		token, err := tokenSource.Token()
		assert.NoError(t, err)
		assert.Equal(t, testToken, token.AccessToken)
		assert.IsType(t, &oauth2.Token{}, token)
	})
}

func TestGetClientOptions(t *testing.T) {
	ctx := context.Background()

	t.Run("NoTokens", func(t *testing.T) {
		accessToken = ""
		inspectionToken = ""
		opts := getClientOptions(ctx)
		// Expecting 2 options: TokenSource (which will be nil), and GRPCDialOption
		assert.Len(t, opts, 2)
	})

	t.Run("WithAccessToken", func(t *testing.T) {
		accessToken = "some-access-token"
		inspectionToken = ""
		defer func() { accessToken = "" }()

		opts := getClientOptions(ctx)
		assert.Len(t, opts, 2)
		// Further inspection would require reflection or more complex checks
	})

	t.Run("WithInspectionToken", func(t *testing.T) {
		accessToken = ""
		inspectionToken = "some-inspection-token"
		defer func() { inspectionToken = "" }()

		opts := getClientOptions(ctx)
		assert.Len(t, opts, 2)
	})

	t.Run("WithBothTokens", func(t *testing.T) {
		accessToken = "some-access-token"
		inspectionToken = "some-inspection-token"
		defer func() {
			accessToken = ""
			inspectionToken = ""
		}()

		opts := getClientOptions(ctx)
		assert.Len(t, opts, 2)
	})
}

// The following tests require live or mocked GCP services.
// Since we cannot modify the source to inject mocks, these tests are limited.

type mockClusterManagerServer struct {
	containerpb.UnimplementedClusterManagerServer
	Cluster *containerpb.Cluster
	Err     error
}

func (s *mockClusterManagerServer) GetCluster(ctx context.Context, req *containerpb.GetClusterRequest) (*containerpb.Cluster, error) {
	if s.Err != nil {
		return nil, s.Err
	}
	return s.Cluster, nil
}

// Mock IAMPolicy server
type mockIAMPolicyServer struct {
	iampb.UnimplementedIAMPolicyServer
	Policy *iampb.Policy
	Err    error
}

func (s *mockIAMPolicyServer) GetIamPolicy(ctx context.Context, req *iampb.GetIamPolicyRequest) (*iampb.Policy, error) {
	if s.Err != nil {
		return nil, s.Err
	}
	return s.Policy, nil
}

func (s *mockIAMPolicyServer) SetIamPolicy(ctx context.Context, req *iampb.SetIamPolicyRequest) (*iampb.Policy, error) {
	if s.Err != nil {
		return nil, s.Err
	}
	s.Policy = req.GetPolicy()
	return s.Policy, nil
}

func (s *mockIAMPolicyServer) TestIamPermissions(ctx context.Context, req *iampb.TestIamPermissionsRequest) (*iampb.TestIamPermissionsResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

func newMockClientset(ksa *corev1.ServiceAccount) kubernetes.Interface {
	return fake.NewSimpleClientset(ksa)
}

func newMockGcpClients(ctx context.Context, t *testing.T, iamPolicy *iampb.Policy, iamErr error, rmPolicy *iampb.Policy, rmErr error) (
	*grpc.ClientConn, func()) {

	rmServer := &mockIAMPolicyServer{Policy: rmPolicy, Err: rmErr}

	rmLis, rmConn := startMockServer(t, func(s *grpc.Server) {
		iampb.RegisterIAMPolicyServer(s, rmServer)
	})

	cleanup := func() {
		rmConn.Close()
		rmLis.Close()
	}

	return rmConn, cleanup
}

func startMockServer(t *testing.T, register func(s *grpc.Server)) (net.Listener, *grpc.ClientConn) {
	lis, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("Failed to listen: %v", err)
	}
	s := grpc.NewServer()
	register(s)
	go s.Serve(lis)

	conn, err := grpc.NewClient(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("Failed to dial: %v", err)
	}
	return lis, conn
}

func getMockClientOptions(ctx context.Context, conn *grpc.ClientConn) []option.ClientOption {
	return []option.ClientOption{
		option.WithGRPCConn(conn),
	}
}
