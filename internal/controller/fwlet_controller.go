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
	"fmt"
	"os"
	"reflect"
	"strings"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/k0kubun/pp"
	vsix "github.com/wide-vsix/kloudnfv/api/v1"
	"github.com/wide-vsix/kloudnfv/pkg/nft"
	"github.com/wide-vsix/kloudnfv/pkg/util"
)

// FwRouterReconciler reconciles a FwRouter object
type FwRouterReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=vsix.wide.ad.jp,resources=fwrouters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=vsix.wide.ad.jp,resources=fwrouters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=vsix.wide.ad.jp,resources=fwrouters/finalizers,verbs=update

func (r *FwRouterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	res := util.NewResult()
	name := os.Getenv("REGION")
	nsname := os.Getenv("NSNAME")

	fwrouter := vsix.FwRouter{}

	if err := r.Get(ctx, req.NamespacedName, &fwrouter); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		log.Error(err, "msg", "line", util.LINE())
		return ctrl.Result{}, err
	}

	// Ignore reconcile request unrelated to me
	if fwrouter.GetName() != name {
		return ctrl.Result{}, nil
	}

	// fwrouterのspecとstatusでzoneの齟齬がないか
	allok := true
	for zoneNameSpec := range fwrouter.Spec.Zones {
		foundZoneNameInStatus := false
		for zoneNameStatus := range fwrouter.Status.Zones {
			if zoneNameSpec == zoneNameStatus {
				foundZoneNameInStatus = true
			}
		}
		if !foundZoneNameInStatus {
			if fwrouter.Status.Zones == nil {
				fwrouter.Status.Zones = make(map[vsix.ZoneName]vsix.FwZoneStatus)
			}
			newZoneStatus := vsix.FwZoneStatus{
				Interfaces:       fwrouter.Spec.Zones[zoneNameSpec].Interfaces,
				Policy:           fwrouter.Spec.Zones[zoneNameSpec].Policy,
				AllowPrefixNames: fwrouter.Spec.Zones[zoneNameSpec].AllowPrefixNames,
				Created:          true,
			}
			fwrouter.Status.Zones[zoneNameSpec] = newZoneStatus
			allok = false
			res.StatusUpdated = true
		}
	}

	if !allok {
		if res.SpecUpdated {
			if err := r.Update(ctx, &fwrouter); err != nil {
				log.Error(err, "msg", "line", util.LINE())
				return ctrl.Result{Requeue: true}, err
			}
		}
		if res.StatusUpdated {
			if err := r.Status().Update(ctx, &fwrouter); err != nil {
				log.Error(err, "msg", "line", util.LINE())
				return ctrl.Result{Requeue: true}, err
			}
		}
		return ctrl.Result{Requeue: true}, nil
	}

	for zoneName, zoneStatus := range fwrouter.Status.Zones {
		zoneSpec := fwrouter.Spec.Zones[zoneName]
		currentIf, currentPolicy, _, err := getNftTables(nsname, zoneName)
		if err != nil {
			log.Error(err, "msg", "line", util.LINE())
			return ctrl.Result{}, err
		}

		// nftablesの値とSpecの値を比較．
		// 違いがあれば,Specの値でnftablesのアップデート.
		// その後,nftablesの値でStatusアップデート.
		// interfaceのチェック
		iflag, pflag := false, false
		if !util.MatchElements(currentIf, zoneSpec.Interfaces) {
			addIf, delIf := util.CmpElements(zoneSpec.Interfaces, currentIf)
			updateNftable(nsname, zoneName, addIf, delIf)
			pp.Println("interface update.")
			iflag = true
		}
		if iflag {
			currentIf, _, _, err = getNftTables(nsname, zoneName)
			if err != nil {
				log.Error(err, "msg", "line", util.LINE())
				return ctrl.Result{}, err
			}
			zoneStatus.Interfaces = currentIf
			fwrouter.Status.Zones[zoneName] = zoneStatus
			res.StatusUpdated = true
		}
		// policyのチェック
		if currentPolicy != zoneSpec.Policy {
			addPolicy, delPolicy := zoneSpec.Policy, currentPolicy
			updateNftable(nsname, zoneName, addPolicy, delPolicy)
			pp.Println("policy update.")
			pflag = true
		}
		if pflag {
			_, currentPolicy, _, err = getNftTables(nsname, zoneName)
			if err != nil {
				log.Error(err, "msg", "line", util.LINE())
				return ctrl.Result{}, err
			}
			zoneStatus.Policy = currentPolicy
			fwrouter.Status.Zones[zoneName] = zoneStatus
			res.StatusUpdated = true
		}

		// allowPrefixNamesが更新された場合
		preFlag := false
		if !util.MatchElements(zoneStatus.AllowPrefixNames, zoneSpec.AllowPrefixNames) {
			addPrefixNames, delPrefixNames := util.CmpElements(zoneSpec.AllowPrefixNames, zoneStatus.AllowPrefixNames)

			// addPrefixNameと一致するprefix-nameのprefix-addressとdelPrefixNameのprefix-addressを取得
			addPrefixAddress, err := r.getPrefixAddressList(ctx, req, addPrefixNames)
			if err != nil {
				log.Error(err, "msg", "line", util.LINE())
				return ctrl.Result{}, err
			}
			delPrefixAddress, err := r.getPrefixAddressList(ctx, req, delPrefixNames)
			if err != nil {
				log.Error(err, "msg", "line", util.LINE())
				return ctrl.Result{}, err
			}
			// これら二つのアドレスでCmpElementをして，追加すべきアドレスと消去すべきアドレスをゲット
			addAddr, delAddr := util.CmpElements(addPrefixAddress, delPrefixAddress)

			conn, fd, err := nft.InitConn(nsname)
			if err != nil {
				log.Error(err, "msg", "line", util.LINE())
				return ctrl.Result{}, err
			}
			defer nft.CloseConn(fd)

			// nftablesのアップデート
			err = nft.UpdatePrefixAddressesList(conn, nsname, zoneName, addAddr, delAddr) 
			pp.Println("prefix-name update.")
			if err != nil {
				log.Error(err, "msg", "line", util.LINE())
				return ctrl.Result{}, err
			}
			preFlag = true
		}
		if preFlag {
			// TODO: nftableからsaddr取得して，SpecのAllowPrefixNamesに対応してるかチェック
			_, _, _, _ = getNftTables(nsname, zoneName)

			zoneStatus.AllowPrefixNames = zoneSpec.AllowPrefixNames
			fwrouter.Status.Zones[zoneName] = zoneStatus
			res.StatusUpdated = true
		}

		// fwprefixlistに変更があった場合
		// nftablesからprefixAddressを取ってくる
		_, _, currentPrefixAddress, err := getNftTables(nsname, zoneName)
		if err != nil {
			log.Error(err, "msg", "line", util.LINE())
			return ctrl.Result{}, err
		}
		newPrefixAddress, err := r.getPrefixAddressList(ctx, req, zoneStatus.AllowPrefixNames)
		if err != nil {
			log.Error(err, "msg", "line", util.LINE())
			return ctrl.Result{}, err
		}
		// nftablesの値とfwprefixのSpecのprefixAddressを比較
		if !util.MatchElements(currentPrefixAddress, newPrefixAddress) {
			addPrefixAddress, delPrefixAddress := util.CmpElements(newPrefixAddress, currentPrefixAddress)

			conn, fd, err := nft.InitConn(nsname)
			if err != nil {
				log.Error(err, "msg", "line", util.LINE())
				return ctrl.Result{}, err
			}
			defer nft.CloseConn(fd)

			err = nft.UpdatePrefixAddressesList(conn, nsname, zoneName, addPrefixAddress, delPrefixAddress) 
			pp.Println("prefix-address-list update.")
			if err != nil {
				log.Error(err, "msg", "line", util.LINE())
				return ctrl.Result{}, err
			}
		}
	}

	if res.SpecUpdated {
		if err := r.Update(ctx, &fwrouter); err != nil {
			log.Error(err, "msg", "line", util.LINE())
			return ctrl.Result{Requeue: true}, err
		}
	}
	if res.StatusUpdated {
		if err := r.Status().Update(ctx, &fwrouter); err != nil {
			log.Error(err, "msg", "line", util.LINE())
			return ctrl.Result{Requeue: true}, err
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *FwRouterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&vsix.FwRouter{}).
		Watches(
			&vsix.FwPrefixList{},
			handler.EnqueueRequestsFromMapFunc(r.findFwRouterByPrefixName),
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
		).
		Complete(r)
}


