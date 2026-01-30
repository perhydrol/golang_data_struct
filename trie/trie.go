package trie

type node struct {
	isWord bool
	kids   map[rune]*node
}

type Trie struct {
	root *node
}

func Constructor() Trie {
	return Trie{root: &node{isWord: false, kids: map[rune]*node{}}}
}

func (this *Trie) Insert(word string) {
	p := this.root.kids
	for i, v := range word {
		if next, ok := p[v]; ok {
			p = next.kids
			if i == len(word)-1 {
				next.isWord = true
			}
		} else {
			newNode := node{isWord: false, kids: map[rune]*node{}}
			if i == len(word)-1 {
				newNode.isWord = true
			}
			p[v] = &newNode
			p = newNode.kids
		}
	}
}

func (this *Trie) Search(word string) bool {
	p := this.root.kids
	for i, v := range word {
		next, ok := p[v]
		if !ok {
			break
		}
		if i == len(word)-1 {
			return next.isWord
		}
		p = next.kids
	}
	return false
}

func (this *Trie) StartsWith(prefix string) bool {
	p := this.root.kids
	for _, v := range prefix {
		next, ok := p[v]
		if !ok {
			return false
		}
		p = next.kids
	}
	return true
}
