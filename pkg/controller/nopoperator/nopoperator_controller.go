package nopoperator

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-logr/logr"
	"github.com/mholt/archiver"
	operatorsv1alpha1 "github.com/periklis/nop-operator/pkg/apis/operators/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/api/rbac/v1beta1"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_nopoperator")

// Add creates a new NopOperator Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	tlsConf := &tls.Config{InsecureSkipVerify: true}
	tr := &http.Transport{TLSClientConfig: tlsConf}
	client := &http.Client{Transport: tr}
	return &ReconcileNopOperator{client: mgr.GetClient(), scheme: mgr.GetScheme(), httpClient: client}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("nopoperator-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource NopOperator
	err = c.Watch(&source.Kind{Type: &operatorsv1alpha1.NopOperator{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}
	return nil
}

// blank assignment to verify that ReconcileNopOperator implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileNopOperator{}

// ReconcileNopOperator reconciles a NopOperator object
type ReconcileNopOperator struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client     client.Client
	scheme     *runtime.Scheme
	httpClient *http.Client
}

// Reconcile reads that state of the cluster for a NopOperator object and makes changes based on the state read
// and what is in the NopOperator.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileNopOperator) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling NopOperator")

	// Fetch the NopOperator instance
	instance := &operatorsv1alpha1.NopOperator{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	for _, op := range instance.Spec.Operators {
		log.Info("Processing operator from channel", "Operator.Name", op.Name, "Operator.Version", op.Version, "Operator.URL", op.URL)
		objs, err := r.readChannel(op)
		if err != nil {
			return reconcile.Result{}, err
		}

		for _, obj := range objs {
			err := r.newObjectForCR(instance, op.Name, obj, reqLogger)
			if err != nil {
				return reconcile.Result{}, err
			}
		}
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileNopOperator) readChannel(oc operatorsv1alpha1.OperatorChannel) ([]*runtime.Object, error) {
	log.Info("Fetch Manifests for operator", "Operator.Name", oc.Name)

	resp, err := r.httpClient.Get(oc.URL)
	if err != nil {
		return nil, fmt.Errorf("Error fetching manifests for %s/%s: %s", oc.Name, oc.Version, err)
	}
	defer resp.Body.Close()

	dir, err := ioutil.TempDir("", "example")
	if err != nil {
		return nil, fmt.Errorf("Error creating manifest tmp dir: %s", err)
	}
	defer os.RemoveAll(dir)

	baseName := fmt.Sprintf("%s-%s", oc.Name, oc.Version)
	source := filepath.Join(dir, fmt.Sprintf("%s.tar.gz", baseName))
	out, err := os.Create(source)
	if err != nil {
		return nil, fmt.Errorf("Error creating manifest tmp file: %s", err)
	}
	defer out.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Error copy manifest contents into tmp file: %s", err)
	}

	target := filepath.Join(dir, baseName)
	if err := archiver.Unarchive(source, target); err != nil {
		return nil, fmt.Errorf("Error unarchiving manifests: %s", err)
	}

	var objs []*runtime.Object

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
		objs = append(objs, &obj)

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("Error walking though manifests: %s", err)
	}

	return objs, nil
}

