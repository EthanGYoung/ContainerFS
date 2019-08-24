// Package filter implements a library for constructing a filter for img layer
package filter


import (
)

type Filter interface {
	// Initialize creates a filter with the specified initial conditions
	//
	// 
	Initialize()

	// AddElement adds an element to the filter
	//
	// 
	AddElement()

	// RemoveElement removes an element from the filter
	//
	//
	RemoveElement()

}

