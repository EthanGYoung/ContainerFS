// Package filter implements a library for constructing a filter for img layer
package filter


import (
)

type Filter interface {
	// Initialize creates a filter with the specified initial conditions
	//
	// 
	Initialize()

	// AddElement adds an element to the filter by hashing the element into the filter
	//
	// elem:	Represents an element to add to the bloom filter 
	AddElement(elem []byte)

	// RemoveElement removes an element from the filter
	//
	//
	RemoveElement()

	// TestElement checks if the specific element exists in data structure
	//
	// Return: False if not present in filter, true if possibly present
	TestElement(elem []byte) bool

}

