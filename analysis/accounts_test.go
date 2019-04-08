package analysis

import (
	"bytes"
	"fmt"
	"math"
	"math/big"
	"math/rand"
	"os"
	"path"
	"sort"
	"testing"
	"time"

	"github.com/aergoio/aergo-lib/db"
	"github.com/aergoio/aergo/pkg/trie"
	"github.com/aergoio/aergo/types"
	"github.com/golang/protobuf/proto"
	sha256 "github.com/minio/sha256-simd"
)

// TestAccountsAnalysis analyses accounts on a simple trie with 2 accounts
func Test2AccountsAnalysis(t *testing.T) {
	store := getDb()
	smt := trie.NewTrie(nil, Hasher, store)

	// authenticate data in trie
	key0 := make([]byte, 32, 32)
	key1 := make([]byte, 32, 32)
	key2 := make([]byte, 32, 32)
	bitSet(key1, 255)
	bitSet(key2, 0)
	dbKeys := getFreshData(3, 32)
	smt.Update([][]byte{key0, key1, key2}, dbKeys)
	smt.Commit()
	fmt.Println(smt.Root)

	// store data in db
	txn := store.NewTx()
	balance, _ := new(big.Int).SetString("18446744073709551616", 10)
	accountData0 := types.State{
		Balance:  balance.Bytes(),
		CodeHash: []byte("code hash"),
	}
	accountData1 := types.State{
		Balance: balance.Bytes(),
	}
	notAccount := types.TxBody{
		Nonce: 1,
	}
	raw0, _ := proto.Marshal(&accountData0)
	(txn).Set(dbKeys[0], raw0)
	raw1, _ := proto.Marshal(&accountData1)
	(txn).Set(dbKeys[1], raw1)
	raw2, _ := proto.Marshal(&notAccount)
	(txn).Set(dbKeys[2], raw2)
	txn.(db.Transaction).Commit()

	// Analyse state
	aa := NewAccountsAnalysis(store, false)
	err := aa.Dfs(smt.Root, 0, 256, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Test results
	if aa.NbUserAccounts != 1 {
		t.Fatal("Expected to find 1 user account in the trie")
	}
	if aa.NbContracts != 1 {
		t.Fatal("Expected to find 1 contract in the trie")
	}
	if aa.Trie.counterOn && aa.Trie.LoadDbCounter != 66 {
		// the nodes are at the tip, so 64 + 2 = 66
		t.Fatal("Expected 66 disk reads, got :", aa.Trie.LoadDbCounter)
	}
	expectedBalance, _ := new(big.Int).SetString("36893488147419103232", 10)
	if aa.TotalAerBalance.Cmp(expectedBalance) != 0 {
		t.Fatal("Expected 36893488147419103232 total balance, got :", aa.TotalAerBalance)
	}
	store.Close()
	os.RemoveAll(".aergo")
}

// TestAccountsAnalysis analyses accounts on a loaded trie
func TestAccountsAnalysisFullLoad(t *testing.T) {
	totalAccounts := uint(math.Pow(10, 5))
	store := getDb()
	smt := trie.NewTrie(nil, Hasher, store)
	balance, _ := new(big.Int).SetString("18446744073709551616", 10)
	accountData0 := types.State{
		Balance:  balance.Bytes(),
		CodeHash: []byte("code hash"),
	}
	accountData1 := types.State{
		Balance: balance.Bytes(),
	}
	raw0, _ := proto.Marshal(&accountData0)
	raw1, _ := proto.Marshal(&accountData1)
	fmt.Println("Generating 2 x 100K test accounts...")
	loadTrieAccounts(smt, store, totalAccounts, raw0)
	loadTrieAccounts(smt, store, totalAccounts, raw1)
	fmt.Println(smt.Root)

	aa := NewAccountsAnalysis(store, false)
	start := time.Now()
	err := aa.Dfs(smt.Root, 0, 256, nil)
	fmt.Println("Analysis time: ", time.Now().Sub(start))
	if err != nil {
		t.Fatal(err)
	}
	if aa.NbUserAccounts != totalAccounts {
		t.Fatal("Expected to find 100K accounts in the trie, got", aa.NbUserAccounts)
	}
	if aa.NbContracts != totalAccounts {
		t.Fatal("Expected to find 100K contracts in the trie, got", aa.NbContracts)
	}
	if aa.NbOtherObjects != 0 {
		t.Fatal("Expected to find 0 other objects in the trie, got", aa.NbOtherObjects)
	}
	expectedBalance := new(big.Int).Mul(balance, new(big.Int).SetUint64(uint64(totalAccounts*2)))
	if aa.TotalAerBalance.Cmp(expectedBalance) != 0 {
		t.Fatal("Expected 18446744073709551616 * 200K total balance, got :", aa.TotalAerBalance)
	}
	fmt.Println(aa.Trie.LoadDbCounter)
	store.Close()
	os.RemoveAll(".aergo")
}

func loadTrieAccounts(smt *trie.Trie, store db.DB, totalAccounts uint, raw []byte) {
	fmt.Println(totalAccounts)
	var keys [][]byte
	var dbkeys [][]byte
	for i := 0; i < 1000; i++ {
		if i%10 == 0 {
			fmt.Println(i)
		}
		keys = getFreshData(int(totalAccounts/1000), 32)
		dbkeys = getFreshData(int(totalAccounts/1000), 32)
		smt.Update(keys, dbkeys)
		smt.Commit()
		txn := store.NewTx()
		for _, dbkey := range dbkeys {
			(txn).Set(dbkey, raw)
		}
		txn.(db.Transaction).Commit()
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
