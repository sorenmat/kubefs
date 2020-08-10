package ingress

import (
	"context"
	"fmt"
	"os"
	"syscall"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"github.com/sorenmat/kubefs/pkg/cluster"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kube "k8s.io/client-go/kubernetes"
	"sigs.k8s.io/yaml"
)

// NamespaceDir implements both Node and Handle for the root directory.
type Dir struct {
	fuse.Dirent
	Cluster   cluster.Cluster
	Namespace string
	pods      map[string]string
	Client    *kube.Clientset
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
	fmt.Println("Ingress.Dir.Lookup")
	if d.pods == nil {
		fmt.Println("Ingress cache empty")
		d.ReadDirAll(ctx) // refresh cache
	}
	if _, ok := d.pods[name]; !ok {
		fmt.Println("no ingress found", d.pods, name)
		return nil, syscall.ENOENT
	}

	obj, err := d.Client.ExtensionsV1beta1().Ingresses(d.Namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		panic(err.Error())
	}
	obj.Kind = "Ingress"
	obj.APIVersion = "networking.k8s.io/v1beta1"
	data, err := yaml.Marshal(obj)
	if err != nil {
		return nil, err
	}

	return File{name: name, content: string(data), cluster: d.Cluster, Namespace: d.Namespace, Client: d.Client}, nil

}

func (d *Dir) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	fmt.Println("Ingress.Dir.ReadAll", d.Namespace)
	var dirDirs = []fuse.Dirent{}
	d.pods = map[string]string{}
	objs, err := d.Client.ExtensionsV1beta1().Ingresses(d.Namespace).List(metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}
	fmt.Println("found ", len(objs.Items))
	for _, obj := range objs.Items {
		dirDirs = append(dirDirs, fuse.Dirent{Name: obj.Name, Type: fuse.DT_Dir})
		d.pods[obj.Name] = obj.Name
	}
	fmt.Println(d.pods)
	return dirDirs, nil
}

func (f *Dir) Create(ctx context.Context, req *fuse.CreateRequest, resp *fuse.CreateResponse) (fs.Node, fs.Handle, error) {
	// Create a dummy file object
	file := &File{
		name:      req.Name,
		Namespace: f.Namespace,
		cluster:   f.Cluster,
		Client:    f.Client,
	}
	return file, file, nil
}

// Remove removes the file from the filesystem and the cluster
func (f *Dir) Remove(ctx context.Context, req *fuse.RemoveRequest) error {
	err := f.Client.NetworkingV1beta1().Ingresses(f.Namespace).Delete(req.Name, &v1.DeleteOptions{})
	if err != nil {
		return err
	}
	return nil
}
