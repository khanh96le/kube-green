package sleepinfo

import (
	"context"
	"os"
	"testing"

	"github.com/kube-green/kube-green/controllers/internal/testutil"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/envfuncs"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

var (
	testenv env.Environment
)

const (
	kindClusterName = "kube-green-e2e"
)

func TestMain(m *testing.M) {
	testenv = env.New()
	runID := envconf.RandomName("kube-green-test", 24)

	testenv.BeforeEachFeature(func(ctx context.Context, c *envconf.Config, t *testing.T, f features.Feature) (context.Context, error) {
		return testutil.CreateNSForTest(ctx, c, t, runID)
	})

	testenv.AfterEachFeature(func(ctx context.Context, c *envconf.Config, t *testing.T, f features.Feature) (context.Context, error) {
		return testutil.DeleteNamespace(ctx, c, t, runID)
	})

	testenv.Setup(
		testutil.CreateKindClusterWithVersion(kindClusterName),
		testutil.GetClusterVersion(),
		testutil.SetupCRDs("../../config/crd/bases", "*"),
	)

	testenv.Finish(
		envfuncs.TeardownCRDs("../../config/crd/bases", "*"),
		testutil.DestroyKindCluster(kindClusterName),
	)

	// launch package tests
	os.Exit(testenv.Run(m))
}
