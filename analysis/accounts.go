package analysis

import (
	"github.com/aergoio/aergo-lib/db"
	sha256 "github.com/minio/sha256-simd"
)

// AccountsAnalysis stores the results of dfs
type AccountsAnalysis struct {
	NbUserAccounts  uint
	NbContracts     uint
	TotalAerBalance []byte
	Trie            *TrieReader
}

// NewAccountsAnalysis initialises AccountsAnalysis
func NewAccountsAnalysis(store db.DB) *AccountsAnalysis {
	return &AccountsAnalysis{
		NbUserAccounts: 0,
		NbContracts:    0,
		//TotalAerBalance: 0,
		Trie: NewTrieReader(store),
	}
}

// Dfs Depth first search all the trie leaves starting from root
// For each leaf count it and add it's balance to the total
func (aa *AccountsAnalysis) Dfs(root []byte, iBatch, height int, batch [][]byte) error {
	batch, iBatch, lnode, rnode, isShortcut, err := aa.Trie.LoadChildren(root, height, iBatch, batch)
	if err != nil {
		return err
	}
	if isShortcut {
		aa.NbUserAccounts++
		// TODO load account in db and get balance
		return nil
	}

	if lnode != nil {
		err = aa.Dfs(lnode, 2*iBatch+1, height-1, batch)
		if err != nil {
			return err
		}
	}
	if rnode != nil {
		err = aa.Dfs(rnode, 2*iBatch+2, height-1, batch)
		if err != nil {
			return err
		}
	}
	return nil
}

// Hasher is in aergo/internal so cannot be imported at this time
var Hasher = func(data ...[]byte) []byte {
	hasher := sha256.New()
	for i := 0; i < len(data); i++ {
		hasher.Write(data[i])
	}
	return hasher.Sum(nil)
}
