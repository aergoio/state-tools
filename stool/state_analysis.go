package stool

import (
	"math/big"
	"sync"

	"github.com/aergoio/aergo-lib/db"
	"github.com/aergoio/aergo/types"
	"github.com/golang/protobuf/proto"
)

// StateAnalysis stores the results of dfs
type StateAnalysis struct {
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

// NewStateAnalysis initialises StateAnalysis
func NewStateAnalysis(store db.DB, countDbReads, snapshot bool) *StateAnalysis {
	return &StateAnalysis{
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
func (sa *StateAnalysis) Dfs(root []byte, iBatch, height int, batch [][]byte) error {
	ch := make(chan error, 1)
	sa.dfs(root, iBatch, height, batch, ch)
	err := <-ch
	return err
}

func (sa *StateAnalysis) dfs(root []byte, iBatch, height int, batch [][]byte, ch chan<- (error)) {
	batch, iBatch, lnode, rnode, isShortcut, err := sa.Trie.LoadChildren(root, height, iBatch, batch)
	if err != nil {
		ch <- err
		return
	}
	if isShortcut {
		raw := sa.Trie.db.Get(rnode[:HashLength])
		// TODO copy to new db
		if len(raw) == 0 {
			// transaction with amount 0 to a new address creates a balance 0 and nonce 0 account
			sa.lock.Lock()
			sa.NbNilObjects++
			sa.lock.Unlock()
			ch <- nil
			return
		}
		data := &types.State{}
		err = proto.Unmarshal(raw, data)
		if err != nil {
			ch <- err
			return
		}
		sa.lock.Lock()
		storageRoot := data.GetStorageRoot()
		if storageRoot != nil {
			sa.NbContracts++
		} else if data.GetBalance() != nil {
			sa.NbUserAccounts++
		} else {
			// User account with 0 balance
			sa.NbUserAccounts0++
		}
		sa.TotalAerBalance = new(big.Int).Add(sa.TotalAerBalance,
			new(big.Int).SetBytes(data.GetBalance()))
		sa.lock.Unlock()
		ch <- nil
		return
	}

	lch := make(chan error, 1)
	rch := make(chan error, 1)
	if lnode != nil && rnode != nil {
		if sa.totalThread < sa.maxThread {
			go sa.dfs(lnode, 2*iBatch+1, height-1, batch, lch)
			go sa.dfs(rnode, 2*iBatch+2, height-1, batch, rch)
			sa.lock.Lock()
			sa.totalThread += 2
			sa.lock.Unlock()
		} else {
			sa.dfs(lnode, 2*iBatch+1, height-1, batch, lch)
			sa.dfs(rnode, 2*iBatch+2, height-1, batch, rch)
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
		sa.dfs(lnode, 2*iBatch+1, height-1, batch, lch)
		lresult := <-lch
		if lresult != nil {
			ch <- lresult
			return
		}
	} else if rnode != nil {
		sa.dfs(rnode, 2*iBatch+2, height-1, batch, rch)
		rresult := <-rch
		if rresult != nil {
			ch <- rresult
			return
		}
	}
	ch <- nil
}
