package nft

import (
	"fmt"

	"github.com/google/nftables"
	"github.com/google/nftables/expr"
)

// 呼び出し先で，よしなにTable, Chainを変えればRuleの追加先を変更できる
func InitRule(nsname string) error {
	conn, fd, err := InitConn(nsname)
	if err != nil {
		return err
	}
	defer CloseConn(fd)

	if err := InitChain(conn); err != nil {
		return err
	}

	for _, chain := range ChainList {
		switch chain {
		case InputChain:
			conn.AddRule(RateLimitICMPv6)
			if err := conn.Flush(); err != nil {
				msg := fmt.Errorf("Failed to flush rule to kernel: %v", err)
				return msg
			}
			conn.AddRule(RateLimitICMP)
			if err := conn.Flush(); err != nil {
				msg := fmt.Errorf("Failed to flush rule to kernel: %v", err)
				return msg
			}
			rule, _ := CreateCtstateRule(chain, 6, expr.VerdictAccept)
			conn.AddRule(rule)
			if err := conn.Flush(); err != nil {
				msg := fmt.Errorf("Failed to flush rule to kernel: %v", err)
				return msg
			}
		case PairTrustToTrustChain, PairUntrustToUntrustChain:
			rule, _ := CreateDefaultPolicyRule(chain, expr.VerdictReturn)
			conn.AddRule(rule)
			if err := conn.Flush(); err != nil {
				msg := fmt.Errorf("Failed to flush rule to kernel: %v", err)
				return msg
			}
		case PairUntustToTrustChain:
			ruleset, _ := CreateEstablishedOnlyRuleset(chain)
			for _, rule := range ruleset {
				conn.AddRule(rule)
				if err := conn.Flush(); err != nil {
					msg := fmt.Errorf("Failed to flush rule to kernel: %v", err)
					return msg
				}
			}
		case PairTrustToUntrustChain:
			ruleset, _ := CreateAllPermitRuleset(chain)
			for _, rule := range ruleset {
				conn.AddRule(rule)
				if err := conn.Flush(); err != nil {
					msg := fmt.Errorf("Failed to flush rule to kernel: %v", err)
					return msg
				}
			}
		}
	}
	return nil
}

// ip6 saddr [IPv6 Address] [PolicyName]系のrule
// Mask, Data, Verdict.Kindをよしなに変更する
var PrefixIPv6InputRule = &nftables.Rule{
	Table: VsixTable,
	Chain: InputChain,
	Exprs: []expr.Any{
		&expr.Meta{
			Key:            0x0000000f,
			SourceRegister: false,
			Register:       0x00000001,
		},
		&expr.Cmp{
			Op:       0x00000000,
			Register: 0x00000001,
			Data: []uint8{
				0x0a,
			},
		},
		&expr.Payload{
			OperationType:  0x00000000,
			DestRegister:   0x00000001,
			SourceRegister: 0x00000000,
			Base:           0x00000001,
			Offset:         0x00000008,
			Len:            0x00000010,
			CsumType:       0x00000000,
			CsumOffset:     0x00000000,
			CsumFlags:      0x00000000,
		},
		&expr.Bitwise{
			SourceRegister: 0x00000001,
			DestRegister:   0x00000001,
			Len:            0x00000010,
			Mask:           []uint8{},
			Xor: []uint8{
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			},
		},
		&expr.Cmp{
			Op:       0x00000000,
			Register: 0x00000001,
			Data:     []uint8{},
		},
		&expr.Verdict{
			Kind:  expr.VerdictAccept,
			Chain: "",
		},
	},
}

// ip saddr [IPv4 Address] [PolicyName]系のrule
// Mask, Data, Verdict.Kindをよしなに変更する
var PrefixIPv4InputRule = &nftables.Rule{
	Table: VsixTable,
	Chain: InputChain,
	Exprs: []expr.Any{
		&expr.Meta{
			Key:            0x0000000f,
			SourceRegister: false,
			Register:       0x00000001,
		},
		&expr.Cmp{
			Op:       0x00000000,
			Register: 0x00000001,
			Data: []uint8{
				0x02,
			},
		},
		&expr.Payload{
			OperationType:  0x00000000,
			DestRegister:   0x00000001,
			SourceRegister: 0x00000000,
			Base:           0x00000001,
			Offset:         0x0000000c,
			Len:            0x00000004,
			CsumType:       0x00000000,
			CsumOffset:     0x00000000,
			CsumFlags:      0x00000000,
		},
		&expr.Bitwise{
			SourceRegister: 0x00000001,
			DestRegister:   0x00000001,
			Len:            0x00000004,
			Mask:           []uint8{},
			Xor: []uint8{
				0x00, 0x00, 0x00, 0x00,
			},
		},
		&expr.Cmp{
			Op:       0x00000000,
			Register: 0x00000001,
			Data:     []uint8{},
		},
		&expr.Verdict{
			Kind:  expr.VerdictAccept,
			Chain: "",
		},
	},
}

