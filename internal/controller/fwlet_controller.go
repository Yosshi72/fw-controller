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
	"os"
	"strings"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	samplecontrollerv1 "github.com/Yosshi72/fw-controller/api/v1"
	"github.com/Yosshi72/fw-controller/pkg/executer"
	"github.com/Yosshi72/fw-controller/pkg/fwconfig"
	"github.com/Yosshi72/fw-controller/pkg/util"
	// "github.com/k0kubun/pp"
)

// FwLetReconciler reconciles a FwLet object
type FwLetReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=samplecontroller.yossy.vsix.wide.ad.jp,resources=fwlets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=samplecontroller.yossy.vsix.wide.ad.jp,resources=fwlets/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=samplecontroller.yossy.vsix.wide.ad.jp,resources=fwlets/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the FwLet object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.15.0/pkg/reconcile
func (r *FwLetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	res := util.NewResult()
	fwl := samplecontrollerv1.FwLet{}
	region := os.Getenv("REGION")

	if err := r.Get(ctx, req.NamespacedName, &fwl); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		log.Error(err, "msg", "line", util.LINE())
		return ctrl.Result{}, err
	}
	// Ignore reconcile request unrelated to me
	if fwl.GetName() != region {
		return ctrl.Result{}, nil
	}
	containerName := convContainerName(fwl.GetName())
	// Finalizer
	// finalizerName := "bg-switcherlet-" + fwl.Name
	// if fwl.ObjectMeta.DeletionTimestamp.IsZero() {
	// 	if !controllerutil.ContainsFinalizer(&fwl, finalizerName) {
	// 		controllerutil.AddFinalizer(&fwl, finalizerName)
	// 		res.SpecUpdated = true
	// 	}
	// } else {
	// 	if controllerutil.ContainsFinalizer(&fwl, finalizerName) {
	//		TODO:nsのnftablesのrulesを削除する. fw-letが消された場合．
	// 		TODO:fw-masterが消されたらfw-letを消す
	// 		controllerutil.RemoveFinalizer(&fwl, finalizerName)
	// 		if err := r.Update(ctx, &fwl); err != nil {
	// 			log.Error(err, "msg", "line", util.LINE())
	// 			return ctrl.Result{}, err
	// 		}
	// 	}
	// 	return ctrl.Result{}, nil
	// }

	trustIf, untrustIf, mgmtAddr, err := getConfig(containerName)
	if err != nil {
		log.Error(err, "msg", "line", util.LINE())
		return ctrl.Result{}, err
	}
	if trustIf == nil && untrustIf == "" && mgmtAddr == nil {
		fwl.Status.TrustIf = fwl.Spec.TrustIf
		fwl.Status.UntrustIf = fwl.Spec.UntrustIf
		fwl.Status.MgmtAddressRange = fwl.Spec.MgmtAddressRange
		setConfig(containerName, fwl.Status.UntrustIf, fwl.Status.TrustIf, fwl.Status.MgmtAddressRange)
	}

	trustIf, untrustIf, mgmtAddr, err = getConfig(containerName)

	// Interfaceの更新
	changedTrustIF := false
	if !fwconfig.MatchElements(trustIf, fwl.Spec.TrustIf) {
		newTrustIF := fwl.Spec.TrustIf
		setConfig(containerName, untrustIf, newTrustIF, mgmtAddr)
		changedTrustIF = true
	}
	if changedTrustIF {
		trustIf, untrustIf, mgmtAddr, err = getConfig(containerName)
		if err != nil {
			log.Error(err, "msg", "line", util.LINE())
			return ctrl.Result{}, err
		}
		fwl.Status.TrustIf = trustIf
		res.StatusUpdated = true
	}

	changedUntrustIF := false
	if untrustIf != fwl.Spec.UntrustIf {
		newUntrustIF := fwl.Spec.UntrustIf
		setConfig(containerName, newUntrustIF, trustIf, mgmtAddr)
		changedUntrustIF = true
	}
	if changedUntrustIF {
		trustIf, untrustIf, mgmtAddr, err = getConfig(containerName)
		if err != nil {
			log.Error(err, "msg", "line", util.LINE())
			return ctrl.Result{}, err
		}
		fwl.Status.UntrustIf = untrustIf
		res.StatusUpdated = true
	}

	changedMgmtAddr := false
	if !fwconfig.MatchElements(mgmtAddr, fwl.Spec.MgmtAddressRange) {
		newMgmtAddr := fwl.Spec.MgmtAddressRange
		setConfig(containerName, untrustIf, trustIf, newMgmtAddr)
		changedMgmtAddr = true
	}
	if changedMgmtAddr {
		trustIf, untrustIf, mgmtAddr, err = getConfig(containerName)
		if err != nil {
			log.Error(err, "msg", "line", util.LINE())
			return ctrl.Result{}, err
		}
		fwl.Status.MgmtAddressRange = mgmtAddr
		res.StatusUpdated = true
	}

	if res.SpecUpdated {
		if err := r.Update(ctx, &fwl); err != nil {
			log.Error(err, "msg", "line", util.LINE())
			return ctrl.Result{}, err
		}
	}

	if res.StatusUpdated {
		if err := r.Status().Update(ctx, &fwl); err != nil {
			log.Error(err, "msg", "line", util.LINE())
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func getConfig(containerName string) ([]string, string, []string, error) {
	// TODO: config.jsonのパスを入れる
	trustIn, untrustIn, mgmtAddr, err := fwconfig.RulesReader("/etc/nftables/fw.rule")

	if err != nil {
		return nil, "", nil, err
	}

	return trustIn, untrustIn, mgmtAddr, nil
}

func setConfig(containerName, untrustif_name string, trustif_name, mgmtaddress []string) error {
	// update fwconfig.json
	err := fwconfig.RuleUpdate(
		containerName,
		"/etc/nftables/fw-template.rule",
		"/etc/nftables/fw.rule",
		untrustif_name,
		trustif_name,
		mgmtaddress,
	)
	if err != nil {
		return err
	}
	err= executer.ExecCommand(
		containerName,
	)
	return err
}

func convContainerName(rawName string) string {
	runes := []rune(rawName)
	runes[0] = []rune(strings.ToUpper(string(runes[0])))[0]
	runes[len(runes)-1] = []rune(strings.ToUpper(string(runes[len(runes)-1])))[0]
	return string(runes)
}

// SetupWithManager sets up the controller with the Manager.
func (r *FwLetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&samplecontrollerv1.FwLet{}).
		Complete(r)
}
