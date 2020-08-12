package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	_ "bazil.org/fuse/fs/fstestutil"
	"github.com/alecthomas/kingpin"
	"github.com/sorenmat/kubefs/pkg/cluster"
	"github.com/sorenmat/kubefs/pkg/kubernetes"
)

func usage() {
	fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s MOUNTPOINT\n", os.Args[0])
	flag.PrintDefaults()
}

var (
	mountpoint = kingpin.Arg("mountpoint", "Where to mount the kubernetes clusters.").Required().String()
	kubeconfig = kingpin.Flag("kubeconfig", "Kuberntes config file to use").Default(filepath.Join(homeDir(), ".kube", "config")).Envar("KUBE_CONFIG").String()
)

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}

func main() {
	kingpin.Parse()

	c, err := fuse.Mount(
		*mountpoint,
		fuse.FSName("KubernetesFS"),
		fuse.Subtype("kubefs"),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		fuse.Unmount(*mountpoint)
		c.Close()
	}()
	listener := make(chan os.Signal)
	signal.Notify(listener, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-listener
		fmt.Println("unmounting")
		fuse.Unmount(*mountpoint)
		c.Close()
		os.Exit(1)
	}()

	// Kubernetes stuff
	x := kubernetes.Config(*kubeconfig)
	clusters := []cluster.Cluster{}
	for k := range x.Clusters {
		clusters = append(clusters, cluster.Cluster{Name: k, Context: x.CurrentContext, Client: kubernetes.Client(*kubeconfig, x.CurrentContext)})
	}
	err = fs.Serve(c, &FS{clusters: clusters})
	if err != nil {
		log.Fatal(err)
	}

	// check if the mount process has an error to report
	<-c.Ready
	if err := c.MountError; err != nil {
		log.Fatal(err)
	}
	fmt.Println("Closing...")
}

// FS implements the hello world file system.
type FS struct {
	clusters []cluster.Cluster
}

func (f *FS) Root() (fs.Node, error) {
	return &ClusterDir{Clusters: f.clusters, Dirent: fuse.Dirent{}}, nil
}

func (f *FS) Statfs(ctx context.Context, req *fuse.StatfsRequest, resp *fuse.StatfsResponse) error {
	fmt.Println("statfs")
	return nil
}
func (f *FS) Attr(ctx context.Context, attr *fuse.Attr) error {
	fmt.Println("FS.Attr")
	return nil
}
func (f *FS) Open(ctx context.Context, req *fuse.OpenRequest, resp *fuse.OpenResponse) (fs.Handle, error) {
	fmt.Println("FS.Open")
	return f, nil
}
