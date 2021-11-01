package controllers

import (
	"context"
	"github.com/GoogleCloudPlatform/spark-on-k8s-operator/pkg/apis/sparkoperator.k8s.io/v1beta2"
	"github.com/google/go-cmp/cmp"
	batchjobv1alpha1 "github.com/ls-1801/batchjob-operator/api/v1alpha1"
	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
)

type SparkController struct {
	Applications map[types.NamespacedName]*v1beta2.SparkApplication
	Client       *SimpleReconciler
}

func NewSparkController(client *SimpleReconciler) *SparkController {
	return &SparkController{Client: client, Applications: map[types.NamespacedName]*v1beta2.SparkApplication{}}
}

const (
	SparkNoChange = iota
	SparkCreated
	SparkSubmitted
	SparkRunning
	SparkRemoved
	SparkProblem
)

func (sc *SparkController) hasSparkChanged(ctx context.Context, job *batchjobv1alpha1.Simple) int {
	var logger = ctrllog.FromContext(ctx)
	logger.Info("Check if SparkApplication has Changed")

	old := sc.Applications[types.NamespacedName{
		Name:      job.Name,
		Namespace: job.Namespace,
	}]

	err, found := sc.getSpark(ctx, types.NamespacedName{Name: job.Name, Namespace: job.Namespace})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			ctrllog.FromContext(ctx).V(TRACE).Info("SparkApplication was not found in Cluster",
				"name", types.NamespacedName{Name: job.Name, Namespace: job.Namespace})
			if old != nil {
				logger.Info("SparkApplication was removed",
					"name", types.NamespacedName{Name: job.Name, Namespace: job.Namespace})
				delete(sc.Applications, types.NamespacedName{Name: job.Name, Namespace: job.Namespace})
				return SparkRemoved
			} else {
				logger.Info(
					"SparkApplication never existed",
					"name", types.NamespacedName{Name: job.Name, Namespace: job.Namespace},
				)
				return SparkProblem
			}
		}
	}

	if old == nil {
		logger.Info("SparkApplication was created")
		sc.manageSpark(ctx, found)
		return SparkCreated
	}

	ctrllog.FromContext(ctx).Info("Check for differences", "old", *old, "new", *found)
	if sc.Client.compareResourceVersion(ctx, old.ResourceVersion, found.ResourceVersion) {
		if cmp.Equal(old.Status, found.Status) {
			ctrllog.FromContext(ctx).Error(errors.New("no status change"), "SparkApplication has no status change")
		}
		sc.manageSpark(ctx, found)
		return toTransitionEnumSpark(
			context.WithValue(context.WithValue(ctx, "old", old.Status.AppState.State), "new", found.Status.AppState.State),
			old.Status.AppState.State,
			found.Status.AppState.State,
		)
	}

	return SparkNoChange
}

func toTransitionEnumSpark(ctx context.Context, before v1beta2.ApplicationStateType, after v1beta2.ApplicationStateType) int {

	if before == v1beta2.NewState && after == v1beta2.SubmittedState {
		ctrllog.FromContext(ctx).Info("Spark went to Submitted State")
		return SparkSubmitted
	}
	if before == v1beta2.SubmittedState && after == v1beta2.RunningState {
		ctrllog.FromContext(ctx).Info("Spark went to Running State")
		return SparkRunning
	}

	ctrllog.FromContext(ctx).Info("Spark is in an unknown/unhandled state")

	return SparkProblem
}

func (sc SparkController) getSpark(context context.Context, name types.NamespacedName) (error, *v1beta2.SparkApplication) {
	var spark = &v1beta2.SparkApplication{}
	ctrllog.FromContext(context).V(TRACE).Info("Get SparkApplication", "Namespaced-Name", name)
	var err = sc.Client.Get(context, name, spark)
	return err, spark
}

func (sc *SparkController) manageSpark(ctx context.Context, spark *v1beta2.SparkApplication) {

	var nn = types.NamespacedName{Name: spark.Name, Namespace: spark.Namespace}
	if old, ok := sc.Applications[nn]; ok {
		if sc.Client.compareResourceVersion(ctx, old.ResourceVersion, spark.ResourceVersion) {
			ctrllog.FromContext(ctx).Info("Managed SparkApplication changes from Version "+old.ResourceVersion+" to "+spark.ResourceVersion,
				"name", nn)
		} else {
			ctrllog.FromContext(ctx).Info("Managed SparkApplication versions did not change from: "+old.ResourceVersion,
				"name", nn)
		}
	} else {
		ctrllog.FromContext(ctx).Info("Manage new SparkApplication", "name", nn)
	}

	sc.Applications[nn] = spark
}
