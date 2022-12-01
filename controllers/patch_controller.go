/*
Copyright 2022.

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

package controllers

import (
	"context"
	"time"

	reformav1beta1 "prosimcorp.com/reforma/api/v1beta1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	defaultSyncTimeForExitWithError = 10 * time.Second

	scheduleSynchronization     = "Schedule synchronization in: %s"
	patchNotFoundError          = "Patch resource not found. Ignoring since object must be deleted."
	patchRetrievalError         = "Error getting the Patch from the cluster"
	patchFinalizersUpdateError  = "Failed to update finalizer of Patch: %s"
	patchConditionUpdateError   = "Failed to update the condition on Patch: %s"
	patchSyncTimeRetrievalError = "Can not get synchronization time from the Patch: %s"
	patchTargetError            = "Can not patch the target for the Patch: %s"

	patchFinalizer = "reforma.prosimcorp.com/finalizer"
)

// PatchReconciler reconciles a Patch object
type PatchReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=reforma.prosimcorp.com,resources=patches,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=reforma.prosimcorp.com,resources=patches/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=reforma.prosimcorp.com,resources=patches/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=secrets;configmaps,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *PatchReconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, err error) {
	//1. Get the content of the Patch
	patchManifest := &reformav1beta1.Patch{}
	err = r.Get(ctx, req.NamespacedName, patchManifest)

	// 2. Check existence on the cluster
	if err != nil {

		// 2.1 It does NOT exist: manage removal
		if err = client.IgnoreNotFound(err); err == nil {
			LogInfof(ctx, patchNotFoundError)
			return result, err
		}

		// 2.2 Failed to get the resource, requeue the request
		LogInfof(ctx, patchRetrievalError)
		return result, err
	}

	// 3. Check if the Patch instance is marked to be deleted: indicated by the deletion timestamp being set
	if !patchManifest.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(patchManifest, patchFinalizer) {
			// Remove the finalizers on Patch CR
			controllerutil.RemoveFinalizer(patchManifest, patchFinalizer)
			err = r.Update(ctx, patchManifest)
			if err != nil {
				LogInfof(ctx, patchFinalizersUpdateError, req.Name)
			}
		}
		result = ctrl.Result{}
		err = nil
		return result, err
	}

	// 4. Add finalizer to the Patch CR
	if !controllerutil.ContainsFinalizer(patchManifest, patchFinalizer) {
		controllerutil.AddFinalizer(patchManifest, patchFinalizer)
		err = r.Update(ctx, patchManifest)
		if err != nil {
			return result, err
		}
	}

	// 5. Update the status before the requeue
	defer func() {
		err = r.Status().Update(ctx, patchManifest)
		if err != nil {
			LogInfof(ctx, patchConditionUpdateError, req.Name)
		}
	}()

	// 6. Schedule periodical request
	RequeueTime, err := r.GetSynchronizationTime(patchManifest)
	if err != nil {
		LogInfof(ctx, patchSyncTimeRetrievalError, patchManifest.Name)
		return result, err
	}
	result = ctrl.Result{
		RequeueAfter: RequeueTime,
	}

	// 7. The Patch CR already exist: manage the update
	err = r.PatchTarget(ctx, patchManifest)
	if err != nil {
		LogInfof(ctx, patchTargetError, patchManifest.Name)
		return result, err
	}

	// 8. Success, update the status
	r.UpdatePatchCondition(patchManifest, r.NewPatchCondition(ConditionTypeResourcePatched,
		metav1.ConditionTrue,
		ConditionReasonTargetPatched,
		ConditionReasonTargetPatchedMessage,
	))

	LogInfof(ctx, scheduleSynchronization, result.RequeueAfter.String())
	return result, err
}

// SetupWithManager sets up the controller with the Manager.
func (r *PatchReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&reformav1beta1.Patch{}).
		Complete(r)
}
