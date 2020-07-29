package pod

import (
	"context"
	"fmt"
	"time"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"github.com/sorenmat/kubefs/pkg/cluster"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	k8yaml "k8s.io/apimachinery/pkg/util/yaml"
	kube "k8s.io/client-go/kubernetes"
	"sigs.k8s.io/yaml"
)

type File struct {
	fs.Node
	fs.NodeSetattrer
	content   string
	cluster   cluster.Cluster
	Namespace string
	name      string
	Modified  time.Time
	Client    *kube.Clientset
}

func (f *File) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Mode = 0644
	a.Size = uint64(len(f.content))
	a.Valid = 0 * time.Millisecond
	a.Mtime = f.Modified
	return nil
}

func (f *File) reread() {

	deployment, err := f.Client.CoreV1().Pods(f.Namespace).Get(f.name, metav1.GetOptions{})
	if err != nil {
		panic(err)
	}
	deployment.Kind = "Pod"
	deployment.APIVersion = "v1"
	data, err := yaml.Marshal(deployment)
	if err != nil {
		panic(err.Error())
	}
	f.content = string(data)
}

func (f *File) Open(ctx context.Context, req *fuse.OpenRequest, resp *fuse.OpenResponse) (fs.Handle, error) {
	f.reread()
	return f, nil
}

func (f File) ReadAll(ctx context.Context) ([]byte, error) {
	return []byte(f.content), nil
}

func (f *File) Write(ctx context.Context, req *fuse.WriteRequest, resp *fuse.WriteResponse) error {
	d, err := f.Client.CoreV1().Pods(f.Namespace).Get(f.name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			//Create the file
			d = &corev1.Pod{}
			err := yaml.Unmarshal(req.Data, d)
			if err != nil {
				panic(err)
			}
			d.Name = f.name
			d.Status = corev1.PodStatus{}
			d.ObjectMeta.ResourceVersion = ""
			_, err = f.Client.CoreV1().Pods(f.Namespace).Create(d)
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
	fmt.Println("data=", data)
	_, err = f.Client.CoreV1().Pods(f.Namespace).Patch(f.name, types.MergePatchType, data)
	if err != nil {
		panic(err)
	}
	resp.Size = len(req.Data)
	return nil
}

func (f *File) Setattr(ctx context.Context, req *fuse.SetattrRequest, resp *fuse.SetattrResponse) error {
	return nil
}

//This is need for write to function
func (f *File) Fsync(ctx context.Context, req *fuse.FsyncRequest) error {
	req.Conn.InvalidateNode(req.Node, 0, 0)
	return nil
}
