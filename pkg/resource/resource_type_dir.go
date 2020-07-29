package resource

import (
	"context"
	"os"
	"syscall"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"github.com/sorenmat/kubefs/pkg/cluster"
	"github.com/sorenmat/kubefs/pkg/configmap"
	"github.com/sorenmat/kubefs/pkg/deployment"
	"github.com/sorenmat/kubefs/pkg/ingress"
	"github.com/sorenmat/kubefs/pkg/pod"
	"github.com/sorenmat/kubefs/pkg/service"
	v1 "k8s.io/api/core/v1"
	kube "k8s.io/client-go/kubernetes"
)

// NamespaceDir implements both Node and Handle for the root directory.
type ResourceTypeDir struct {
	fuse.Dirent
	Cluster   cluster.Cluster
	Namespace v1.Namespace
	Client    *kube.Clientset
}

func (d *ResourceTypeDir) GetDirent() fuse.Dirent {
	return d.Dirent
}
func (d *ResourceTypeDir) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Inode = d.Inode
	a.Mode = os.ModeDir | 0555
	return nil
}

func (d *ResourceTypeDir) Lookup(ctx context.Context, name string) (fs.Node, error) {
	if name == "pods" {
		return &pod.Dir{Namespace: d.Namespace, Cluster: d.Cluster, Client: d.Client}, nil
	}
	if name == "deployments" {
		return &deployment.Dir{Namespace: d.Namespace, Cluster: d.Cluster, Client: d.Client}, nil
	}
	if name == "services" {
		return &service.Dir{Namespace: d.Namespace, Cluster: d.Cluster, Client: d.Client}, nil
	}
	if name == "ingresses" {
		return &ingress.Dir{Namespace: d.Namespace.Name, Cluster: d.Cluster, Client: d.Client}, nil
	}
	if name == "configmaps" {
		return &configmap.Dir{Namespace: d.Namespace.Name, Cluster: d.Cluster, Client: d.Client}, nil
	}

	return nil, syscall.ENOENT

}

func (d *ResourceTypeDir) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	var dirDirs = []fuse.Dirent{}

	dirDirs = append(dirDirs, fuse.Dirent{Name: "pods", Type: fuse.DT_Dir})
	dirDirs = append(dirDirs, fuse.Dirent{Name: "deployments", Type: fuse.DT_Dir})
	dirDirs = append(dirDirs, fuse.Dirent{Name: "services", Type: fuse.DT_Dir})
	dirDirs = append(dirDirs, fuse.Dirent{Name: "ingresses", Type: fuse.DT_Dir})
	dirDirs = append(dirDirs, fuse.Dirent{Name: "configmaps", Type: fuse.DT_Dir})

	return dirDirs, nil
}
