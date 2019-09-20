package stool

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"math/big"
	"sync"

	"github.com/aergoio/aergo-lib/db"
	"github.com/aergoio/aergo/types"
	"github.com/golang/protobuf/proto"
)

// Hash type of snapshotNodes map keys
type Hash [32]byte

var (
	// DefaultLeaf is the root of an empty branch
	DefaultLeaf = []byte{0}
)

// Hasher is in aergo/internal so cannot be imported at this time
var Hasher = func(data ...[]byte) []byte {
	hasher := sha256.New()
	for i := 0; i < len(data); i++ {
		hasher.Write(data[i])
	}
	return hasher.Sum(nil)
}

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
	// hash nodes to perform integrity check on state (analyses general trie and contract tries)
	integrityCheck bool
	// set accountKey to snapshot a specific account (voting contract)
	// and the key path nodes in general trie.
	accountKey []byte
}

// Counters groups counters together
type Counters struct {
	// -------------- General trie counters only -----------------------------
	// Number of pubkey accounts
	NbUserAccounts uint
	// Number of pubkey accounts with 0 balance
	NbUserAccounts0 uint
	// Number of contract accounts
	NbContracts uint
	// Number of nil objects
	// transaction with amount 0 to a new address creates a balance 0 and nonce 0 account
	NbNilObjects uint
	// Total Aer balace held by accounts (pubkey and contract)
	TotalAerBalance *big.Int

	// -------------- General trie and Contract trie counters ----------------
	// cumulated height (used for calulating avg depth)
	CumulatedHeight int
	AverageDepth    float64
	DeepestLeaf     int

	// -------------- Contract trie counters only ----------------------------
	// Number of storage values (leaves) in the contract trie
	NbStorageValues uint
}

// NewStateAnalysis initialises StateAnalysis
func NewStateAnalysis(store db.DB, countDbReads, generalTrie, integrityCheck bool, maxThread uint) *StateAnalysis {
	c := &Counters{
		NbUserAccounts:  0,
		NbUserAccounts0: 0,
		NbContracts:     0,
		NbNilObjects:    0,
		NbStorageValues: 0,
		CumulatedHeight: 0,
		AverageDepth:    0,
		DeepestLeaf:     256,
		TotalAerBalance: new(big.Int).SetUint64(0),
	}
	return &StateAnalysis{
		Counters:       c,
		maxThread:      maxThread,
		totalThread:    0,
		snapshotNodes:  make(map[Hash][]byte),
		snapshot:       false,
		generalTrie:    generalTrie,
		store:          store,
		countDbReads:   countDbReads,
		integrityCheck: integrityCheck,
	}
}

