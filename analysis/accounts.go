package analysis

import (
	"math/big"
	"sync"

	"github.com/aergoio/aergo-lib/db"
	"github.com/aergoio/aergo/types"
	"github.com/golang/protobuf/proto"
)

// AccountsAnalysis stores the results of dfs
type AccountsAnalysis struct {
	lock            sync.RWMutex
	NbUserAccounts  uint
	NbUserAccounts0 uint
	NbContracts     uint
	NbNilObjects    uint
	TotalAerBalance *big.Int
	Trie            *TrieReader
	Snapshot        bool
	maxThread       uint
	totalThread     uint
}

// NewAccountsAnalysis initialises AccountsAnalysis
func NewAccountsAnalysis(store db.DB, countDbReads, snapshot bool) *AccountsAnalysis {
	return &AccountsAnalysis{
		NbUserAccounts:  0,
		NbUserAccounts0: 0,
		NbContracts:     0,
		NbNilObjects:    0,
		TotalAerBalance: new(big.Int),
		Trie:            NewTrieReader(store, countDbReads),
		Snapshot:        snapshot,
		maxThread:       10000,
		totalThread:     0,
	}
}

// Dfs Depth first search all the trie leaves starting from root
// For each leaf count it and add it's balance to the total
func (aa *AccountsAnalysis) Dfs(root []byte, iBatch, height int, batch [][]byte) error {
	ch := make(chan error, 1)
	aa.dfs(root, iBatch, height, batch, ch)
	err := <-ch
	return err
}

func (aa *AccountsAnalysis) dfs(root []byte, iBatch, height int, batch [][]byte, ch chan<- (error)) {
	batch, iBatch, lnode, rnode, isShortcut, err := aa.Trie.LoadChildren(root, height, iBatch, batch)
	if err != nil {
		ch <- err
		return
	}
	if isShortcut {
		raw := aa.Trie.db.Get(rnode[:HashLength])
		if len(raw) == 0 {
			// transaction with amount 0 to a new address creates a balance 0 and nonce 0 account
			aa.lock.Lock()
			aa.NbNilObjects++
			aa.lock.Unlock()
			ch <- nil
			return
		}
		data := &types.State{}
		err = proto.Unmarshal(raw, data)
		if err != nil {
			ch <- err
			return
		}
		aa.lock.Lock()
		if data.GetStorageRoot() != nil {
			aa.NbContracts++
		} else if data.GetBalance() != nil {
			aa.NbUserAccounts++
		} else {
			// User account with 0 balance
			aa.NbUserAccounts0++
		}
		aa.TotalAerBalance = new(big.Int).Add(aa.TotalAerBalance,
			new(big.Int).SetBytes(data.GetBalance()))
		aa.lock.Unlock()
		ch <- nil
		return
	}

	lch := make(chan error, 1)
	rch := make(chan error, 1)
	if lnode != nil && rnode != nil {
		if aa.totalThread < aa.maxThread {
			go aa.dfs(lnode, 2*iBatch+1, height-1, batch, lch)
			go aa.dfs(rnode, 2*iBatch+2, height-1, batch, rch)
			aa.lock.Lock()
			aa.totalThread += 2
			aa.lock.Unlock()
		} else {
			aa.dfs(lnode, 2*iBatch+1, height-1, batch, lch)
			aa.dfs(rnode, 2*iBatch+2, height-1, batch, rch)
		}
		lresult := <-lch
		if lresult != nil {
			ch <- lresult
			return
		}
		rresult := <-rch
		if rresult != nil {
			ch <- rresult
			return
		}
	} else if lnode != nil {
		aa.dfs(lnode, 2*iBatch+1, height-1, batch, lch)
		lresult := <-lch
		if lresult != nil {
			ch <- lresult
			return
		}
	} else if rnode != nil {
		aa.dfs(rnode, 2*iBatch+2, height-1, batch, rch)
		rresult := <-rch
		if rresult != nil {
			ch <- rresult
			return
		}
	}
	ch <- nil
}
