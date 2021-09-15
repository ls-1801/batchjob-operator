/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"container/list"
	"context"
	"flag"
	"fmt"
	"os"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"net/http"

	batchjobv1alpha1 "github.com/ls-1801/batchjob-operator/api/v1alpha1"
	"github.com/ls-1801/batchjob-operator/controllers"
	//+kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(batchjobv1alpha1.AddToScheme(scheme))
	utilruntime.Must(batchjobv1alpha1.AddToSchemeSpark(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "4430f8a5.gcr.io",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	var waiting = make(chan batchjobv1alpha1.Simple)
	var scheduled = make(chan batchjobv1alpha1.Simple)
	if err = (&controllers.SimpleReconciler{
		Client:    mgr.GetClient(),
		Scheme:    mgr.GetScheme(),
		Waiting:   waiting,
		Scheduled: scheduled,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Simple")
		os.Exit(1)
	}
	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	mgr.Add(WebServer{IncomingJobs: waiting, ScheduledJobs: scheduled, Queue: list.New()})

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}

}

type WebServer struct {
	Queue         *list.List
	IncomingJobs  chan batchjobv1alpha1.Simple
	ScheduledJobs chan batchjobv1alpha1.Simple
}

func (w WebServer) Start(context context.Context) error {
	var log = ctrllog.FromContext(context)
	log.Info("Init WebServer on port 9090")
	http.HandleFunc("/queue", w.GetQueue)
	go w.ListenForNewJobs(context)
	log.Info("Listening on port 9090")
	return http.ListenAndServe(":9090", nil)
}

func (w *WebServer) ListenForNewJobs(context context.Context) {
	var log = ctrllog.FromContext(context)
	log.Info("Waiting for New Jobs to be added to the Queue")
	for {
		select {
		case <-context.Done():
			return
		case newJob := <-w.IncomingJobs:
			log.Info("Adding new Job to the Queue")
			w.Queue.PushBack(newJob)
		}
	}
}

func (ws *WebServer) GetQueue(w http.ResponseWriter, req *http.Request) {
	var log = ctrllog.FromContext(req.Context())
	defer log.Info("Queue Request Done")
	log.Info("Queue Request Started")
	log.Info("Current Queue contains", "queue", ws.Queue)
	var string = "["
	for e := ws.Queue.Front(); e != nil; e = e.Next() {
		v := e.Value.(batchjobv1alpha1.Simple)
		string += (fmt.Sprintf("%s,", v.Name))
	}
	string += "]"
	fmt.Fprintf(w, "%s", string)
}
