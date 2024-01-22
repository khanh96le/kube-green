package daemonsets

import (
	"context"
	"testing"

	"github.com/kube-green/kube-green/api/v1alpha1"
	"github.com/kube-green/kube-green/controllers/sleepinfo/resource"
	"github.com/kube-green/kube-green/internal/testutil"

	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func TestSleep(t *testing.T) {
	testLogger := zap.New(zap.UseDevMode(true))

	namespace := "my-namespace"

	d1 := GetMock(MockSpec{
		Namespace:       namespace,
		Name:            "d1",
		ResourceVersion: "2",
		PodNodeSelector: map[string]string{},
	})
	d2 := GetMock(MockSpec{
		Namespace:       namespace,
		Name:            "d2",
		ResourceVersion: "1",
		PodNodeSelector: map[string]string{
			"valid-node-selector": "true",
		},
	})
	d3 := GetMock(MockSpec{
		Namespace:       namespace,
		Name:            "d3",
		ResourceVersion: "1",
		PodNodeSelector: map[string]string{
			"non-existing-node-selector": "true",
		},
	})

	ctx := context.Background()
	emptySleepInfo := &v1alpha1.SleepInfo{}
	listOptions := &client.ListOptions{
		Namespace: namespace,
		Limit:     500,
	}

	t.Run("update daemonset to have zero replicas", func(t *testing.T) {
		c := fake.NewClientBuilder().WithRuntimeObjects(&d1, &d2, &d3).Build()
		fakeClient := &testutil.PossiblyErroringFakeCtrlRuntimeClient{
			Client: c,
		}

		resource, err := NewResource(ctx, resource.ResourceClient{
			Client:    fakeClient,
			Log:       testLogger,
			SleepInfo: emptySleepInfo,
		}, namespace, map[string]int32{})
		require.NoError(t, err)

		require.NoError(t, resource.Sleep(ctx))

		list := appsv1.DaemonSetList{}
		err = c.List(ctx, &list, listOptions)
		require.NoError(t, err)
		require.Equal(t, appsv1.DaemonSetList{
			TypeMeta: metav1.TypeMeta{
				Kind:       "DaemonSetList",
				APIVersion: "apps/v1",
			},
			Items: []appsv1.DaemonSet{
				GetMock(MockSpec{
					Namespace:       namespace,
					Name:            "d1",
					ResourceVersion: "3",
					PodNodeSelector: map[string]string{
						"non-existing-node-selector": "true",
					},
				}),
				GetMock(MockSpec{
					Namespace:       namespace,
					Name:            "d2",
					ResourceVersion: "2",
					PodNodeSelector: map[string]string{
						"valid-node-selector":        "true",
						"non-existing-node-selector": "true",
					},
				}),
				GetMock(MockSpec{
					Namespace:       namespace,
					Name:            "d3",
					ResourceVersion: "2",
					PodNodeSelector: map[string]string{
						"non-existing-node-selector": "true",
					},
				}),
			},
		}, list)
	})
}
