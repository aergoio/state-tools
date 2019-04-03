package analysis

import (
	"fmt"
	"sync"

	"github.com/aergoio/aergo-lib/db"
)

const (
	// HashLength is the number of bytes in a hash
	HashLength = 32
)

// TrieReader provides tools for parsing trie nodes in a db
type TrieReader struct {
	db db.DB
	// dbLock is a lock for db access
	dbLock sync.RWMutex
	// TrieHeight is the number if bits in a key
	TrieHeight int
	// LoadDbCounter counts the nb of db reads in on update
	LoadDbCounter int
	// loadDbMux is a lock for LoadDbCounter
	loadDbMux sync.RWMutex
	// counterOn is used to enable/diseable for efficiency
	counterOn bool
	// CacheHeightLimit is the number of tree levels we want to store in cache
}

// NewTrieReader creates a new TrieReader
func NewTrieReader(store db.DB) *TrieReader {
	s := &TrieReader{
		TrieHeight:    256, // hash any string to get output length
		counterOn:     false,
		db:            store,
		LoadDbCounter: 0,
	}
	return s
}

// LoadChildren looks for the children of a node.
// if the node is not stored in cache, it will be loaded from db.
func (s *TrieReader) LoadChildren(root []byte, height, iBatch int, batch [][]byte) ([][]byte, int, []byte, []byte, bool, error) {
	isShortcut := false
	if height%4 == 0 {
		if len(root) == 0 {
			// create a new default batch
			batch = make([][]byte, 31, 31)
			batch[0] = []byte{0}
		} else {
			var err error
			batch, err = s.loadBatch(root)
			if err != nil {
				return nil, 0, nil, nil, false, err
			}
		}
		iBatch = 0
		if batch[0][0] == 1 {
			isShortcut = true
		}
	} else {
		if len(batch[iBatch]) != 0 && batch[iBatch][HashLength] == 1 {
			isShortcut = true
		}
	}
	return batch, iBatch, batch[2*iBatch+1], batch[2*iBatch+2], isShortcut, nil
}

// loadBatch fetches a batch of nodes in cache or db
func (s *TrieReader) loadBatch(root []byte) ([][]byte, error) {
	//Fetch node in disk database
	if s.db == nil {
		return nil, fmt.Errorf("DB not connected to trie")
	}
	if s.counterOn {
		s.loadDbMux.Lock()
		s.LoadDbCounter++
		s.loadDbMux.Unlock()
	}
	// s.dbLock.Lock()
	dbval := s.db.Get(root[:HashLength])
	// s.dbLock.Unlock()
	nodeSize := len(dbval)
	if nodeSize != 0 {
		return s.parseBatch(dbval), nil
	}
	return nil, fmt.Errorf("the trie node %x is unavailable in the disk db, db may be corrupted", root)
}

// parseBatch decodes the byte data into a slice of nodes and bitmap
func (s *TrieReader) parseBatch(val []byte) [][]byte {
	batch := make([][]byte, 31, 31)
	bitmap := val[:4]
	// check if the batch root is a shortcut
	if bitIsSet(val, 31) {
		batch[0] = []byte{1}
		batch[1] = val[4 : 4+33]
		batch[2] = val[4+33 : 4+33*2]
	} else {
		batch[0] = []byte{0}
		j := 0
		for i := 1; i <= 30; i++ {
			if bitIsSet(bitmap, i-1) {
				batch[i] = val[4+33*j : 4+33*(j+1)]
				j++
			}
		}
	}
	return batch
}

func bitIsSet(bits []byte, i int) bool {
	return bits[i/8]&(1<<uint(7-i%8)) != 0
}

func bitSet(bits []byte, i int) {
	bits[i/8] |= 1 << uint(7-i%8)
}
