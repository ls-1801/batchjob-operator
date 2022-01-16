package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/julienschmidt/httprouter"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"net/http"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
)

type WebServer struct {
	Client *SimpleReconciler
	Router *httprouter.Router
}

func NewWebServer(client *SimpleReconciler) *WebServer {
	ws := &WebServer{Client: client}
	ws.setupRouter()
	return ws
}

func (ws *WebServer) setupRouter() {
	ws.Router = httprouter.New()
	ws.Router.GET("/queue", ws.GetQueue)
	ws.Router.GET("/nodes", ws.GetNodes)
	ws.Router.POST("/schedule", ws.SubmitSchedule)

	ws.Router.POST("/extender/filter", ws.Filter)
	ws.Router.POST("/extender/prioritize", ws.Prioritize)
}

func (ws *WebServer) Start(context context.Context) error {
	var logger = ctrllog.FromContext(context)
	logger.Info("Listening on port 9090")
	return http.ListenAndServe(":9090", ws.Router)
}

func (ws *WebServer) GetQueue(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	var logger = ctrllog.FromContext(req.Context())
	defer logger.Info("Queue Request Done")
	logger.Info("Queue Request Started")

	var jobDescriptions = ws.Client.JobQueue.copyQueue()

	w.Header().Set("Content-Type", "application/json")
	logger.Info("Current Queue contains", "queue", jobDescriptions)

	HandleError(json.NewEncoder(w).Encode(jobDescriptions))
}

func (ws *WebServer) GetNodes(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	var logger = ctrllog.FromContext(req.Context())
	defer logger.Info("Node Request Done")
	logger.Info("Node Request Started")

	var nodeList = &corev1.NodeList{}
	if err := ws.Client.Client.List(req.Context(), nodeList); err != nil {
		logger.Error(err, "Error getting Nodes")
		HandleError1(fmt.Fprintln(w, "Error getting the Node list"))
	}

	var podList = &corev1.PodList{}
	if err := ws.Client.Client.List(req.Context(), podList); err != nil {
		logger.Error(err, "Error getting Pods")
		HandleError1(fmt.Fprintln(w, "Error getting the Pod list"))
	}

	var nodeMap = make(map[string][]string)

	for _, node := range nodeList.Items {
		if _, ok := nodeMap[node.Name]; !ok {
			nodeMap[node.Name] = []string{}
		}
	}

	for _, pod := range podList.Items {
		if val, ok := nodeMap[pod.Spec.NodeName]; ok {
			nodeMap[pod.Spec.NodeName] = append(val, pod.Name)
		} else {
			logger.Error(errors.New("Node not found: "+pod.Spec.NodeName), "Pod is Scheduled on an unknown Node")
		}
	}

	w.Header().Set("Content-Type", "application/json")
	HandleError(json.NewEncoder(w).Encode(nodeMap))
}

func (ws *WebServer) SubmitSchedule(writer http.ResponseWriter, request *http.Request, _ httprouter.Params) {
	logger := ctrllog.FromContext(request.Context())
	var schedulingDecision = map[string][]types.NamespacedName{}
	err := json.NewDecoder(request.Body).Decode(&schedulingDecision)
	if err != nil {
		logger.Error(err, "When decoding Payload")
		http.Error(writer, "Could not decode JSON", http.StatusBadRequest)
		return
	}

	err, executor := NewScheduleExecutor(request.Context(), ws.Client, schedulingDecision)
	if err != nil {
		logger.Error(err, "When creating the Scheduling Executor")
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}

	err, responseMap := executor.Execute(request.Context(), ws.Client)
	if err != nil {
		logger.Error(err, "During Scheduling Executor execution")
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}

	HandleError(json.NewEncoder(writer).Encode(responseMap))

}

func (ws *WebServer) Filter(writer http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
	writer.Write([]byte("OK"))
	writer.WriteHeader(http.StatusOK)
}

func (ws *WebServer) Prioritize(writer http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
	writer.Write([]byte("OK"))
	writer.WriteHeader(http.StatusOK)
}
