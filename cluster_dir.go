package main

import (
	"context"
	"fmt"
	"os"
	"syscall"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"github.com/sorenmat/kubefs/pkg/cluster"
	"github.com/sorenmat/kubefs/pkg/namespace"
)

// ClusterDir implements both Node and Handle for the root directory.
type ClusterDir struct {
	fuse.Dirent
	Clusters   []cluster.Cluster
	Namespaces []*namespace.Dir
}

func (d *ClusterDir) Attr(ctx context.Context, a *fuse.Attr) error {
	fmt.Println("Cluster.Attr")
	a.Inode = d.Inode
	a.Mode = os.ModeDir | 0555
	return nil
}

func (d *ClusterDir) Lookup(ctx context.Context, name string) (fs.Node, error) {
	if d.Namespaces == nil {
		d.ReadDirAll(ctx) // try to re-populate the 'cache'
	}
	for _, v := range d.Namespaces {
		if v.Name == name {
			return v, nil
		}
	}

	return nil, syscall.ENOENT
}

func (d *ClusterDir) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	var dirDirs = []fuse.Dirent{}
	fmt.Println("Clusters readdirall: ", d.Clusters)
	for _, v := range d.Clusters {

		x := &namespace.Dir{Dirent: fuse.Dirent{Name: v.Name, Type: fuse.DT_Dir}, Cluster: v, Client: v.Client}
		d.Namespaces = append(d.Namespaces, x)
		dirDirs = append(dirDirs, x.GetDirent())

	}
	return dirDirs, nil
}