func (r *ReconcileNopOperator) newObjectForCR(instance *operatorsv1alpha1.NopOperator, name string, obj *runtime.Object, logger logr.Logger) error {

	appendLabels := func(labels map[string]string) map[string]string {
		extra := map[string]string{
			"app": name,
		}

		for k, v := range extra {
			labels[k] = v
		}

		return labels
	}

	mo := (*obj).(metav1.Object)
	if err := controllerutil.SetControllerReference(instance, mo, r.scheme); err != nil {
		return fmt.Errorf("Error setting controller reference: %s", err)
	}

	switch o := mo.(type) {
	case *v1beta1.Role:
		o.Labels = appendLabels(o.GetLabels())

		found := &v1beta1.Role{}
		err := r.client.Get(context.TODO(), types.NamespacedName{Name: o.Name, Namespace: o.Namespace}, found)
		if err != nil && errors.IsNotFound(err) {
			logger.Info("Creating a new Role", "Role.Namespace", o.Namespace, "Role.Name", o.Name)
			err = r.client.Create(context.TODO(), o)
			if err != nil {
				return fmt.Errorf("Error creating new Role: %s", err)
			}
		} else if err != nil {
			return err
		}

	case *v1beta1.RoleBinding:
		o.Labels = appendLabels(o.GetLabels())

		found := &v1beta1.RoleBinding{}
		err := r.client.Get(context.TODO(), types.NamespacedName{Name: o.Name, Namespace: o.Namespace}, found)
		if err != nil && errors.IsNotFound(err) {
			logger.Info("Creating a new RoleBinding", "RoleBinding.Namespace", o.Namespace, "RoleBinding.Name", o.Name)
			err = r.client.Create(context.TODO(), o)
			if err != nil {
				return fmt.Errorf("Error creating new RoleBinding: %s", err)
			}
		} else if err != nil {
			return err
		}

	case *v1beta1.ClusterRole:
		o.Labels = appendLabels(o.GetLabels())
		found := &v1beta1.ClusterRole{}
		err := r.client.Get(context.TODO(), types.NamespacedName{Name: o.Name, Namespace: o.Namespace}, found)
		if err != nil && errors.IsNotFound(err) {
			logger.Info("Creating a new ClusterRole", "ClusterRole.Namespace", o.Namespace, "ClusterRole.Name", o.Name)
			err = r.client.Create(context.TODO(), o)
			if err != nil {
				return fmt.Errorf("Error creating new ClusterRole: %s", err)
			}
		} else if err != nil {
			return err
		}

	case *v1beta1.ClusterRoleBinding:
		o.Labels = appendLabels(o.GetLabels())
		found := &v1beta1.ClusterRoleBinding{}
		err := r.client.Get(context.TODO(), types.NamespacedName{Name: o.Name, Namespace: o.Namespace}, found)
		if err != nil && errors.IsNotFound(err) {
			logger.Info("Creating a new ClusterRoleBinding", "ClusterRoleBinding.Namespace", o.Namespace, "ClusterRoleBinding.Name", o.Name)
			err = r.client.Create(context.TODO(), o)
			if err != nil {
				return fmt.Errorf("Error creating new ClusterRoleBinding: %s", err)
			}
		} else if err != nil {
			return err
		}

	case *v1.ServiceAccount:
		o.Labels = appendLabels(o.GetLabels())
		found := &v1.ServiceAccount{}
		err := r.client.Get(context.TODO(), types.NamespacedName{Name: o.Name, Namespace: o.Namespace}, found)
		if err != nil && errors.IsNotFound(err) {
			logger.Info("Creating a new ServiceAccount", "ServiceAccount.Namespace", o.Namespace, "ServiceAccount.Name", o.Name)
			err = r.client.Create(context.TODO(), o)
			if err != nil {
				return fmt.Errorf("Error creating new ServiceAccount: %s", err)
			}
		} else if err != nil {
			return err
		}
	case *apiextensionsv1beta1.CustomResourceDefinition:
		o.Labels = appendLabels(o.GetLabels())

		found := &apiextensionsv1beta1.CustomResourceDefinition{}
		err := r.client.Get(context.TODO(), types.NamespacedName{Name: o.Name, Namespace: o.Namespace}, found)
		if err != nil && errors.IsNotFound(err) {
			logger.Info("Creating a new CRD", "CRD.Namespace", o.Namespace, "CRD.Name", o.Name)
			err = r.client.Create(context.TODO(), o)
			if err != nil {
				return fmt.Errorf("Error creating new CRD: %s", err)
			}
		} else if err != nil {
			return err
		}

	case *appsv1.Deployment:
		o.Labels = appendLabels(o.GetLabels())

		found := &appsv1.Deployment{}
		err := r.client.Get(context.TODO(), types.NamespacedName{Name: o.Name, Namespace: o.Namespace}, found)
		if err != nil && errors.IsNotFound(err) {
			logger.Info("Creating a new Deployment", "Deployment.Namespace", o.Namespace, "Deployment.Name", o.Name)
			err = r.client.Create(context.TODO(), o)
			if err != nil {
				return fmt.Errorf("Error creating new Deployment: %s", err)
			}
		} else if err != nil {
			return err
		}
	}

	return nil
}
