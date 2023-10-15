package nft

import (
	"github.com/google/nftables"
)

func InitTable(conn *nftables.Conn) error {
	table := VsixTable

	conn.AddTable(table)
	conn.DelTable(table)
	if err := conn.Flush(); err != nil {
		return err
	}

	conn.AddTable(table)
	if err := conn.Flush(); err != nil {
		return err
	}
	return nil
}

var VsixTable = &nftables.Table{
	Family: nftables.TableFamilyINet,
	Name:   "filter",
}
