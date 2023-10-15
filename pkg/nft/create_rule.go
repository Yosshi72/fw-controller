package nft

import (
	"github.com/google/nftables"
	"github.com/google/nftables/expr"
	"github.com/sbezverk/nftableslib"
)

// ruleを追加するchain, address, policyを設定
func CreatePrefixRule(chain *nftables.Chain, address string, kind expr.VerdictKind) (*nftables.Rule, error) {
	var tmp *nftables.Rule
	ip, _ := nftableslib.NewIPAddr(address)
	ipaddr := []uint8(ip.IPAddr.IP)
	var version int
	if len(ipaddr) == 16 {
		version = 6
	} else if len(ipaddr) == 4 {
		version = 4
	}
	prefix := int(*ip.Mask)
	mask, err := GenerateBitmask(prefix, version)
	if err != nil {
		return nil, err
	}

	switch version {
	case 6:
		tmp = PrefixIPv6InputRule
	case 4:
		tmp = PrefixIPv4InputRule
	}
	tmp.Exprs[3].(*expr.Bitwise).Mask = mask
	tmp.Exprs[4].(*expr.Cmp).Data = ipaddr

	tmp.Chain = chain
	tmp.Exprs[5].(*expr.Verdict).Kind = kind

	return tmp, nil
}

func CreateCtstateRule(chain *nftables.Chain, ctstate int, kind expr.VerdictKind) (*nftables.Rule, error) {
	tmp := CtStateRule
	tmp.Chain = chain
	tmp.Exprs[3].(*expr.Verdict).Kind = kind
	tmp.Exprs[1].(*expr.Bitwise).Mask[0] = byte(ctstate)

	return tmp, nil
}

func CreateDefaultPolicyRule(chain *nftables.Chain, kind expr.VerdictKind) (*nftables.Rule, error) {
	tmp := DefaultPolicyRule
	tmp.Chain = chain
	tmp.Exprs[0].(*expr.Verdict).Kind = kind

	return tmp, nil
}

func CreateInterfacePolicyRule(chain *nftables.Chain, oflag bool, ifname string, jumpToChain string) (*nftables.Rule, error) {
	var tmp *nftables.Rule
	if oflag {
		tmp = OifJumpRule
	} else {
		tmp = IifJumpRule
	}

	tmp.Chain = chain
	tmp.Exprs[1].(*expr.Cmp).Data = []byte(PadStringToLength16(ifname))
	tmp.Exprs[2].(*expr.Verdict).Chain = jumpToChain

	return tmp, nil
}

func CreateProtocolPolicyRule(chain *nftables.Chain, version int, proto int, kind expr.VerdictKind) (*nftables.Rule, error) {
	var tmp *nftables.Rule
	switch version {
		case 4:
			tmp = ProtocolRule
		case 6:
			tmp = NexthdrRule
	}

	tmp.Chain = chain
	tmp.Exprs[3].(*expr.Cmp).Data = []uint8{byte(proto)}
	tmp.Exprs[4].(*expr.Verdict).Kind = kind

	return tmp, nil
}

func CreateEstablishedOnlyRuleset(chain *nftables.Chain) ([]*nftables.Rule, error) {
	tmp_ruleset := EstablishedOnlyRuleset
	for _, rule := range tmp_ruleset {
		rule.Chain = chain
		switch getRuleType(rule) {
		case protoRule:
			var version int
			switch int(rule.Exprs[1].(*expr.Cmp).Data[0]) {
				case 2:
					version = 4
				case 10:
					version = 6
			}	
			rule, _ = CreateProtocolPolicyRule(chain, version, 1, expr.VerdictReturn)
		case ctstateRule:
			rule, _ = CreateCtstateRule(chain, 6, expr.VerdictReturn)
		case policyRule:
			rule, _ = CreateDefaultPolicyRule(chain, expr.VerdictDrop)
		}
	}
	return tmp_ruleset, nil
}

func CreateAllPermitRuleset(chain *nftables.Chain) ([]*nftables.Rule, error) {
	tmp_ruleset := AllPermitRuleset
	for _, rule := range tmp_ruleset {
		rule.Chain = chain
		switch getRuleType(rule) {
		case policyRule:
			rule, _ = CreateDefaultPolicyRule(chain, expr.VerdictReturn)
		}
	}

	return tmp_ruleset, nil
}
