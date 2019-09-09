package stool

import (
	"math/big"
	"sync"

	"github.com/aergoio/aergo-lib/db"
	"github.com/aergoio/aergo/types"
	"github.com/golang/protobuf/proto"
)

// Hash type of snapshotNodes map keys
type Hash [32]byte

// DbTx represents Set and Delete interface to store data
type DbTx interface {
	Set(key, value []byte)
	Delete(key []byte)
}

// StateAnalysis stores the results of dfs
type StateAnalysis struct {
	// counterLock for Counters writing
	counterLock sync.RWMutex
	// Counters keeps track of the nb of different types of accounts
	Counters *Counters
	// Trie contains trie reading functionality
	Trie *TrieReader
	// if true copies a snapshot of nodes to a snapshot db
	snapshot bool
	// max Threads created while parsing trie
	maxThread uint
	// current total nb of threads created
	totalThread uint
	// cache shortcut nodes before writing them to snapshot db
	snapshotNodes map[Hash][]byte
	// snapshotLock for snapshot nodes caching
	snapshotLock sync.RWMutex
	// snapStoreLock for snapshotNodes writing to snapStore
	snapStoreLock sync.RWMutex
	// differenciate a general trie analysis from a storage trie analysis
	generalTrie bool
	// database to read from
	store db.DB
	// database to write snapshot
	snapStore db.DB
	// countDbReads
	countDbReads bool
}

// Counters groups counters together
type Counters struct {
	NbUserAccounts  uint
	NbUserAccounts0 uint
	NbContracts     uint
	NbNilObjects    uint
	NbStorageValues uint
	TotalAerBalance *big.Int
}

// NewStateAnalysis initialises StateAnalysis
func NewStateAnalysis(store db.DB, countDbReads, generalTrie bool, maxThread uint) *StateAnalysis {
	c := &Counters{
		NbUserAccounts:  0,
		NbUserAccounts0: 0,
		NbContracts:     0,
		NbNilObjects:    0,
		NbStorageValues: 0,
		TotalAerBalance: new(big.Int).SetUint64(0),
	}
	return &StateAnalysis{
		Counters:      c,
		maxThread:     maxThread,
		totalThread:   0,
		snapshotNodes: make(map[Hash][]byte),
		snapshot:      false,
		generalTrie:   generalTrie,
		store:         store,
		countDbReads:  countDbReads,
	}
}

// Snapshot uses Dfs to copy nodes to a new snapshot db
func (sa *StateAnalysis) Snapshot(snapStore db.DB, root []byte) error {
	sa.snapStore = snapStore
	sa.snapshot = true
	err := sa.Dfs(root)
	if err != nil {
		return err
	}
	sa.commitSnapshotNodes(sa.snapshotNodes)
	sa.commitSnapshotNodes(sa.Trie.snapshotNodes)
	return nil
}

// Analyse uses Dfs to analyse and count trie nodes
func (sa *StateAnalysis) Analyse(root []byte) error {
	sa.snapshot = false
	return sa.Dfs(root)
}

// Dfs Depth first search all the trie leaves starting from root
// For each leaf count it and add it's balance to the total
func (sa *StateAnalysis) Dfs(root []byte) error {
	sa.Trie = NewTrieReader(sa.store, sa.countDbReads, sa.snapshot)
	ch := make(chan error, 1)
	sa.dfs(root, 0, 256, nil, ch)
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
		if sa.generalTrie {
			// always parse account in general trie
			storageRoot, err := sa.parseAccount(raw)
			if err != nil {
				ch <- err
				return
			}
			// if snapshot and is contract account, parse contract storage
			if sa.snapshot && storageRoot != nil {
				// snapshot contract storage nodes
				err := sa.snapshotContractState(storageRoot)
				if err != nil {
					ch <- err
					return
				}
			}
		} else {
			// storage values cannot be parsed so just count them
			sa.counterLock.Lock()
			sa.Counters.NbStorageValues++
			sa.counterLock.Unlock()
		}
		if sa.snapshot {
			// snapshot shortcut node
			// fmt.Println("height: ", height)
			var dbkey Hash
			copy(dbkey[:], rnode[:HashLength])
			sa.snapshotLock.Lock()
			sa.snapshotNodes[dbkey] = raw
			sa.snapshotLock.Unlock()
		}
		ch <- nil
		return
	}
	err = sa.stepRightLeft(lnode, rnode, iBatch, height, batch)
	ch <- err
}