// ct state [CtState Name] [Policy Name]系のrule
// Bitwise.Maskをよしなに変更する
// invalid:1, established:2, related:4, new:8, untracked:64
var CtStateRule = &nftables.Rule{
	Table: VsixTable,
	Chain: InputChain,
	Exprs: []expr.Any{
		&expr.Ct{
			Register:       0x00000001,
			SourceRegister: false,
			Key:            0x00000000,
		},
		&expr.Bitwise{
			SourceRegister: 0x00000001,
			DestRegister:   0x00000001,
			Len:            0x00000004,
			Mask: []uint8{
				0x00, 0x00, 0x00, 0x00,
			},
			Xor: []uint8{
				0x00, 0x00, 0x00, 0x00,
			},
		},
		&expr.Cmp{
			Op:       0x00000001,
			Register: 0x00000001,
			Data: []uint8{
				0x00, 0x00, 0x00, 0x00,
			},
		},
		&expr.Verdict{
			Kind:  expr.VerdictReturn,
			Chain: "",
		},
	},
}

// oifname [IF_NAME] jump [Jump先のChain Name]系の命令
// Data, Verdict.Chainをよしなに変更する
var OifJumpRule = &nftables.Rule{
	Table: VsixTable,
	Chain: ForwardChain,
	Exprs: []expr.Any{
		&expr.Meta{
			Key:            expr.MetaKeyOIFNAME,
			SourceRegister: false,
			Register:       0x00000001,
		},
		&expr.Cmp{
			Op:       0x00000000,
			Register: 0x00000001,
			Data:     []uint8{},
		},
		&expr.Verdict{
			Kind:  expr.VerdictJump,
			Chain: "",
		},
	},
}

// iifname [IF_NAME] jump [Jump先のChain Name]系の命令
// Data, Verdict.Chainをよしなに変更する
var IifJumpRule = &nftables.Rule{
	Table: VsixTable,
	Chain: ZoneTrustChain,
	Exprs: []expr.Any{
		&expr.Meta{
			Key:            expr.MetaKeyIIFNAME,
			SourceRegister: false,
			Register:       0x00000001,
		},
		&expr.Cmp{
			Op:       0x00000000,
			Register: 0x00000001,
			Data:     []uint8{},
		},
		&expr.Verdict{
			Kind:  expr.VerdictJump,
			Chain: "",
		},
	},
}

// accept, drop系の命令
// Verdict.Kindをよしなに変更
var DefaultPolicyRule = &nftables.Rule{
	Table: VsixTable,
	Chain: PairTrustToTrustChain,
	Exprs: []expr.Any{
		&expr.Verdict{
			Kind:  expr.VerdictReturn,
			Chain: "",
		},
	},
}

// ip protocol [protocol name] [policy name]系の命令
// 二番目のCmp.Data, Verdict.Kindをよしなに変更
var ProtocolRule = &nftables.Rule{
	Table: VsixTable,
	Chain: PairUntustToTrustChain,
	Exprs: []expr.Any{
		&expr.Meta{
			Key:            0x0000000f,
			SourceRegister: false,
			Register:       0x00000001,
		},
		&expr.Cmp{
			Op:       0x00000000,
			Register: 0x00000001,
			Data: []uint8{
				0x02,
			},
		},
		&expr.Payload{
			OperationType:  0x00000000,
			DestRegister:   0x00000001,
			SourceRegister: 0x00000000,
			Base:           0x00000001,
			Offset:         0x00000009,
			Len:            0x00000001,
			CsumType:       0x00000000,
			CsumOffset:     0x00000000,
			CsumFlags:      0x00000000,
		},
		&expr.Cmp{
			Op:       0x00000000,
			Register: 0x00000001,
			Data:     []uint8{},
		},
		&expr.Verdict{
			Kind:  expr.VerdictReturn,
			Chain: "",
		},
	},
}

