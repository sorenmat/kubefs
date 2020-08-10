package cluster

import "k8s.io/client-go/kubernetes"

type Cluster struct {
	Name    string
	Context string
	// somehow save the client
	Client *kubernetes.Clientset
}
