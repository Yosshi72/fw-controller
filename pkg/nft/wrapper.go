package nft

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	vsix "github.com/wide-vsix/kloudnfv/api/v1"

	"github.com/google/nftables"
	"github.com/google/nftables/expr"
)

type RuleType string

const (
	policyRule    RuleType = "policyRule"
	ifRule        RuleType = "ifRule"
	ctstateRule   RuleType = "ctstateRule"
	protoRule     RuleType = "protoRule"
	limitRateRule RuleType = "limitRateRule"
	prefixRule    RuleType = "prefixRule"
	otherRule     RuleType = "otherRule"
)

// ruleのexpsの数でrule typeを識別する
func getRuleType(rule *nftables.Rule) RuleType {
	exprs := rule.Exprs
	msgSize := len(exprs)

	var rtype RuleType
	switch msgSize {
	case 1:
		rtype = policyRule
	case 3:
		rtype = ifRule
	case 4:
		rtype = ctstateRule
	case 5:
		rtype = protoRule
	case 6:
		// prefix系ルールかrate-limit系ルール
		switch exprs[3].(type) {
		case *expr.Bitwise:
			rtype = prefixRule
		case *expr.Cmp:
			rtype = limitRateRule
		}
	default:
		rtype = otherRule
	}

	return rtype
}

func findChain(name string) *nftables.Chain {
	chainList := ChainList

	for _, c := range chainList {
		if c.Name == name {
			return c
		}
	}
	return nil
}

// []uint8型のMaskからプレフィクス長を取得
func getPrefixLength(data []uint8) int {
	count := 0
	for _, b := range data {
		if b == 0xff {
			count += 8
			continue
		}
		for i := 7; i >= 0; i-- {
			if b&(1<<uint(i)) != 0 {
				count++
			} else {
				return count
			}
		}
	}
	return count
}

// プレフィクス長から[]utin8型のMask生成
func GenerateBitmask(prefixLen int, version int) ([]uint8, error) {
	switch version {
	case 6:
		var result [16]uint8
		for i := 0; i < prefixLen; i++ {
			result[i/8] |= 1 << (7 - i%8)
		}
		return result[:], nil
	case 4:
		var result [4]uint8
		for i := 0; i < prefixLen; i++ {
			result[i/8] |= 1 << (7 - i%8)
		}
		return result[:], nil
	default:
		msg := fmt.Errorf("Failed to generate bitmask.\n")
		return nil, msg
	}
}

func contains(s []string, searchterm string) bool {
	for _, v := range s {
		if v == searchterm {
			return true
		}
	}
	return false
}

func PadStringToLength16(input string) string {
	padding := make([]byte, 16-len(input))
	return input + string(padding)
}

func StrToIP(strIPs []string) ([]string, error) {
	var ips []string
	for _, strIP := range strIPs {
		_, ip, err := net.ParseCIDR(strIP)
		if err != nil {
			msg := fmt.Errorf("Failed to parce IP Address: %v", err)
			return nil, msg
		}
		addr := ip.String()
		ips = append(ips, addr)
	}
	return ips, nil
}

// nftableのruleからinterface情報を抽出する
func findInterfaceInNft(rule *nftables.Rule, zoneName vsix.ZoneName) string {
	var zone string
	var ifname string
	switch zoneName {
	case vsix.Trust:
		zone = "ZONE_TRUST"
	case vsix.Untrust:
		zone = "ZONE_UNTRUST"
	}

	jumpToChainName := rule.Exprs[2].(*expr.Verdict).Chain
	if jumpToChainName == zone {
		data := rule.Exprs[1].(*expr.Cmp)
		ifname = strings.ReplaceAll(string(data.Data), "\x00", "")
	}

	return ifname
}

// nftableのruleからprefix情報を抽出する．
func findPrefixInNft(rule *nftables.Rule) string {
	// prefix長を求める
	mask := rule.Exprs[3].(*expr.Bitwise).Mask
	num := getPrefixLength(mask)
	prefix := strconv.Itoa(num)

	// IP Addressを取得する
	data := rule.Exprs[4].(*expr.Cmp).Data
	ip := net.IP(data)
	addr := ip.String() + "/" + prefix

	return addr
}

func findPolicyInNft(rule *nftables.Rule) vsix.ZonePolicy {
	kind := rule.Exprs[0].(*expr.Verdict).Kind
	var policy vsix.ZonePolicy
	switch kind {
	case expr.VerdictDrop:
		policy = vsix.EstablishedOnly
	case expr.VerdictReturn:
		policy = vsix.AllPermit
	}
	return policy
}
