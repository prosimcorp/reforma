package controller

import (
	"context"
	"errors"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/log"
)

// NewErrorf return an error with the message already formatted from parameters
func NewErrorf(msg string, params ...interface{}) error {
	msg = fmt.Sprintf(msg, params...)
	return errors.New(msg)
}

func LogInfof(ctx context.Context, message string, params ...interface{}) {
	log.FromContext(ctx).Info(fmt.Sprintf(message, params...))
}

func LogErrorf(ctx context.Context, err error, message string, params ...interface{}) {
	message = fmt.Sprintf(message, params...)
	log.FromContext(ctx).Error(err, message)
}
