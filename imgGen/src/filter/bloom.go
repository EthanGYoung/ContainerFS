package filter

import (
	"bitarray"
)

// Hardcoded hash functions
var HASH hash.Hash128 = murmur3.New128()

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
	BitSet bool[]
}

// Initialize implements Filter.Initialize
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
	b.NumHashes = calcNumHashes()

}

// calcFilterSize calculates the optimal size of bit array given prob and elements
// m = ceil((n*log(p)) / log(1 / pow(2, log(2))) 
func (b *BloomFilter) calcFilterSize() int64 {
	return Ceil((b.NumElem * Log(b.FPProb)) / Log(1 / Pow(2, Log(2))))
}

// calcNumHashes calculates the aptimal number of hashes given the filter size and the number of elements
// k = round((m / n) * log(2))
func (b *BloomFilter) calcNumHashes() int64 {
	return Round((b.FilterSize / b.NumElem) * Log(2))
}

// AddElement adds an element to the filter
//
// 
AddElement()

// RemoveElement removes an element from the filter
//
//
RemoveElement()

TestElement()
