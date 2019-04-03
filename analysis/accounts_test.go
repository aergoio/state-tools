package analysis

import (
	"bytes"
	"fmt"
	"math"
	"math/rand"
	"os"
	"path"
	"sort"
	"testing"
	"time"

	"github.com/aergoio/aergo-lib/db"
	"github.com/aergoio/aergo/pkg/trie"
	sha256 "github.com/minio/sha256-simd"
)

// TestAccountsAnalysis analyses accounts on a simple trie with 2 accounts
func Test2AccountsAnalysis(t *testing.T) {
	store := getDb()
	smt := trie.NewTrie(nil, Hasher, store)
	key0 := make([]byte, 32, 32)
	key1 := make([]byte, 32, 32)
	bitSet(key1, 255)
	values := getFreshData(2, 32)
	smt.Update([][]byte{key0, key1}, values)
	smt.Commit()
	fmt.Println(smt.Root)

	aa := NewAccountsAnalysis(store)
	aa.Dfs(smt.Root, 0, 256, nil)
	if aa.NbUserAccounts != 2 {
		t.Fatal("Expected to find 2 accounts in the trie")
	}
	if aa.Trie.counterOn && aa.Trie.LoadDbCounter != 66 {
		// the nodes are at the tip, so 64 + 2 = 66
		t.Fatal("Expected 66 disk reads, got :", aa.Trie.LoadDbCounter)
	}

	store.Close()
	os.RemoveAll(".aergo")
}

// TestAccountsAnalysis analyses accounts on a loaded trie
func TestAccountsAnalysisFullLoad(t *testing.T) {
	// 100K accounts takes 3s to analyse but time rises steeply with 1M
	totalAccounts := uint(math.Pow(10, 5))
	store := getDb()
	smt := trie.NewTrie(nil, Hasher, store)
	loadTrieAccounts(smt, store, totalAccounts)
	fmt.Println(smt.Root)

	start := time.Now()
	aa := NewAccountsAnalysis(store)
	aa.Dfs(smt.Root, 0, 256, nil)
	if aa.NbUserAccounts != totalAccounts {
		t.Fatal("Expected to find 1000000 accounts in the trie, got", aa.NbUserAccounts)
	}
	fmt.Println(aa.Trie.LoadDbCounter)
	fmt.Println("Analysis time: ", time.Now().Sub(start))
	store.Set([]byte("Root"), smt.Root)

	store.Close()
	os.RemoveAll(".aergo")
}

func loadTrieAccounts(smt *trie.Trie, store db.DB, totalAccounts uint) {
	fmt.Println(totalAccounts)
	var keys [][]byte
	var values [][]byte
	for i := 0; i < 1000; i++ {
		if i%10 == 0 {
			fmt.Println(i)
		}
		keys = getFreshData(int(totalAccounts/1000), 32)
		values = keys
		smt.Update(keys, values)
		smt.Commit()
	}
}

func getDb() db.DB {
	dbPath := path.Join(".aergo", "db")
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		_ = os.MkdirAll(dbPath, 0711)
	}
	store := db.NewDB(db.BadgerImpl, dbPath)
	return store
}
func getFreshData(size, length int) [][]byte {
	var data [][]byte
	for i := 0; i < size; i++ {
		key := make([]byte, 32)
		_, err := rand.Read(key)
		if err != nil {
			panic(err)
		}
		data = append(data, Hasher(key)[:length])
	}
	sort.Sort(DataArray(data))
	return data
}

// DataArray is for sorting test data
type DataArray [][]byte

func (d DataArray) Len() int {
	return len(d)
}
func (d DataArray) Swap(i, j int) {
	d[i], d[j] = d[j], d[i]
}
func (d DataArray) Less(i, j int) bool {
	return bytes.Compare(d[i], d[j]) == -1
}

// Hasher is in aergo/internal so cannot be imported at this time
var Hasher = func(data ...[]byte) []byte {
	hasher := sha256.New()
	for i := 0; i < len(data); i++ {
		hasher.Write(data[i])
	}
	return hasher.Sum(nil)
}
