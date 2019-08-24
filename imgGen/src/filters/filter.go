
// Package filter implements a library for constructing a filter for img layer
package filter


import (
	"encoding/gob"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"

	"fileio/writer"
)


// fileType is an integer representating the file type (RegularFile, Directory, Symlink)
type fileType int

const (
	// Represent the possible file types for files
    RegularFile fileType = iota
    Directory
    Symlink
)

type Filter interface {
	

}

// Manager is an interface for creating the image file.
// This interface allows for multiple implementations of its creation.
type Manager interface {
        // WalkDir recursively traverses each directory below the root director and processes files
        // by creating Metadata.
        //
        // Parameter (dir)              : name of path relative to root dir
        // parameter (foldername)       : name of current folder
        // parameter (root)             : whether or not dir is the root dir
        WalkDir(dir string, foldername string, mod_time int64, root bool)

        // IncludeFolderBegin initializes Metadata for the beginning of a file
        //
        // parameter (name)     : name of the file beginning
        IncludeFolderBegin(name string, mod_time int64)

        // IncludeFolderEnd initializes Metadata for the end of a file
        IncludeFolderEnd()

        // IncludeFile reads the given file, adds it to the file, and creates the Metadata.
        //
        // parameter (fn)       : name of the file to be read
        // paramter (basedir)   : name of the current directory relative to root
        // return               : new offset into the image file
        IncludeFile(fn string, basedir string, mod_time int64) (int64, error)
