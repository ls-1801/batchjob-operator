package controllers

import (
	"context"
	"log"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
	"strconv"
)

func CompareResourceVersion(ctx context.Context, rv1 string, rv2 string) bool {
	oldRV, err := strconv.Atoi(rv1)
	if err != nil {
		ctrllog.FromContext(ctx).Error(err, "Could not extract old Resource Version")
	}
	newRV, err := strconv.Atoi(rv2)
	if err != nil {
		ctrllog.FromContext(ctx).Error(err, "Could not extract new Resource Version")
	}

	return oldRV < newRV
}

func HandleError(err error) {
	if err == nil {
		return
	}
	log.Default().Print("HandleError:", err)
}

func HandleError1(i int, err error) {
	if err == nil {
		return
	}
	log.Default().Print("HandleError: ", "int", i, "error", err)
}