// ip6 nextdhr [protocol name] [policy name] 系の命令
// 二番目のCmp.Data, Verdict.Kindをよしなに変更
var NexthdrRule = &nftables.Rule{
	Table: VsixTable,
	Chain: PairUntustToTrustChain,
	Exprs: []expr.Any{
		&expr.Meta{
			Key:            0x0000000f,
			SourceRegister: false,
			Register:       0x00000001,
		},
		&expr.Cmp{
			Op:       0x00000000,
			Register: 0x00000001,
			Data: []uint8{
				0x0a,
			},
		},
		&expr.Payload{
			OperationType:  0x00000000,
			DestRegister:   0x00000001,
			SourceRegister: 0x00000000,
			Base:           0x00000001,
			Offset:         0x00000006,
			Len:            0x00000001,
			CsumType:       0x00000000,
			CsumOffset:     0x00000000,
			CsumFlags:      0x00000000,
		},
		&expr.Cmp{
			Op:       0x00000000,
			Register: 0x00000001,
			Data:     []uint8{},
		},
		&expr.Verdict{
			Kind:  expr.VerdictReturn,
			Chain: "",
		},
	},
}

// ip protocol icmp limit rate [rate] accept 系の命令
// Limit.Rateをよしなに変更
var RateLimitICMP = &nftables.Rule{
	Table: VsixTable,
	Chain: InputChain,
	Exprs: []expr.Any{
		&expr.Meta{
			Key:            0x0000000f,
			SourceRegister: false,
			Register:       0x00000001,
		},
		&expr.Cmp{
			Op:       0x00000000,
			Register: 0x00000001,
			Data: []uint8{
				0x02,
			},
		},
		&expr.Payload{
			OperationType:  0x00000000,
			DestRegister:   0x00000001,
			SourceRegister: 0x00000000,
			Base:           0x00000001,
			Offset:         0x00000009,
			Len:            0x00000001,
			CsumType:       0x00000000,
			CsumOffset:     0x00000000,
			CsumFlags:      0x00000000,
		},
		&expr.Cmp{
			Op:       0x00000000,
			Register: 0x00000001,
			Data: []uint8{
				0x01,
			},
		},
		&expr.Limit{
			Type:  0x00000000,
			Rate:  0x000000000000000a,
			Over:  false,
			Unit:  0x0000000000000001,
			Burst: 0x00000005,
		},
		&expr.Verdict{
			Kind:  1,
			Chain: "",
		},
	},
}

// ip6 nexthdr icmp6 limit rate [rate] accept 系の命令
// Limit.Rateをよしなに変更
var RateLimitICMPv6 = &nftables.Rule{
	Table: VsixTable,
	Chain: InputChain,
	Exprs: []expr.Any{
		&expr.Meta{
			Key:            0x0000000f,
			SourceRegister: false,
			Register:       0x00000001,
		},
		&expr.Cmp{
			Op:       0x00000000,
			Register: 0x00000001,
			Data: []uint8{
				0x0a,
			},
		},
		&expr.Payload{
			OperationType:  0x00000000,
			DestRegister:   0x00000001,
			SourceRegister: 0x00000000,
			Base:           0x00000001,
			Offset:         0x00000006,
			Len:            0x00000001,
			CsumType:       0x00000000,
			CsumOffset:     0x00000000,
			CsumFlags:      0x00000000,
		},
		&expr.Cmp{
			Op:       0x00000000,
			Register: 0x00000001,
			Data: []uint8{
				0x3a,
			},
		},
		&expr.Limit{
			Type:  0x00000000,
			Rate:  0x000000000000000a,
			Over:  false,
			Unit:  0x0000000000000001,
			Burst: 0x00000005,
		},
		&expr.Verdict{
			Kind:  expr.VerdictAccept,
			Chain: "",
		},
	},
}

var EstablishedOnlyRuleset = []*nftables.Rule{
	NexthdrRule,
	ProtocolRule,
	CtStateRule,
	DefaultPolicyRule,
}

var AllPermitRuleset = []*nftables.Rule{
	DefaultPolicyRule,
}
