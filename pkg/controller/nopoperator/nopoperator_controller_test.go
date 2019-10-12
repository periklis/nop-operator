package nopoperator

import (
	"fmt"
	"net/http"
	"testing"

	"net/http/httptest"

	"github.com/google/go-cmp/cmp"
	"github.com/periklis/nop-operator/pkg/apis/operators/v1alpha1"
	operatorsv1alpha1 "github.com/periklis/nop-operator/pkg/apis/operators/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var (
	mgr  manager.Manager
	logt = logf.Log.WithName("nopoperator-controller-test")
)

type TestReconciler struct {
	client     client.Client
	scheme     *runtime.Scheme
	httpClient *http.Client
}

func newTestReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &TestReconciler{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

func (_ TestReconciler) Reconcile(o reconcile.Request) (reconcile.Result, error) {
	return reconcile.Result{}, nil
}

func newTestManager(cfg *rest.Config) (manager.Manager, error) {
	opts := manager.Options{}
	mgr, err := manager.New(cfg, opts)
	if err != nil {
		return nil, fmt.Errorf("unable to create manager: %s", err)
	}

	if err := v1alpha1.SchemeBuilder.AddToScheme(mgr.GetScheme()); err != nil {
		return nil, fmt.Errorf("unable to add scheme: %s", err)
	}

	return mgr, nil
}

func newTestHttpServer(statusCode int, archive string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(statusCode)
		if archive != "" {
			http.ServeFile(w, r, archive)
		}
	}))
}

func TestAddToManager(t *testing.T) {
	testEnv := &envtest.Environment{
		CRDDirectoryPaths: []string{"../../../deploy/crds"},
	}
	cfg, err := testEnv.Start()
	if err != nil {
		t.Fatalf("unable to create test env: %s", err)
	}
	defer testEnv.Stop()

	mgr, err := newTestManager(cfg)
	if err != nil {
		t.Fatal(err)
	}

	err = add(mgr, newTestReconciler(mgr))
	if err != nil {
		t.Errorf("Error adding controller to manager: %s", err)
	}
}

func TestReconcile(t *testing.T) {
	scheme := scheme.Scheme
	v1alpha1.SchemeBuilder.AddToScheme(scheme)

	tests := []struct {
		desc       string
		operator   *operatorsv1alpha1.NopOperator
		statusCode int
		archive    string
		wantErr    bool
		want       reconcile.Result
	}{
		{
			desc: "reconcile success empty channels",
			operator: &operatorsv1alpha1.NopOperator{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "empty-nop-operator",
					Namespace: "test-namespace",
				},
				Spec: operatorsv1alpha1.NopOperatorSpec{
					Operators: []operatorsv1alpha1.OperatorChannel{},
				},
			},
			statusCode: 200,
			want:       reconcile.Result{},
		},
		{
			desc: "reconcile success with valid archive",
			operator: &operatorsv1alpha1.NopOperator{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "empty-nop-operator",
					Namespace: "test-namespace",
				},
				Spec: operatorsv1alpha1.NopOperatorSpec{
					Operators: []operatorsv1alpha1.OperatorChannel{
						{
							Name:    "a-operator",
							Version: "1.2.3",
						},
					},
				},
			},
			archive:    "./testdata/manifests.tar.gz",
			statusCode: 200,
			want:       reconcile.Result{},
		},
		{
			desc: "reconcile with requeue empty archive",
			operator: &operatorsv1alpha1.NopOperator{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "empty-nop-operator",
					Namespace: "test-namespace",
				},
				Spec: operatorsv1alpha1.NopOperatorSpec{
					Operators: []operatorsv1alpha1.OperatorChannel{
						{
							Name:    "a-operator",
							Version: "1.2.3",
						},
					},
				},
			},
			statusCode: 200,
			want:       reconcile.Result{Requeue: true},
		},
		{
			desc: "reconcile failure channel server error",
			operator: &operatorsv1alpha1.NopOperator{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "empty-nop-operator",
					Namespace: "test-namespace",
				},
				Spec: operatorsv1alpha1.NopOperatorSpec{
					Operators: []operatorsv1alpha1.OperatorChannel{
						{
							Name:    "a-operator",
							Version: "1.2.3",
						},
					},
				},
			},
			statusCode: 500,
			wantErr:    true,
			want:       reconcile.Result{},
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			ts := newTestHttpServer(test.statusCode, test.archive)
			defer ts.Close()

			if len(test.operator.Spec.Operators) > 0 {
				test.operator.Spec.Operators[0].URL = ts.URL
			}

			cs := fake.NewFakeClientWithScheme(scheme, test.operator)
			rc := &ReconcileNopOperator{
				client:     cs,
				scheme:     scheme,
				httpClient: ts.Client(),
			}

			key := types.NamespacedName{Name: test.operator.Name, Namespace: test.operator.Namespace}
			got, err := rc.Reconcile(reconcile.Request{NamespacedName: key})
			if test.wantErr && err == nil {
				t.Error("want err but go nothing")
			} else if diff := cmp.Diff(got, test.want); diff != "" {
				t.Errorf("got diff: %s", diff)
			}
		})
	}
}
