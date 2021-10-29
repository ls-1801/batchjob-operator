package controllers

import (
	"k8s.io/apimachinery/pkg/types"
	"sync"
)

type JobQueue struct {
	QueueMutex sync.Mutex
	Queue      []types.NamespacedName
}

func NewJobQueue() *JobQueue {
	return &JobQueue{
		QueueMutex: sync.Mutex{},
		Queue:      []types.NamespacedName{},
	}
}

func (jq *JobQueue) removeFromQueue(jobName types.NamespacedName) *types.NamespacedName {
	jq.QueueMutex.Lock()
	defer func() { jq.QueueMutex.Unlock() }()

	for i, v := range jq.Queue {
		if v.Name == jobName.Name && v.Namespace == jobName.Namespace {
			jq.Queue[i] = jq.Queue[len(jq.Queue)-1]
			jq.Queue = jq.Queue[:len(jq.Queue)-1]
			return &v
		}
	}
	return nil
}

func (jq *JobQueue) copyQueue() []JobDescription {
	jq.QueueMutex.Lock()
	defer func() { jq.QueueMutex.Unlock() }()

	var jobDescriptions = make([]JobDescription, len(jq.Queue))

	for idx, name := range jq.Queue {
		jobDescriptions[idx] = JobDescription{
			JobName: name,
		}
	}
	return jobDescriptions
}

func (jq *JobQueue) addJobToQueue(jobName types.NamespacedName) {
	jq.QueueMutex.Lock()
	defer func() { jq.QueueMutex.Unlock() }()

	jq.Queue = append(jq.Queue, jobName)
}
