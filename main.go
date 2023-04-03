package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	v1alpha1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	argo "github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

func getClientset() (*argo.Clientset, error) {
	var config *rest.Config
	var err error
	// Check if the program is running inside a Kubernetes cluster.
	if _, err = rest.InClusterConfig(); err != nil {
		kubeconfig := os.Getenv("KUBECONFIG")
		if kubeconfig == "" {
			kubeconfig = clientcmd.RecommendedHomeFile
		}

		config, err = clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
			&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfig},
			&clientcmd.ConfigOverrides{ClusterInfo: clientcmdapi.Cluster{Server: ""}}).ClientConfig()
		if err != nil {
			log.Fatalf("Failed to load Kubernetes configuration: %v", err)
		}
	} else {
		config, err = rest.InClusterConfig()
		if err != nil {
			log.Fatalf("Failed to load in-cluster configuration: %v", err)
		}
	}

	//return kubernetes.NewForConfig(config)
	return argo.NewForConfig(config)

}

func deleteCronWorkflow(argoClientset *argo.Clientset, ctx context.Context, namespace, name string) error {
	err := argoClientset.ArgoprojV1alpha1().CronWorkflows(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		return err
	}
	return nil
}

func main() {

	argoClientset, err := getClientset()
	if err != nil {
		log.Fatalf("Error creating Argo clientset: %v", err)
	}

	ctx := context.Background()

	cronWorkflow := &v1alpha1.CronWorkflow{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "hello-world-cron-",
		},
		Spec: v1alpha1.CronWorkflowSpec{
			Schedule:          "* * * * *", // Run every minute
			ConcurrencyPolicy: "Forbid",
			WorkflowSpec: v1alpha1.WorkflowSpec{
				Entrypoint: "hello-world",
				Templates: []v1alpha1.Template{
					{
						Name: "hello-world",
						Script: &v1alpha1.ScriptTemplate{
							Container: apiv1.Container{
								Image:   "python:alpine3.6",
								Command: []string{"python"},
							},
							Source: "print('Hello, world!')",
						},
					},
				},
			},
		},
	}

	result, err := argoClientset.ArgoprojV1alpha1().CronWorkflows("default").Create(ctx, cronWorkflow,
		metav1.CreateOptions{})
	if err != nil {
		log.Fatalf("Error creating Argo Cron Workflow: %v", err)
	}

	fmt.Printf("Cron Workflow created: %s", result.Name)

	fmt.Println("Cron Workflow created successfully.")

	for i := 0; i < 4; i++ {
		time.Sleep(30 * time.Second)
		workflows, err := argoClientset.ArgoprojV1alpha1().Workflows("default").List(ctx, metav1.ListOptions{})
		if err != nil {
			log.Printf("Error listing Argo Workflows: %v", err)
			continue
		}

		fmt.Println("Workflows:")
		for _, wf := range workflows.Items {
			fmt.Printf(" - Name: %s, Status: %s\n", wf.Name, wf.Status.Phase)
		}
	}

	err = deleteCronWorkflow(argoClientset, ctx, "default", result.Name)
	if err != nil {
		log.Fatalf("Error deleting Argo Cron Workflow: %v", err)
	}

	fmt.Println("Cron Workflow deleted successfully.")

}
