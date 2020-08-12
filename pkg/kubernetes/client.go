package kubernetes

import (
	"fmt"
	"os"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

func FindContext(server string, x *api.Config) string {
	for k, v := range x.Contexts {
		if v.Cluster == server {
			return k
		}
	}
	return ""
}

func Config(configfile string) *api.Config {

	x, err := clientcmd.LoadFromFile(configfile)
	if err != nil {
		panic(err.Error())
	}
	return x
}

func Client(configfile string, context string) *kubernetes.Clientset {
	fmt.Println("Getting client")

	x, err := clientcmd.LoadFromFile(configfile)
	if err != nil {
		panic(err.Error())
	}

	config, err := clientcmd.NewInteractiveClientConfig(*x, context, &clientcmd.ConfigOverrides{}, nil, nil).ClientConfig()
	if err != nil {
		panic(err.Error())
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	return clientset
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}
