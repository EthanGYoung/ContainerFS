package filter

import (
	"bitarray"
)

// BloomFilter is a struct for creating a Bloom filter for an image file. A
// A Bloom filter specifies whether a specific file path is "definitily" not
// in the image file or is "maybe" in the file with a certain probability. 
// This struct implements the Filter interface.
type BloomFilter struct {
	// NumHashes represents the number of hash functions for this bloom filter
	NumHashes int64

	// NumElem represents the number of elements in this filter
	NumElem int64

	// FilterSize represents the number of bits in this filter
	FilterSize int64

	// FilterArray is the array of bits that implements a bloom filter
	FilterArray bitarray
}
