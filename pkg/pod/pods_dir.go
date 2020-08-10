package pod

import (
	"context"
	"os"
	"syscall"
	"time"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"github.com/sorenmat/kubefs/pkg/cluster"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kube "k8s.io/client-go/kubernetes"
	"sigs.k8s.io/yaml"
)

// NamespaceDir implements both Node and Handle for the root directory.
type Dir struct {
	fuse.Dirent
	Cluster     cluster.Cluster
	Namespace   v1.Namespace
	deployments map[string]string
	Client      *kube.Clientset
}

func (d *Dir) GetDirent() fuse.Dirent {
	return d.Dirent
}
func (d *Dir) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Inode = d.Inode
	a.Mode = os.ModeDir | 0555
	a.Mtime = time.Now()
	return nil
}

func (d *Dir) Lookup(ctx context.Context, name string) (fs.Node, error) {
	if d.deployments == nil {
		d.ReadDirAll(ctx) // refresh cache
	}
	if _, ok := d.deployments[name]; !ok {
		return nil, syscall.ENOENT
	}

	deployment, err := d.Client.CoreV1().Pods(d.Namespace.Name).Get(name, metav1.GetOptions{})
	if err != nil {
		panic(err)
	}
	deployment.Kind = "Pod"
	deployment.APIVersion = "v1"
	data, err := yaml.Marshal(deployment)
	if err != nil {
		panic(err.Error())
	}

	return &File{
		content:   string(data),
		name:      name,
		Namespace: d.Namespace.Name,
		cluster:   d.Cluster,
		Modified:  deployment.Status.StartTime.Time,
		Client:    d.Client,
	}, nil
}

// ReadDirAll lists all deployments in a given namespace
func (d *Dir) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	d.deployments = map[string]string{}
	var dirDirs = []fuse.Dirent{}
	deployments, err := d.Client.CoreV1().Pods(d.Namespace.Name).List(metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}
	for _, deployment := range deployments.Items {
		dirDirs = append(dirDirs, fuse.Dirent{Name: deployment.Name, Type: fuse.DT_Dir})
		d.deployments[deployment.Name] = deployment.Name
	}
	return dirDirs, nil
}

func (f *Dir) Create(ctx context.Context, req *fuse.CreateRequest, resp *fuse.CreateResponse) (fs.Node, fs.Handle, error) {
	// Create a dummy file object
	file := &File{
		name:      req.Name,
		Namespace: f.Namespace.Name,
		cluster:   f.Cluster,
		Client:    f.Client,
	}
	return file, file, nil
}

// Remove removes the file from the filesystem and the cluster
func (f *Dir) Remove(ctx context.Context, req *fuse.RemoveRequest) error {
	err := f.Client.CoreV1().Pods(f.Namespace.Name).Delete(req.Name, &metav1.DeleteOptions{})
	if err != nil {
		return err
	}
	return nil
}
