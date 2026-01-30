package trie

import "testing"

func TestTrie(t *testing.T) {
	trie := Constructor()
	trie.Insert("apple")
	if !trie.Search("apple") {
		t.Fatal()
	}
	if trie.Search("app") {
		t.Fatal()
	}
	if !trie.StartsWith("app") {
		t.Fatal()
	}
	trie.Insert("app")
	if !trie.Search("app") {
		t.Fatal()
	}
}