// Snapshot uses Dfs to copy nodes to a new snapshot db
func (sa *StateAnalysis) Snapshot(snapStore db.DB, root []byte) error {
	sa.snapStore = snapStore
	sa.snapshot = true
	sa.accountKey = nil
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

// SnapshotAccount uses Dfs to copy account state nodes and key path to a new snapshot db
func (sa *StateAnalysis) SnapshotAccount(snapStore db.DB, root, trieKey []byte) error {
	sa.snapStore = snapStore
	sa.snapshot = true
	sa.accountKey = trieKey
	err := sa.Dfs(root)
	if err != nil {
		return err
	}
	sa.commitSnapshotNodes(sa.snapshotNodes)
	sa.commitSnapshotNodes(sa.Trie.snapshotNodes)
	return nil
}

// Dfs Depth first search all the trie leaves starting from root
// For each leaf count it and add it's balance to the total
func (sa *StateAnalysis) Dfs(root []byte) error {
	sa.Trie = NewTrieReader(sa.store, sa.countDbReads, sa.snapshot)
	ch := make(chan error, 1)
	sa.dfs(root, 0, 256, nil, ch)
	err := <-ch
	sa.Counters.DeepestLeaf = 256 - sa.Counters.DeepestLeaf
	if sa.generalTrie {
		totalLeaves := float64(sa.Counters.NbUserAccounts + sa.Counters.NbUserAccounts0 + sa.Counters.NbContracts + sa.Counters.NbNilObjects)
		sa.Counters.AverageDepth = 256.0 - (float64(sa.Counters.CumulatedHeight) / totalLeaves)
	} else {
		sa.Counters.AverageDepth = 256.0 - (float64(sa.Counters.CumulatedHeight) / float64(sa.Counters.NbStorageValues))
	}
	return err
}

func (sa *StateAnalysis) dfs(root []byte, iBatch, height int, batch [][]byte, ch chan<- (error)) {
	batch, iBatch, lnode, rnode, isShortcut, err := sa.Trie.LoadChildren(root, height, iBatch, batch)
	if err != nil {
		ch <- err
		return
	}
	if isShortcut {
		if sa.integrityCheck {
			if !bytes.Equal(root[:HashLength], Hasher(lnode[:HashLength], rnode[:HashLength], []byte{byte(height)})) {
				ch <- fmt.Errorf("Warning: state integrity failed")
				return
			}
		}
		sa.counterLock.Lock()
		sa.Counters.CumulatedHeight += height
		if sa.Counters.DeepestLeaf > height {
			sa.Counters.DeepestLeaf = height
		}
		sa.counterLock.Unlock()
		raw := sa.Trie.db.Get(rnode[:HashLength])
		if sa.generalTrie {
			// always parse account in general trie
			storageRoot, codeHash, err := sa.parseAccount(raw)
			if err != nil {
				ch <- err
				return
			}
			if sa.snapshot {
				// snapshot always requires copying contract state
				if sa.accountKey != nil && !bytes.Equal(sa.accountKey, lnode[:HashLength]) {
					// if snapshot of a single account (aergo.system) then it should match leaf
					ch <- fmt.Errorf("lnode doesnt match requested account key snapshot")
					return
				}
				if storageRoot != nil {
					// snapshot contract storage nodes
					err := sa.snapshotContractState(storageRoot)
					if err != nil {
						ch <- err
						return
					}
				}
				if codeHash != nil {
					code := sa.Trie.db.Get(codeHash)
					var dbkey Hash
					copy(dbkey[:], codeHash)
					sa.snapshotLock.Lock()
					sa.snapshotNodes[dbkey] = code
					sa.snapshotLock.Unlock()
				}
			} else if sa.integrityCheck && storageRoot != nil {
				// contracts only need to be analysed when doing integrity check
				err := sa.analyseContractState(storageRoot)
				if err != nil {
					ch <- err
					return
				}
			} else {
				// do nothing, only analysing the General trie
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
	} else if sa.integrityCheck {
		// if not leaf node and check integrity, then hash nodes to perform check
		var h []byte
		// lnode and rnode cannot be default at the same time
		if len(lnode) == 0 {
			h = Hasher(DefaultLeaf, rnode[:HashLength])
		} else if len(rnode) == 0 {
			h = Hasher(lnode[:HashLength], DefaultLeaf)
		} else {
			h = Hasher(lnode[:HashLength], rnode[:HashLength])
		}
		if !bytes.Equal(root[:HashLength], h) {
			fmt.Println(root, lnode, rnode)
			ch <- fmt.Errorf("Warning: state integrity failed")
			return
		}
	}
	if sa.generalTrie && sa.accountKey != nil {
		// snapshot single account path in general trie
		if bitIsSet(sa.accountKey, 256-height) {
			if rnode == nil {
				ch <- fmt.Errorf("nil node in the path: account not in general trie")
				return
			}
			err := sa.stepRight(rnode, iBatch, height, batch)
			ch <- err
		} else {
			if lnode == nil {
				ch <- fmt.Errorf("nil node in the path: account not in general trie")
				return
			}
			err := sa.stepLeft(lnode, iBatch, height, batch)
			ch <- err
		}
	} else {
		err = sa.stepRightLeft(lnode, rnode, iBatch, height, batch)
		ch <- err
	}
}

func (sa *StateAnalysis) stepRightLeft(lnode, rnode []byte, iBatch, height int, batch [][]byte) error {
	if lnode != nil && rnode != nil {
		lch := make(chan error, 1)
		rch := make(chan error, 1)
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
		return sa.stepLeft(lnode, iBatch, height, batch)
	} else if rnode != nil {
		return sa.stepRight(rnode, iBatch, height, batch)
	}
	return nil
}

func (sa *StateAnalysis) stepRight(rnode []byte, iBatch, height int, batch [][]byte) error {
	rch := make(chan error, 1)
	if rnode != nil {
		sa.dfs(rnode, 2*iBatch+2, height-1, batch, rch)
		rresult := <-rch
		if rresult != nil {
			return rresult
		}
	}
	return nil
}

func (sa *StateAnalysis) stepLeft(lnode []byte, iBatch, height int, batch [][]byte) error {
	lch := make(chan error, 1)
	if lnode != nil {
		sa.dfs(lnode, 2*iBatch+1, height-1, batch, lch)
		lresult := <-lch
		if lresult != nil {
			return lresult
		}
	}
	return nil
}

func (sa *StateAnalysis) parseAccount(raw []byte) ([]byte, []byte, error) {
	if len(raw) == 0 {
		// transaction with amount 0 to a new address creates a balance 0 and nonce 0 account
		sa.counterLock.Lock()
		sa.Counters.NbNilObjects++
		sa.counterLock.Unlock()
		return nil, nil, nil
	}
	data := &types.State{}
	err := proto.Unmarshal(raw, data)
	if err != nil {
		return nil, nil, err
	}
	sa.counterLock.Lock()
	storageRoot := data.GetStorageRoot()
	codeHash := data.GetCodeHash()
	if codeHash != nil {
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
	return storageRoot, codeHash, nil
}

func (sa *StateAnalysis) snapshotContractState(storageRoot []byte) error {
	// TODO count db reads of contracts
	storageAnalysis := NewStateAnalysis(sa.store, false, false, false, 1000)
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

func (sa *StateAnalysis) analyseContractState(storageRoot []byte) error {
	// TODO count db reads of contracts
	storageAnalysis := NewStateAnalysis(sa.store, false, false, sa.integrityCheck, 1000)
	storageAnalysis.snapshot = false
	err := storageAnalysis.Dfs(storageRoot)
	if err != nil {
		return err
	}
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
