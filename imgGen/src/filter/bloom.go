package filter

import (
	"github.com/spaolacci/murmur3"
	"math"
)

// Hardcoded hash functions
var HASH murmur3.Hash128 = murmur3.New128()

// BloomFilter is a struct for creating a Bloom filter for an image file. A
// A Bloom filter specifies whether a specific file path is "definitily" not
// in the image file or is "maybe" in the file with a certain probability. 
// This struct implements the Filter interface.
type BloomFilter struct {
	// FPProb (False Positive Probability) is the desired probability of a false positive in the filter
	FPProb float64

	// NumHashes represents the number of hash functions for this bloom filter (k)
	NumHashes int64

	// NumElem represents the number of elements in this filter (n)
	NumElem int64

	// FilterSize represents the number of bits in this filter (m)
	FilterSize int64

	// BitSet is the array of bits that implements a bloom filter
	BitSet []bool
}

// Initialize implements Filter.Initialize. Assumes b.NumElem is set to number of expected elements
// and FPProb is set
func (b *BloomFilter) Initialize() {
	// Check for error conditions
	if (b.NumElem < 0) {
		// Return error
		return
	}

	// Compute filter size and initialize bitarray
	b.FilterSize = b.calcFilterSize()
	b.BitSet = make([]bool, b.FilterSize)

	// Compute number of hashes (k)
	b.NumHashes = b.calcNumHashes()

}

// calcFilterSize calculates the optimal size of bit array given prob and elements
// Assumes FPProb and NumElem is set
// m = ceil((n*log(p)) / log(1 / pow(2, log(2))) 
func (b *BloomFilter) calcFilterSize() int64 {
	return int64(math.Ceil((float64(b.NumElem) * math.Log(b.FPProb)) / math.Log(1 / math.Pow(2, math.Log(2)))))
}

// calcNumHashes calculates the aptimal number of hashes given the filter size and the number of elements
// Assumes FilterSize and NumElem set
// k = round((m / n) * log(2))
func (b *BloomFilter) calcNumHashes() int64 {
	return int64(math.Round(float64(b.FilterSize / b.NumElem) * math.Log(2)))
}

// AddElement implements Filter.AddElement
func (b *BloomFilter) AddElement(elem []byte) {
	// Get the hashed value of the element
	hash := b.hashElement(elem)

	h1 := int64(hash[0:64]) // First 64 bits
	h2 := int64(hash[64:128]) // Second 64 bits

	intHash := h1

	// Set bits in bitset to represent added element -> TODO: Does int cast affect anything?
	for i:=0; i < int(b.NumHashes); i++ {
		intHash += (b.NumHashes*h2)
		bitToSet := intHash % b.FilterSize
		b.setBits(bitToSet)
	}

}

// hashElement hashes the elem passed in based on the global HASH variable
func (b *BloomFilter) hashElement(elem []byte) int64 {
	return HASH(elem)
}

// setBits will set the bits in bits to 1 in the bloom filter's bitset
func (b *BloomFilter) setBits(bits int64) {
	b.BitSet |= bits
}
// RemoveElement removes an element from the filter
//
//
func (b *BloomFilter) RemoveElement() {
	// No-op for bloom filter
}

// TestElement implements Filter.TestElement
func (b *BloomFilter) TestElement() bool {
	// TODO: Make this modular with add element
	// Get the hashed value of the element
	hash := b.hashElement(elem)
	h1 := int64(hash[0:64])	// First 64 bits
	h2 := int64(hash[64:128]) // Second 64 bits

	intHash := h1

	// Create a test bit array
	testFilter = bool[b.FilterSize]

	// Set bits in bitset to represent added element
	for i:=0; i < b.NumHashes; i++ {
		intHash += (b.NumHashes*h2)
		bitToSet := intHash % b.FilterSize
		testFilter |= bitToSet
	}

	// Test if found by checking that all bits set in BitSet
	return ((testFilter & b.BitSet) == testFilter)
}
