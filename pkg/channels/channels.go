package channels

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-logr/logr"
	"github.com/mholt/archiver"
	"github.com/periklis/nop-operator/pkg/apis/operators/v1alpha1"
	"github.com/prometheus/common/log"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
)

type ChannelReader interface {
	Read() ([]runtime.Object, bool, error)
}

type simpleReader struct {
	client  *http.Client
	log     logr.Logger
	channel v1alpha1.OperatorChannel
}

func NewChannelReader(client *http.Client, log logr.Logger, channel v1alpha1.OperatorChannel) ChannelReader {
	return &simpleReader{client: client, log: log, channel: channel}
}

func (sr *simpleReader) Read() ([]runtime.Object, bool, error) {
	oc := sr.channel

	log.Info("Fetch Manifests for operator: ", "Name: ", oc.Name)

	resp, err := sr.client.Get(oc.URL)
	if err != nil {
		return nil, false, fmt.Errorf("Error fetching manifests for %s/%s: %s", oc.Name, oc.Version, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode > 299 {
		return nil, false, fmt.Errorf("Error response status code %d", resp.StatusCode)
	}

	dir, err := ioutil.TempDir("", "example")
	if err != nil {
		return nil, true, fmt.Errorf("Error creating manifest tmp dir: %s", err)
	}
	defer os.RemoveAll(dir)

	baseName := fmt.Sprintf("%s-%s", oc.Name, oc.Version)
	source := filepath.Join(dir, fmt.Sprintf("%s.tar.gz", baseName))
	out, err := os.Create(source)
	if err != nil {
		return nil, true, fmt.Errorf("Error creating manifest tmp file: %s", err)
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return nil, true, fmt.Errorf("Error copy manifest contents into tmp file: %s", err)
	}

	target := filepath.Join(dir, baseName)
	if err := archiver.Unarchive(source, target); err != nil {
		return nil, true, fmt.Errorf("Error unarchiving manifests: %s", err)
	}

	var objs []runtime.Object
	err = filepath.Walk(target, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}

		contents, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}

		obj, err := runtime.Decode(scheme.Codecs.UniversalDeserializer(), contents)
		if err != nil {
			return err
		}
		objs = append(objs, obj)

		return nil
	})

	if err != nil {
		return nil, false, fmt.Errorf("Error walking though manifests: %s", err)
	}

	return objs, false, nil
}
