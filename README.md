# state-tools

This tool iterates all the leaves in a state Trie (of given root) stored in the aergo state db.


## Installation


```sh
$ glide install
$ cd $GOPATH/src/github.com/aergoio/state-tools
$ go install
```

## Usage

```sh
$ state-tools

Welcome to the Aergo state analysis tool:
===================================
===================================

Path/to/aergo/database/data/state:
> /Users/pa/Go_workspace/src/github.com/aergoio/aergo/.aergo/data/state

Blocks TrieRoot (b58):
> 66QMsEUpLc7zx2D4jkYMN5UbGefmdFFHePnXrZitKJc2

Count DB reads (y/n):
> y

Analysis results:
=================
Number of contracts:  1
Number of pubKey accounts:  2
Total number of accounts (pubkey + contract):  3
Number of other objects stored in trie:  0
Number of DB reads performed to iterate Trie:  1
Total Aer Balance of all pubKeys and contracts:  60000000000000000000000

```