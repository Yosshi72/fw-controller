package nft

import (
	"fmt"
	vsix "github.com/wide-vsix/kloudnfv/api/v1"
	"os"
	"strings"
	"syscall"

	"github.com/google/nftables"
	"github.com/google/nftables/expr"

	// "github.com/k0kubun/pp"
)

func GetFdFromNsname(netns string) (int, error) {
	netnsPath := fmt.Sprintf("/var/run/netns/%s", netns)

	fd, err := syscall.Open(netnsPath, os.O_RDONLY, 0)
    if err != nil {
        msg := fmt.Errorf("Failed to find netns %s: %v", netns, err)
        return -1, msg
    }
	return fd, nil
}
func InitConn(netns ...string) (*nftables.Conn, int, error) {
	var conn *nftables.Conn
	var fdnum int
	switch len(netns) {
		case 0:
			conn = &nftables.Conn{}
		case 1:
			fd, err := GetFdFromNsname(netns[0])
			if err != nil {
				msg := fmt.Errorf("Failed to get fd from nsname %s: %v", netns[0], err)
				return nil, fd, msg
			}
			c, err := nftables.New(nftables.WithNetNSFd(int(fd)), nftables.AsLasting())
			if err != nil {
				msg := fmt.Errorf("Failed to create netlink connection: %v", err)
				return nil, fd, msg
			}
			conn = c
			fdnum = fd
		default:
			msg := fmt.Errorf("the number of arguments should be smaller than 1")
			return nil, 0, msg
	}
	return conn, fdnum, nil
}

func CloseConn(fd int) error {
	err := syscall.Close(fd)
    if err != nil {
        msg := fmt.Errorf("Failed to close netns: %v", err)
		return msg
    }
	return nil
}

// nftablesからinterfacesの情報を取ってくる
func GetInterfaces(conn *nftables.Conn, zoneName vsix.ZoneName) ([]string, error) {
	table := VsixTable
	chain := findChain("Forward")
	if chain == nil {
		msg := fmt.Errorf("Failed to find chain")
		return nil, msg
	}

	rules, err := conn.GetRules(table, chain)
	if err != nil {
		msg := fmt.Errorf("Failed to get rules: %v", err)
		return nil, msg
	}

	var ifnames []string
	for i := range rules {
		if getRuleType(rules[i]) == ifRule {
			ifname := findInterfaceInNft(rules[i], zoneName)
			if ifname != "" {
				ifnames = append(ifnames, ifname)
			}
		}
	}

	return ifnames, nil
}

// nftablesからpolicyの情報を取ってくる
// PAIR_UNTRUST_TO_TRUST, PAIR_TRUST_TO_UNTRUSTは2パターンのルールセットしかないと仮定
func GetPolicy(conn *nftables.Conn, zoneName vsix.ZoneName) (vsix.ZonePolicy, error) {
	table := VsixTable
	var chain *nftables.Chain
	var currentPolicy vsix.ZonePolicy
	switch zoneName {
	case "trust":
		chain = findChain("PAIR_UNTRUST_TO_TRUST")
	case "untrust":
		chain = findChain("PAIR_TRUST_TO_UNTRUST")
	}
	if chain == nil {
		msg := fmt.Errorf("Failed to find chain")
		return "", msg
	}

	rules, err := conn.GetRules(table, chain)
	if err != nil {
		msg := fmt.Errorf("Failed to get rules: %v", err)
		return "", msg
	}
	for i := range rules {
		if getRuleType(rules[i]) == policyRule {
			currentPolicy = findPolicyInNft(rules[i])
		}
	}

	return currentPolicy, nil
}

func GetAddresses(conn *nftables.Conn, zoneName vsix.ZoneName) ([]string, error) {
	table := VsixTable
	var chain *nftables.Chain
	switch zoneName {
	case "trust":
		chain = findChain("PAIR_UNTRUST_TO_TRUST")
	case "untrust":
		chain = findChain("PAIR_TRUST_TO_UNTRUST")
	}
	if chain == nil {
		msg := fmt.Errorf("Failed to find chain")
		return nil, msg
	}

	rules, err := conn.GetRules(table, chain)
	if err != nil {
		msg := fmt.Errorf("Failed to get rules: %v", err)
		return nil, msg
	}
	var addresses []string
	for i := range rules {
		if getRuleType(rules[i]) == prefixRule { // address系の命令
			addr := findPrefixInNft(rules[i])
			if addr != "" {
				addresses = append(addresses, addr)
			}
		}
	}

	return addresses, nil
}

