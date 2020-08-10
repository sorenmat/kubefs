package namespace

import (
	"context"
	"fmt"
	"os"
	"syscall"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"github.com/sorenmat/kubefs/pkg/cluster"
	"github.com/sorenmat/kubefs/pkg/resource"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kube "k8s.io/client-go/kubernetes"
)

// NamespaceDir represents the list of namespaces in the Cluster
type Dir struct {
	fuse.Dirent
	Cluster    cluster.Cluster
	Namespaces []string
	Resources  []*resource.ResourceTypeDir
	Client     *kube.Clientset
}

func (d *Dir) GetDirent() fuse.Dirent {
	return d.Dirent
}

func (d *Dir) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Inode = d.Inode
	a.Mode = os.ModeDir | 0555
	return nil
}

func (d *Dir) Lookup(ctx context.Context, name string) (fs.Node, error) {
	if d.Resources == nil {
		d.ReadDirAll(ctx) // try to re-populate the 'cache'
	}
	for _, v := range d.Resources {
		if v.Name == name {
			return v, nil
		}
	}
	return nil, syscall.ENOENT
}

func (d *Dir) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	namespaces, err := d.Client.CoreV1().Namespaces().List(v1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}
	var dirDirs = []fuse.Dirent{}
	for _, ns := range namespaces.Items {
		fmt.Println(ns.Name)

		x := &resource.ResourceTypeDir{Dirent: fuse.Dirent{Name: ns.Name, Type: fuse.DT_Dir}, Cluster: d.Cluster, Namespace: ns, Client: d.Client}
		d.Resources = append(d.Resources, x)
		dirDirs = append(dirDirs, x.GetDirent())

	}
	return dirDirs, nil
}
