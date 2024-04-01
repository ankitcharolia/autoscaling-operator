/*
Copyright 2024.

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
	"strconv"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	k8sv1beta1 "github.com/ankitcharolia/autoscaling-operator/api/v1beta1"
)

// AutoScalerReconciler reconciles a AutoScaler object
type AutoScalerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=k8s.charolia.io,resources=autoscalers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=k8s.charolia.io,resources=autoscalers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=k8s.charolia.io,resources=autoscalers/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the AutoScaler object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.16.3/pkg/reconcile
func (r *AutoScalerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	// fetch the AutoScaler object
	as := &k8sv1beta1.AutoScaler{}
	err := r.Get(ctx, req.NamespacedName, as)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}

	// get the attributes from Autoscaler object
	name := as.Spec.ScaleTargetRef.Name
	resourceType := as.Spec.ScaleTargetRef.Type
	minReplicaCount := as.Spec.MinReplicaCount

	// Check if the service account exists
	sa := &corev1.ServiceAccount{}
	err = r.Get(ctx, types.NamespacedName{Name: "autoscaler-" + name, Namespace: req.Namespace}, sa)
	if err != nil && errors.IsNotFound(err) {
		// Service account does not exist, create it
		sa := &corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "autoscaler-" + name,
				Namespace: req.Namespace,
			},
		}
		if err := r.Create(ctx, sa); err != nil {
			// Failed to create the service account
			return ctrl.Result{}, err
		}
	} else if err != nil {
		// Error occurred while checking the service account
		return ctrl.Result{}, err
	}

	// Check if the role exists
	cr := &rbacv1.Role{}
	err = r.Get(ctx, types.NamespacedName{Name: "autoscaler-" + name, Namespace: req.Namespace}, cr)
	if err != nil && errors.IsNotFound(err) {
		// role does not exist, create it
		cr := &rbacv1.Role{
			ObjectMeta: metav1.ObjectMeta{
				Name: "autoscaler-" + name,
			},
			Rules: []rbacv1.PolicyRule{
				{
					APIGroups:     []string{"apps"},
					Resources:     []string{resourceType, resourceType + "/scale"},
					ResourceNames: []string{name},
					Verbs:         []string{"get", "list", "watch", "update", "patch"},
				},
			},
		}
		if err := r.Create(ctx, cr); err != nil {
			// Failed to create the role
			return ctrl.Result{}, err
		}
	} else if err != nil {
		// Error occurred while checking the role
		return ctrl.Result{}, err
	}

	// Check if the role binding exists
	crb := &rbacv1.RoleBinding{}
	err = r.Get(ctx, types.NamespacedName{Name: "autoscaler-" + name, Namespace: req.Namespace}, crb)
	if err != nil && errors.IsNotFound(err) {
		// role binding does not exist, create it
		crb := &rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name: "autoscaler-" + name,
			},
			Subjects: []rbacv1.Subject{
				{
					Kind:      "ServiceAccount",
					Name:      "autoscaler-" + name,
					Namespace: req.Namespace,
				},
			},
			RoleRef: rbacv1.RoleRef{
				Kind:     "Role",
				Name:     "autoscaler-" + name,
				APIGroup: "rbac.authorization.k8s.io",
			},
		}
		if err := r.Create(ctx, crb); err != nil {
			// Failed to create the role binding
			return ctrl.Result{}, err
		}
	} else if err != nil {
		// Error occurred while checking the role binding
		return ctrl.Result{}, err
	}

	kubectlImage := "bitnami/kubectl:1.29.3"

	for _, trigger := range as.Spec.Triggers {
		DesiredReplicas := trigger.Metadata.DesiredReplicas
		timezone := trigger.Metadata.Timezone
		start := trigger.Metadata.Start
		end := trigger.Metadata.End

		// scale up the kubernetes resources to desired replicas
		// check if cronjob is already created for the autoscaler
		cronjob := &batchv1.CronJob{}
		err = r.Get(ctx, types.NamespacedName{Name: "autoscaler-" + name + "-scaleup", Namespace: req.Namespace}, cronjob)
		if err != nil && errors.IsNotFound(err) {
			// cronjob does not exist, create it
			cronjob = &batchv1.CronJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "autoscaler-" + name + "-scaleup",
					Namespace: req.Namespace,
				},
				Spec: batchv1.CronJobSpec{
					TimeZone:                   &timezone,
					Schedule:                   start,
					ConcurrencyPolicy:          batchv1.ForbidConcurrent,
					SuccessfulJobsHistoryLimit: pointer.Int32(1),
					FailedJobsHistoryLimit:     pointer.Int32(1),
					JobTemplate: batchv1.JobTemplateSpec{
						Spec: batchv1.JobSpec{
							Template: corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									ServiceAccountName: "autoscaler-" + name,
									Containers: []corev1.Container{
										{
											Name:  "autoscaler-main",
											Image: kubectlImage,
											Command: []string{
												"/bin/sh",
												"-c",
												"kubectl scale" + resourceType + " " + name + " --replicas=" + strconv.Itoa(int(DesiredReplicas)),
											},
										},
									},
									RestartPolicy: corev1.RestartPolicyOnFailure,
								},
							},
						},
					},
				},
			}
			if err := r.Create(ctx, cronjob); err != nil {
				// Failed to create the cronjob
				return ctrl.Result{}, err
			}
		} else if err != nil {
			// Error occurred while checking the cronjob
			return ctrl.Result{}, err
		}

		// scale down the kubernetes resources to minimum replicas
		// check if cronjob is already created for the autoscaler
		cronjob = &batchv1.CronJob{}
		err = r.Get(ctx, types.NamespacedName{Name: "autoscaler-" + name + "-scaledown", Namespace: req.Namespace}, cronjob)
		if err != nil && errors.IsNotFound(err) {
			// cronjob does not exist, create it
			cronjob = &batchv1.CronJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "autoscaler-" + name + "-scaledown",
					Namespace: req.Namespace,
				},
				Spec: batchv1.CronJobSpec{
					TimeZone:                   &timezone,
					Schedule:                   end,
					ConcurrencyPolicy:          batchv1.ForbidConcurrent,
					SuccessfulJobsHistoryLimit: pointer.Int32(1),
					FailedJobsHistoryLimit:     pointer.Int32(1),
					JobTemplate: batchv1.JobTemplateSpec{
						Spec: batchv1.JobSpec{
							Template: corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									ServiceAccountName: "autoscaler-" + name,
									Containers: []corev1.Container{
										{
											Name:  "autoscaler-main",
											Image: kubectlImage,
											Command: []string{
												"/bin/sh",
												"-c",
												"kubectl scale" + resourceType + " " + name + " --replicas=" + strconv.Itoa(int(minReplicaCount)),
											},
										},
									},
									RestartPolicy: corev1.RestartPolicyOnFailure,
								},
							},
						},
					},
				},
			}
			if err := r.Create(ctx, cronjob); err != nil {
				// Failed to create the cronjob
				return ctrl.Result{}, err
			}
		} else if err != nil {
			// Error occurred while checking the cronjob
			return ctrl.Result{}, err
		}
	}

	// update the Status.LastScaleTime of the AutoScaler object
	as.Status.LastScaleTime = metav1.Now()
	if err := r.Status().Update(ctx, as); err != nil {
		// Failed to update the status of the AutoScaler object
		return ctrl.Result{}, err
	}

	// update the Status.CurretReplicas of the AutoScaler object
	// get the current replicas of the resource
	currentReplicas := int32(0)
	if resourceType == "deployment" {
		deployment := &appsv1.Deployment{}
		err = r.Get(ctx, types.NamespacedName{Name: name, Namespace: req.Namespace}, deployment)
		if err != nil {
			// Error occurred while getting the deployment
			return ctrl.Result{}, err
		}
		currentReplicas = deployment.Status.Replicas
	} else if resourceType == "statefulset" {
		statefulset := &appsv1.StatefulSet{}
		err = r.Get(ctx, types.NamespacedName{Name: name, Namespace: req.Namespace}, statefulset)
		if err != nil {
			// Error occurred while getting the statefulset
			return ctrl.Result{}, err
		}
		currentReplicas = statefulset.Status.Replicas
	}
	as.Status.CurrentReplicas = currentReplicas
	if err := r.Status().Update(ctx, as); err != nil {
		// Failed to update the status of the AutoScaler object
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *AutoScalerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8sv1beta1.AutoScaler{}).
		Owns(&batchv1.CronJob{}). // Watch CronJob resources
		Complete(r)
}