// nftablesでinterfacesのアップデート
func UpdateInterfaces(nsname string, conn *nftables.Conn, zoneName vsix.ZoneName, addIf, delIf []string) error {
	table := VsixTable
	conn.AddTable(table)
	chains := []*nftables.Chain{ForwardChain, ZoneTrustChain, ZoneUnrustChain}

	for _, chain := range chains {
		conn, fd , _:= InitConn(nsname)
        defer CloseConn(fd)
		conn.AddChain(chain)
		// 現在のrulesetを取得
		rules, err := conn.GetRules(table, chain)
		if err != nil {
			msg := fmt.Errorf("Failed to get rules: %v", err)
			return msg
		}

		// ruleの削除
		for _, rule := range rules {
			if getRuleType(rule) == ifRule {
				data := rule.Exprs[1].(*expr.Cmp)
				ifname := strings.ReplaceAll(string(data.Data), "\x00", "")
				if contains(delIf, ifname) {
					conn.DelRule(rule)
					if err := conn.Flush(); err != nil {
						msg := fmt.Errorf("Failed to flush interface-rules to kernel: %v", err)
						return msg
					}
				}
			}
		}

		// ruleの追加
		var addRule *nftables.Rule
		for _, ifname := range addIf {
			if chain.Name == "Forward" {
				if zoneName == "trust" {
					addRule, err = CreateInterfacePolicyRule(chain, true, ifname, "ZONE_TRUST")
				} else if zoneName == "untrust" {
					addRule, err = CreateInterfacePolicyRule(chain, true, ifname, "ZONE_UNTRUST")
				}
			} else if chain.Name == "ZONE_TRUST" {
				if zoneName == "trust" {
					addRule, err = CreateInterfacePolicyRule(chain, false, ifname, "PAIR_TRUST_TO_TRUST")
				} else if zoneName == "untrust" {
					addRule, err = CreateInterfacePolicyRule(chain, false, ifname, "PAIR_UNTRUST_TO_TRUST")
				}
			} else if chain.Name == "ZONE_UNTRUST" {
				addRule = IifJumpRule
				addRule.Chain = chain
				if zoneName == "trust" {
					addRule, err = CreateInterfacePolicyRule(chain, false, ifname, "PAIR_TRUST_TO_UNTRUST")
				} else if zoneName == "untrust" {
					addRule, err = CreateInterfacePolicyRule(chain, false, ifname, "PAIR_UNTRUST_TO_UNTRUST")
				}
			}
			conn.AddRule(addRule)
			if err := conn.Flush(); err != nil {
				msg := fmt.Errorf("Failed to flush interface-rules to kernel: %v", err)
				return msg
			}
		}
	}
	return nil
}

// nftablesでzonePolicyのアップデート
func UpdateZonePolicy(conn *nftables.Conn, zoneName vsix.ZoneName, newPolicy vsix.ZonePolicy) error {
	table := VsixTable
	var chain *nftables.Chain
	switch zoneName {
	case "trust":
		chain = findChain("PAIR_UNTRUST_TO_TRUST")
	case "untrust":
		chain = findChain("PAIR_TRUST_TO_UNTRUST")
	}
	rules, err := conn.GetRules(table, chain)
	if err != nil {
		msg := fmt.Errorf("Failed to get rules: %v", err)
		return msg
	}
	// 既存のルールセットを一旦削除
	for _, rule := range rules {
		conn.DelRule(rule)
		if err := conn.Flush(); err != nil {
			msg := fmt.Errorf("Failed to flush interface-rules to kernel: %v", err)
			return msg
		}
	}

	// new policyに紐づいたルールセットをnftableに適用
	var newrules []*nftables.Rule
	switch newPolicy {
	case vsix.EstablishedOnly:
		newrules, err = CreateEstablishedOnlyRuleset(chain)
	case vsix.AllPermit:
		newrules, err = CreateAllPermitRuleset(chain)
	}
	if err != nil {
		msg := fmt.Errorf("Failed to create new policy ruleset")
		return msg
	}
	for _, rule := range newrules {
		conn.AddRule(rule)
		if err := conn.Flush(); err != nil {
			msg := fmt.Errorf("Failed to flush interface-rules to kernel: %v", err)
			return msg
		}
	}
	return nil
}


func UpdatePrefixAddressesList(conn *nftables.Conn, nsname string, zoneName vsix.ZoneName, addStrAddressesList, delStrAddressesList []string) error {
	addAddressesList, err := StrToIP(addStrAddressesList)
	if err != nil {
		return err
	}
	delAddressesList, err := StrToIP(delStrAddressesList)
	if err != nil {
		return err
	}

	table := VsixTable
	var chain *nftables.Chain
	switch zoneName {
	case "trust":
		chain = findChain("PAIR_UNTRUST_TO_TRUST")
	case "untrust":
		chain = findChain("PAIR_TRUST_TO_UNTRUST")
	}

	// 現在のrulesetを取得
	rules, err := conn.GetRules(table, chain)
	if err != nil {
		msg := fmt.Errorf("Failed to get rules: %v", err)
		return msg
	}

	// ruleの削除
	for _, rule := range rules {
		if getRuleType(rule) == prefixRule {
			addr := findPrefixInNft(rule)
			if contains(delAddressesList, addr) || contains(addAddressesList, addr) {
				conn.DelRule(rule)
				if err := conn.Flush(); err != nil {
					msg := fmt.Errorf("Failed to flush interface-rules to kernel: %v", err)
					return msg
				}
			}
		}
	}

	// ruleの追加
	for _, addAddr := range addAddressesList {
		addRule, err := CreatePrefixRule(chain, addAddr, expr.VerdictAccept)
		if err != nil {
			msg := fmt.Errorf("Failed to create prefix-rule: %v", err)
			return msg
		}
		conn.AddRule(addRule)
		if err := conn.Flush(); err != nil {
			msg := fmt.Errorf("Failed to flush interface-rules to kernel: %v", err)
			return msg
		}
	}

	return nil
}
