/*
Copyright 2023.

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

package controller

import (
	"context"
	"log"
	"time"

	v1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	batchv1 "rakshitgondwal/project/api/v1"
)

// ScalerReconciler reconciles a Scaler object
type ScalerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=batch.rakshitgondwal.online,resources=scalers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=batch.rakshitgondwal.online,resources=scalers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=batch.rakshitgondwal.online,resources=scalers/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Scaler object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.4/pkg/reconcile
func (r *ScalerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log.Printf("reconcile called")
	scaler := &batchv1.Scaler{}
	err := r.Get(ctx, req.NamespacedName, scaler)
	if err != nil {
		return ctrl.Result{}, err
	}

	startTime := scaler.Spec.Start
	endTime := scaler.Spec.End
	replicas := scaler.Spec.Replicas

	currentHour := time.Now().UTC().Hour()

	if currentHour >= startTime && currentHour <= endTime {
		if err := scaleDeployment(scaler, r, ctx, replicas); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{RequeueAfter: time.Duration(30 * time.Second)}, nil
}

func scaleDeployment(scaler *batchv1.Scaler, r *ScalerReconciler, ctx context.Context, replicas int32) error {
	for _, deploy := range scaler.Spec.Deployments {
		deployment := &v1.Deployment{}
		err := r.Get(ctx, types.NamespacedName{
			Namespace: deploy.Namespace,
			Name:      deploy.Name,
		},
			deployment,
		)
		if err != nil {
			return err
		}

		if deployment.Spec.Replicas != &replicas {
			deployment.Spec.Replicas = &replicas
			err := r.Update(ctx, deployment)
			if err != nil {
				scaler.Status.Status = batchv1.FAILED
				return err
			}
			scaler.Status.Status = batchv1.SUCCESS

			err = r.Status().Update(ctx, scaler)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ScalerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&batchv1.Scaler{}).
		Complete(r)
}
