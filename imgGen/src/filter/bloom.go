package filter

import (
)

// BloomFilter is a struct for creating a Bloom filter for an image file. A
// A Bloom filter specifies whether a specific file path is "definitily" not
// in the image file or is "maybe" in the file with a certain probability. 
// This struct implements the Filter interface.
type BloomFilter struct {
	// TODO:
	// NumHashFuncs = k -> Calculated from equation
	// Num Elements -> Set in initialization
	// Num Bits -> Calculated from equation
}
