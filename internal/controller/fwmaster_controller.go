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

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	controllerutil "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	samplecontrollerv1 "github.com/Yosshi72/fw-controller/api/v1"
	"github.com/Yosshi72/fw-controller/pkg/util"
)

// FwMasterReconciler reconciles a FwMaster object
type FwMasterReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=samplecontroller.yossy.vsix.wide.ad.jp,resources=fwmasters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=samplecontroller.yossy.vsix.wide.ad.jp,resources=fwmasters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=samplecontroller.yossy.vsix.wide.ad.jp,resources=fwmasters/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the FwMaster object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.15.0/pkg/reconcile
func (r *FwMasterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	res := util.NewResult()
	fwm := samplecontrollerv1.FwMaster{}

	if err := r.Get(ctx, req.NamespacedName, &fwm); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{Requeue: true}, nil
		}
		log.Error(err, "msg", "line", util.LINE())
		return ctrl.Result{Requeue: true}, err
	}

	// SpecとStatusでRegionに齟齬がないか
	allok := true
	for _, regionSpec := range fwm.Spec.Regions {
		foundRegionInStatus := false
		for _, regionStatus := range fwm.Status.Regions {
			if regionStatus.RegionName == regionSpec.RegionName {
				foundRegionInStatus = true
			}
		}
		if !foundRegionInStatus {
			newRegionStatus := samplecontrollerv1.RegionStatus{
				RegionName:       regionSpec.RegionName,
				TrustIf:          regionSpec.TrustIf,
				UntrustIf:        regionSpec.UntrustIf,
				MgmtAddressRange: fwm.Spec.MgmtAddressRange,
				Created:          false,
			}
			err := r.ReconcileFwLet(ctx, fwm, regionSpec, fwm.Spec.MgmtAddressRange)
			if err != nil {
				log.Error(err, "msg", "line", util.LINE())
				return ctrl.Result{Requeue: true}, err
			}
			newRegionStatus.Created = true
			fwm.Status.Regions = append(fwm.Status.Regions, newRegionStatus)
			allok = false
			res.StatusUpdated = true
		}
	}

	for _, regionSpec := range fwm.Spec.Regions {
		for _, regionStatus := range fwm.Status.Regions {
			if regionSpec.RegionName == regionStatus.RegionName {
				err := r.ReconcileFwLet(ctx, fwm, regionSpec, fwm.Spec.MgmtAddressRange)
				if err != nil {
					log.Error(err, "msg", "line", util.LINE())
					return ctrl.Result{Requeue: true}, err
				}
			}
		}
	}
	if !allok {
		if res.SpecUpdated {
			if err := r.Update(ctx, &fwm); err != nil {
				log.Error(err, "msg", "line", util.LINE())
				return ctrl.Result{Requeue: true}, err
			}
		}
		if res.StatusUpdated {
			if err := r.Status().Update(ctx, &fwm); err != nil {
				log.Error(err, "msg", "line", util.LINE())
				return ctrl.Result{Requeue: true}, err
			}
		}
		return ctrl.Result{Requeue: true}, nil
	}

	if res.SpecUpdated {
		if err := r.Update(ctx, &fwm); err != nil {
			log.Error(err, "msg", "line", util.LINE())
			return ctrl.Result{Requeue: true}, err
		}
	}
	if res.StatusUpdated {
		if err := r.Status().Update(ctx, &fwm); err != nil {
			log.Error(err, "msg", "line", util.LINE())
			return ctrl.Result{Requeue: true}, err
		}
	}
	return ctrl.Result{}, nil
}

func (r *FwMasterReconciler) ReconcileFwLet(ctx context.Context, fwm samplecontrollerv1.FwMaster, regionSpec samplecontrollerv1.RegionSpec, MgmtAddressRange []string) error {
	// FwMasterからFwLetへ
	log := log.FromContext(ctx)
	fwl := samplecontrollerv1.FwLet{}
	fwl.SetNamespace(fwm.GetNamespace())
	fwl.SetName(regionSpec.RegionName)

	op, err := ctrl.CreateOrUpdate(ctx, r.Client, &fwl, func() error {
		fwl.Spec.TrustIf = regionSpec.TrustIf
		fwl.Spec.UntrustIf = regionSpec.UntrustIf
		fwl.Spec.MgmtAddressRange = MgmtAddressRange
		return ctrl.SetControllerReference(&fwm, &fwl, r.Scheme)
	})

	if err != nil {
		log.Error(err, "unable to create or update Fwlet resource")
		return err
	}
	if op != controllerutil.OperationResultNone {
		log.Info("reconcile FwLet successfully", "op", op)
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *FwMasterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&samplecontrollerv1.FwMaster{}).
		Owns(&samplecontrollerv1.FwLet{}).
		Complete(r)
}
