package nopoperator

import (
	"context"
	"fmt"
	"net/http"

	operatorsv1alpha1 "github.com/periklis/nop-operator/pkg/apis/operators/v1alpha1"
	"github.com/periklis/nop-operator/pkg/channels"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
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
func Add(mgr manager.Manager, client *http.Client) error {
	return add(mgr, newReconciler(mgr, client))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager, client *http.Client) reconcile.Reconciler {
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

	ctx := context.TODO()

	// Fetch the NopOperator instance
	instance := &operatorsv1alpha1.NopOperator{}
	err := r.client.Get(ctx, request.NamespacedName, instance)
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

		reader := channels.NewChannelReader(r.httpClient, log, op)

		var objs []runtime.Object
		_, shouldRequeue, err := reader.Read(objs)
		if err != nil {
			return reconcile.Result{Requeue: shouldRequeue}, err
		}

		for _, obj := range objs {
			err := r.newObjectForCR(ctx, instance, op.Name, obj)
			if err != nil {
				return reconcile.Result{}, err
			}
		}
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileNopOperator) newObjectForCR(ctx context.Context, instance *operatorsv1alpha1.NopOperator, name string, obj runtime.Object) error {
	mo := obj.(metav1.Object)
	if err := controllerutil.SetControllerReference(instance, mo, r.scheme); err != nil {
		return fmt.Errorf("Error setting controller reference: %s", err)
	}

	var found runtime.Object
	key := types.NamespacedName{Name: mo.GetName(), Namespace: mo.GetNamespace()}

	err := r.client.Get(ctx, key, found)
	if errors.IsNotFound(err) {
		log.Info("Creating a new Object", "Namespace", mo.GetNamespace(), "Name", mo.GetName())
		err = r.client.Create(ctx, obj)
		if err != nil {
			return fmt.Errorf("Error creating new Role: %s", err)
		}
	} else if err != nil {
		return err
	}

	return nil
}