func (r *FwRouterReconciler) findFwRouterByPrefixName(ctx context.Context, prefixName client.Object) []reconcile.Request {
	reqs := []reconcile.Request{}
	// ctx := context.TODO()
	log := log.FromContext(ctx)
	namespace := prefixName.GetNamespace()
	routerName := os.Getenv("REGION")

	// get updated prefixName
	updatedPrefix := vsix.FwPrefixList{}
	updatedPrefixName := prefixName.GetName()
	if err := r.Get(ctx, client.ObjectKey{Namespace: namespace, Name: prefixName.GetName()}, &updatedPrefix); err != nil {
		// 対象のprefixNameが削除された
		log.Info("PrefixList deleted: prefixName is ", updatedPrefixName)
		reqs = append(reqs, reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      routerName,
				Namespace: namespace,
			},
		})
		return reqs
	}

	// fwrouterを取得
	fwrouter := vsix.FwRouter{}
	if err := r.Get(ctx, client.ObjectKey{Namespace: namespace, Name: routerName}, &fwrouter); err != nil{
		log.Error(err, "msg", "line", util.LINE())
		return reqs
	}
	specZones := fwrouter.Spec.Zones
	for _, specZone := range specZones {
		specPrefixNames := specZone.AllowPrefixNames
		for _, prefixName := range specPrefixNames {
			// udpateされたprefixListがrouterのAllowedPrefixNamesに含まれていたら
			prefixName = strings.ToLower(prefixName)
			if prefixName == updatedPrefixName {
				reqs = append(reqs, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name: fwrouter.GetName(),
						Namespace: fwrouter.GetNamespace(),
					},
				})
			}
		}
	}
	return reqs
}
func getNftTables(nsname string, zoneName vsix.ZoneName) ([]string, vsix.ZonePolicy, []string, error) {
	conn, fd, err := nft.InitConn(nsname)
	if err != nil {
		return nil, "", nil, err
	}
	defer nft.CloseConn(fd)

	ifs, err := nft.GetInterfaces(conn, zoneName)
	if err != nil {
		return ifs, "", nil, err
	}
	policy, err := nft.GetPolicy(conn, zoneName)
	if err != nil {
		return ifs, policy, nil, err
	}

	addresses, err := nft.GetAddresses(conn, zoneName)
	if err != nil {
		return ifs, policy, addresses, err
	}
	return ifs, policy, addresses, nil
}

