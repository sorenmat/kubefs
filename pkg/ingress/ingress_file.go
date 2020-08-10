package ingress

import (
	"bytes"
	"context"
	"fmt"
	"time"

	k8yaml "k8s.io/apimachinery/pkg/util/yaml"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"github.com/sorenmat/kubefs/pkg/cluster"
	"k8s.io/api/networking/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	kube "k8s.io/client-go/kubernetes"
	"sigs.k8s.io/yaml"
)

// File implements both Node and Handle for the hello file.
type File struct {
	fs.Node
	content   string
	cluster   cluster.Cluster
	Namespace string
	name      string
	Modified  time.Time
	Client    *kube.Clientset
	NotKube   bool
}

func (f File) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Mode = 0644
	a.Size = uint64(len(f.content))
	return nil
}

func (f File) ReadAll(ctx context.Context) ([]byte, error) {
	fmt.Println("Ingress.File.ReadAll,", f.name)
	return []byte(f.content), nil
}
func (f *File) reread() {
	fmt.Println("file.reread")
	deployment, err := f.Client.ExtensionsV1beta1().Ingresses(f.Namespace).Get(f.name, metav1.GetOptions{})
	if err != nil {
		panic(err)
	}

	data, err := yaml.Marshal(deployment)
	if err != nil {
		panic(err.Error())
	}
	f.content = string(data)
}
func (f *File) Open(ctx context.Context, req *fuse.OpenRequest, resp *fuse.OpenResponse) (fs.Handle, error) {
	fmt.Println("Ingress.File.Open,", f.name)
	f.reread()
	return f, nil
}

func (f *File) Write(ctx context.Context, req *fuse.WriteRequest, resp *fuse.WriteResponse) error {
	d, err := f.Client.NetworkingV1beta1().Ingresses(f.Namespace).Get(f.name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			//Create the file
			d = &v1beta1.Ingress{}
			dec := k8yaml.NewYAMLOrJSONDecoder(bytes.NewReader(req.Data), 100)
			err = dec.Decode(d)
			if err != nil {
				panic(err)
			}

			d.Name = f.name
			d.Status = v1beta1.IngressStatus{}
			d.ObjectMeta.ResourceVersion = ""

			fmt.Println("D=", d)
			_, err = f.Client.NetworkingV1beta1().Ingresses(f.Namespace).Create(d)
			if err != nil {
				panic(err)
			}
			resp.Size = len(req.Data)

			return nil
		}
		panic(err)
	}

	data, err := k8yaml.ToJSON(req.Data)
	if err != nil {
		panic(err)
	}
	fmt.Println("data=", string(data))

	_, err = f.Client.NetworkingV1beta1().Ingresses(f.Namespace).Patch(f.name, types.MergePatchType, data)
	if err != nil {
		panic(err)
	}
	resp.Size = len(req.Data)
	return nil
}

func (f *File) Setattr(ctx context.Context, req *fuse.SetattrRequest, resp *fuse.SetattrResponse) error {
	fmt.Println("Ingress.File.Setattr,", f.name)
	return nil
}

//This is need for write to function
func (f *File) Fsync(ctx context.Context, req *fuse.FsyncRequest) error {
	fmt.Println("Ingress.File.Fsync,", f.name)
	req.Conn.InvalidateNode(req.Node, 0, 0)
	return nil
}
