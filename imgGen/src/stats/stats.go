package stats

import (

)

const (

)

// stats keeps track of information pertinent to the image file
type ImgStats struct {
	// NumFiles represents number of unique files in file (Include symbolic links)
	NumFiles uint

	// NumSymLinks represents number of unique symbolic links in image file
	NumSymLinks uint

	// NumDirs represents number of directories in image file
	NumDirs uint
}

// AddFile increments NumFile in the ImgStats struct
func (s *ImgStats) AddFile() {
	s.NumFiles++
}

// AddSymLink increments NumSymLinks in the ImgStats struct
func (s *ImgStats) AddSymLink() {
	s.NumSymLinks++
}

// AddDir increments NumDirs in the ImgStats struct
func (s *ImgStats) AddDir() {
	s.NumDirs++
}
