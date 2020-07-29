package configmap

import (
	"context"
	"time"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"github.com/sorenmat/kubefs/pkg/cluster"
	"github.com/sorenmat/kubefs/pkg/kubernetes"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	kube "k8s.io/client-go/kubernetes"
	"sigs.k8s.io/yaml"
)

// File implements both Node and Handle for the hello file.
type ConfigMapFile struct {
	content   string
	Cluster   cluster.Cluster
	Namespace string
	name      string
	Client    *kube.Clientset
	Modified  time.Time
}

func (f ConfigMapFile) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Mode = 0644
	a.Size = uint64(len(f.content))
	return nil
}

func (f ConfigMapFile) ReadAll(ctx context.Context) ([]byte, error) {
	return []byte(f.content), nil
}
func (f *ConfigMapFile) Open(ctx context.Context, req *fuse.OpenRequest, resp *fuse.OpenResponse) (fs.Handle, error) {
	f.reread()
	return f, nil
}

func (f *ConfigMapFile) reread() {
	cli := kubernetes.Client(f.Cluster.Context)

	deployment, err := cli.CoreV1().ConfigMaps(f.Namespace).Get(f.name, metav1.GetOptions{})
	if err != nil {
		panic(err)
	}
	deployment.Kind = "ConfigMap"
	deployment.APIVersion = "v1"

	data, err := yaml.Marshal(deployment)
	if err != nil {
		panic(err.Error())
	}
	f.content = string(data)
}

func (f *ConfigMapFile) Write(ctx context.Context, req *fuse.WriteRequest, resp *fuse.WriteResponse) error {
	d, err := f.Client.CoreV1().ConfigMaps(f.Namespace).Get(f.name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			//Create the file
			d = &corev1.ConfigMap{}
			err := yaml.Unmarshal(req.Data, d)
			if err != nil {
				panic(err)
			}
			d.Name = f.name
			d.ObjectMeta.ResourceVersion = ""
			_, err = f.Client.CoreV1().ConfigMaps(f.Namespace).Create(d)
			if err != nil {
				panic(err)
			}
			resp.Size = len(req.Data)

			return nil
		}
		panic(err)
	}
	data, err := yaml.YAMLToJSON(req.Data)
	if err != nil {
		panic(err)
	}

	_, err = f.Client.CoreV1().ConfigMaps(f.Namespace).Patch(f.name, types.MergePatchType, data)
	if err != nil {
		panic(err)
	}
	resp.Size = len(req.Data)
	return nil
}

func (f *ConfigMapFile) Setattr(ctx context.Context, req *fuse.SetattrRequest, resp *fuse.SetattrResponse) error {
	return nil
}

//This is need for write to function
func (f *ConfigMapFile) Fsync(ctx context.Context, req *fuse.FsyncRequest) error {
	req.Conn.InvalidateNode(req.Node, 0, 0)
	return nil
}