func updateNftable(nsname string, zoneName vsix.ZoneName, addElement interface{}, delElement interface{}) error {
	if reflect.TypeOf(addElement) != reflect.TypeOf(delElement) {
		err := fmt.Errorf("Type of addElement != Type of delElement")
		return err
	}

	conn, fd, err := nft.InitConn(nsname)
	if err != nil {
		return err
	}
	defer nft.CloseConn(fd)

	switch addElement.(type) {
	case []string:
		addStrings, ok1 := addElement.([]string)
		delStrings, ok2 := delElement.([]string)
		if !ok1 || !ok2 {
			return fmt.Errorf("Failed to cast to []string")
		}
		err := nft.UpdateInterfaces(nsname, conn, zoneName, addStrings, delStrings)
		if err != nil {
			msg := fmt.Errorf("Failed to update interfaces: %v", err)
			return msg
		}
	case vsix.ZonePolicy:
		newZonePolicy, ok1 := addElement.(vsix.ZonePolicy)
		if !ok1 {
			return fmt.Errorf("Failed to cast to vsix.ZonePolicy")
		}
		err := nft.UpdateZonePolicy(conn, zoneName, newZonePolicy)
		if err != nil {
			msg := fmt.Errorf("Failed to update zonepolicy: %v", err)
			return msg
		}
	}
	return nil
}

// PrefixNameからPrefixAddressListを取得
func (r *FwRouterReconciler) getPrefixAddressList(ctx context.Context, req ctrl.Request, prefixNames []string) ([]string, error) {
	prefixList := &vsix.FwPrefixList{}
	var prefixAddressList []string
	for _, prefixName := range prefixNames {
		prefixName = strings.ToLower(prefixName)
		keys := client.ObjectKey{Namespace: req.Namespace, Name: prefixName}

		if err := r.Get(ctx, keys, prefixList); err != nil {
			msg := fmt.Errorf("Failed to get prefixAddressList: %v", err)
			return nil, msg
		}
		prefixAddressList = append(prefixAddressList, prefixList.Spec.Prefixes...)
	}

	return prefixAddressList, nil
}