func (sa *StateAnalysis) stepRightLeft(lnode, rnode []byte, iBatch, height int, batch [][]byte) error {
	lch := make(chan error, 1)
	rch := make(chan error, 1)
	if lnode != nil && rnode != nil {
		if sa.totalThread < sa.maxThread {
			go sa.dfs(lnode, 2*iBatch+1, height-1, batch, lch)
			go sa.dfs(rnode, 2*iBatch+2, height-1, batch, rch)
			sa.counterLock.Lock()
			sa.totalThread += 2
			sa.counterLock.Unlock()
		} else {
			sa.dfs(lnode, 2*iBatch+1, height-1, batch, lch)
			sa.dfs(rnode, 2*iBatch+2, height-1, batch, rch)
		}
		lresult := <-lch
		if lresult != nil {
			return lresult
		}
		rresult := <-rch
		if rresult != nil {
			return rresult
		}
	} else if lnode != nil {
		sa.dfs(lnode, 2*iBatch+1, height-1, batch, lch)
		lresult := <-lch
		if lresult != nil {
			return lresult
		}
	} else if rnode != nil {
		sa.dfs(rnode, 2*iBatch+2, height-1, batch, rch)
		rresult := <-rch
		if rresult != nil {
			return rresult
		}
	}
	return nil
}

func (sa *StateAnalysis) parseAccount(raw []byte) ([]byte, error) {
	if len(raw) == 0 {
		// transaction with amount 0 to a new address creates a balance 0 and nonce 0 account
		sa.counterLock.Lock()
		sa.Counters.NbNilObjects++
		sa.counterLock.Unlock()
		return nil, nil
	}
	data := &types.State{}
	err := proto.Unmarshal(raw, data)
	if err != nil {
		return nil, err
	}
	sa.counterLock.Lock()
	storageRoot := data.GetStorageRoot()
	if storageRoot != nil {
		sa.Counters.NbContracts++
	} else if data.GetBalance() != nil {
		sa.Counters.NbUserAccounts++
	} else {
		// User account with 0 balance
		sa.Counters.NbUserAccounts0++
	}
	sa.Counters.TotalAerBalance = new(big.Int).Add(sa.Counters.TotalAerBalance,
		new(big.Int).SetBytes(data.GetBalance()))
	sa.counterLock.Unlock()
	return storageRoot, nil
}

func (sa *StateAnalysis) snapshotContractState(storageRoot []byte) error {
	storageAnalysis := NewStateAnalysis(sa.store, sa.countDbReads, false, 100)
	storageAnalysis.snapStore = sa.snapStore
	storageAnalysis.snapshot = true
	err := storageAnalysis.Dfs(storageRoot)
	if err != nil {
		return err
	}
	sa.commitSnapshotNodes(storageAnalysis.snapshotNodes)
	sa.commitSnapshotNodes(storageAnalysis.Trie.snapshotNodes)
	return nil
}

func (sa *StateAnalysis) commitSnapshotNodes(snapshotNodes map[Hash][]byte) {
	sa.snapStoreLock.Lock()
	txn := sa.snapStore.NewTx().(DbTx)
	for key, value := range snapshotNodes {
		var node []byte
		txn.Set(append(node, key[:]...), value)
	}
	txn.(db.Transaction).Commit()
	sa.snapStoreLock.Unlock()
}
