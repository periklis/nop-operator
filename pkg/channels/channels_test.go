package channels

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/periklis/nop-operator/pkg/apis/operators/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	v1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

func newTestHttpServer(statusCode int, archive string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(statusCode)
		if archive != "" {
			http.ServeFile(w, r, archive)
		}
	}))
}

func TestRead(t *testing.T) {
	tests := []struct {
		desc        string
		channel     *v1alpha1.OperatorChannel
		statusCode  int
		archivePath string
		want        []runtime.Object
		wantErr     bool
		wantRequeue bool
	}{
		{
			desc: "non 2xx status code",
			channel: &v1alpha1.OperatorChannel{
				Name:    "a-operator",
				Version: "1.2.3",
			},
			statusCode:  http.StatusBadRequest,
			wantErr:     true,
			wantRequeue: false,
		},
		{
			desc: "empty response body",
			channel: &v1alpha1.OperatorChannel{
				Name:    "a-operator",
				Version: "1.2.3",
			},
			statusCode:  http.StatusOK,
			wantErr:     true,
			wantRequeue: true,
		},
		{
			desc: "broken archive",
			channel: &v1alpha1.OperatorChannel{
				Name:    "a-operator",
				Version: "1.2.3",
			},
			statusCode:  http.StatusOK,
			archivePath: "./testdata/broken.tar.gz",
			wantErr:     true,
			wantRequeue: true,
		},
		{
			desc: "empty archive",
			channel: &v1alpha1.OperatorChannel{
				Name:    "a-operator",
				Version: "1.2.3",
			},
			statusCode:  http.StatusOK,
			archivePath: "./testdata/empty.tar.gz",
			wantErr:     false,
			wantRequeue: false,
		},
		{
			desc: "valid manifests",
			channel: &v1alpha1.OperatorChannel{
				Name:    "a-operator",
				Version: "1.2.3",
			},
			statusCode:  http.StatusOK,
			archivePath: "./testdata/valid.tar.gz",
			want: []runtime.Object{
				&rbacv1.RoleBinding{
					TypeMeta: metav1.TypeMeta{
						Kind:       "RoleBinding",
						APIVersion: "rbac.authorization.k8s.io/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "a-operator",
						Namespace: "default",
					},
					Subjects: []v1.Subject{
						{
							Kind: "ServiceAccount",
							Name: "a-operator",
						},
					},
					RoleRef: v1.RoleRef{
						APIGroup: "rbac.authorization.k8s.io",
						Kind:     "Role",
						Name:     "a-operator",
					},
				},
				&corev1.ServiceAccount{
					TypeMeta: metav1.TypeMeta{
						Kind:       "ServiceAccount",
						APIVersion: "v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "a-operator",
						Namespace: "default",
					},
				},
			},
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()
			ts := newTestHttpServer(test.statusCode, test.archivePath)

			if test.channel != nil {
				test.channel.URL = ts.URL
			}

			c := ts.Client()
			r := NewChannelReader(c, logf.Log, *test.channel)

			got, gotR, err := r.Read()
			if test.wantErr && err == nil {
				t.Error("Want error but got nothing")
			}
			if gotR != test.wantRequeue {
				t.Errorf("got requeue request: %t, want requeue request: %t", gotR, test.wantRequeue)
			}
			if diff := cmp.Diff(got, test.want); diff != "" {
				t.Errorf("got diff: %s", diff)
			}
		})
	}
}
