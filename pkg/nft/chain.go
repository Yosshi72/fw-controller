package nft

import (
	"github.com/google/nftables"
)

var policyDrop = nftables.ChainPolicyDrop
var policyAccept = nftables.ChainPolicyAccept

func InitChain(conn *nftables.Conn) error {
	if err := InitTable(conn); err != nil {
		return err
	}
	for _, chain := range ChainList {
		conn.AddChain(chain)
		if err := conn.Flush(); err != nil {
			return err
		}
	}
	return nil
}

// Chainの追加・削除をしたらここも変更して
var ChainList = []*nftables.Chain{
	InputChain,
	ForwardChain,
	ZoneTrustChain,
	ZoneUnrustChain,
	PairTrustToUntrustChain,
	PairUntustToTrustChain,
	PairTrustToTrustChain,
	PairUntrustToUntrustChain,
}

var InputChain = &nftables.Chain{
	Name:     "Input",
	Table:    VsixTable,
	Type:     nftables.ChainTypeFilter,
	Hooknum:  nftables.ChainHookInput,
	Priority: nftables.ChainPriorityFilter,
	Policy:   &policyAccept,
}

var ForwardChain = &nftables.Chain{
	Name:     "Forward",
	Table:    VsixTable,
	Type:     nftables.ChainTypeFilter,
	Hooknum:  nftables.ChainHookForward,
	Priority: nftables.ChainPriorityFilter,
	Policy:   &policyAccept,
}

var ZoneTrustChain = &nftables.Chain{
	Name:  "ZONE_TRUST",
	Table: VsixTable,
}

var ZoneUnrustChain = &nftables.Chain{
	Name:  "ZONE_UNTRUST",
	Table: VsixTable,
}

var PairUntustToTrustChain = &nftables.Chain{
	Name:  "PAIR_UNTRUST_TO_TRUST",
	Table: VsixTable,
}

var PairTrustToUntrustChain = &nftables.Chain{
	Name:  "PAIR_TRUST_TO_UNTRUST",
	Table: VsixTable,
}

var PairTrustToTrustChain = &nftables.Chain{
	Name:  "PAIR_TRUST_TO_TRUST",
	Table: VsixTable,
}

var PairUntrustToUntrustChain = &nftables.Chain{
	Name:  "PAIR_UNTRUST_TO_UNTRUST",
	Table: VsixTable,
}
