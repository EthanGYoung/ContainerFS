// Package manager implements a library for constructing an image file
package manager


import (
	"encoding/gob"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"

	"fileio/writer"
	"filter"
	"stats"
	"strings"
)

// fileType is an integer representating the file type (RegularFile, Directory, Symlink)
type fileType int

const (
	// Represent the possible file types for files
    RegularFile fileType = iota
    Directory
    Symlink
)

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

	// GenerateFilter creates a filter based on the files in the img file
	GenerateFilter()

        // WriterHeader writes the Metadata for the imagefile to the end of the image file.
        // The location of the beginning of the header is written at the very end as an int64
        WriteHeader() error
}

// FileMetadata holds information for the location of a file in the image file
type FileMetadata struct {
        // Begin indicates the beginning of a file (pointer) in the file
        Begin int64

        // End indicates the ending of a file (pointer) in the file
        End int64

        // Name indicates the name of a specific file in the file
        Name string

        // If the file is a symlink, this entry is used for link info
        Link string

	// File modification time
        ModTime int64

        // Type indicated the type of a specific file (dir, symlink or regular file)
        Type fileType
}

// Manager is the main driver of creating the image file. It writes the data and stores Metadata.
type ZarManager struct {
        // PageAlign indicates whether files will be aligned at page boundaries
        PageAlign bool

        // The FileWriter for this zar image
        Writer writer.FileWriter

        // Metadata is a list of FileMetadata structs indicating start and end of directories and files
        Metadata []FileMetadata

	// FilterMetadata is a struct refering to info about the bloom filter
	FilterMetadata filter.FilterMetadata

	// Statistics is a ImgStats struct that tracks relevant statistics for the image file
	Statistics *stats.ImgStats

	// Filter is a filter used for this image file
	Filter filter.Filter
}

type DirInfo struct {
        Name string
        ModTime int64
}

// WalkDir implemented Manager.WalkDir
func (z *ZarManager) WalkDir(dir string, foldername string, mod_time int64, root bool) {
        // root dir not marked as directory
        if !root {
                fmt.Printf("including folder: %v, name: %v\n", dir, foldername)
                z.IncludeFolderBegin(foldername, mod_time)
        }

        // Retrieve all files in current directory
        files, err := ioutil.ReadDir(dir)
        if err != nil {
                log.Fatalf("walk dir unknown err when processing dir %v", dir)
        }

        var dirs []*DirInfo

        // Process each file in the directory
        for _, file := range files {
                name := file.Name()
                symlink := file.Mode() & os.ModeSymlink != 0
                file_path := path.Join(dir, name)
                mod_time := file.ModTime().UnixNano()

                if symlink {
                        // Symbolic link is an indirection, thus read and include
                        fmt.Printf("%v is symlink.", file_path)
                        real_dest, err := os.Readlink(file_path)
                        if err != nil {
                                log.Fatalf("error. Can't read symlink file. %v", real_dest)
                        }
                        // TODO: Can we replace with file redirecting to here? Could eliminate symbolic links
                        z.IncludeSymlink(name, real_dest, mod_time)
                } else {
                        if !file.IsDir() {
                                fmt.Printf("including file: %v\n", name)
                                z.IncludeFile(name, dir, mod_time)
                        } else {
                                dirs = append(dirs, &DirInfo{name, mod_time})
                        }
                }
        }

        // Recursively search each directory (DFS)
        // After file processing to improve spatial locatlity for files
        for _, subDir := range dirs {
                z.WalkDir(path.Join(dir, subDir.Name), subDir.Name, subDir.ModTime, false)
        }

        // root dir not marked as directory
        if !root {
                z.IncludeFolderEnd()
        }
}

// TODO: Change to interface for Metadata to have diff types of Metadata
// IncludeFolderBegin implements Manager.IncludeFolderBegin
func (z *ZarManager) IncludeFolderBegin(name string, mod_time int64) {
        h := &FileMetadata{
                    Begin   : -1,
                    End     : -1,
                    Name    : name,
                    Type    : Directory,
                    ModTime : mod_time,
        }

        // Add to the image's Metadata at end
        z.Metadata = append(z.Metadata, *h)

	z.Statistics.AddDir()
}

// IncludeFolderEnd implements IncludeFolderEnd
func (z *ZarManager) IncludeFolderEnd() {
        h := &FileMetadata{
                        Begin   : -1,
                        End     : -1,
                        Name    : "..",
                        Type    : Directory,
        }

        // Add to the image's Metadata at end
        z.Metadata = append(z.Metadata, *h)
}

// IncludeSymlink adds Metadata to the image file for a symbolic link. This
// allows for paths to be indirections. Not included in interface because
// not necessarily fundamental for correctness.
//
// parameter (name)     : name of file
// parameter (link)     : the actual path to the desired file
// parameter (mod_time) : the modification time fo the file

func (z *ZarManager) IncludeSymlink(name string, link string, mod_time int64) {
        h := &FileMetadata{
                        Begin   : -1,
                        End     : -1,
                        Name    : name,
                        Link    : link,
                        Type    : Symlink,
                        ModTime : mod_time,
        }
        z.Metadata = append(z.Metadata, *h)

	z.Statistics.AddSymLink()
}

// IncludeFile implements Manager.IncludeFile
func (z *ZarManager) IncludeFile(fn string, basedir string, mod_time int64) (int64, error) {
        content, err := ioutil.ReadFile(path.Join(basedir, fn))
        if err != nil {
                log.Fatalf("can't include file %v, err: %v", fn, err)
                return 0, nil
        }

        // Retrieve the current offset into the file and write the file contents
        oldCounter := z.Writer.Count
        real_end, err := z.Writer.Write(content, z.PageAlign)
        if err != nil {
                        log.Fatalf("can't write to file")
                        return 0, err
        }

        // Create the file Metadata
        h := &FileMetadata{
                        Begin   : oldCounter,
                        End     : real_end,
                        Name    : fn,
                        Type    : RegularFile,
                        ModTime : mod_time,
        }
        z.Metadata = append(z.Metadata, *h)

	z.Statistics.AddFile()

        return real_end, err
}

// GenerateFilter implements manager.GenerateFilter
func (z *ZarManager) GenerateFilter() {
	// Check type of filter

	// Initialize filter (TODO: Check error)
	z.Filter.Initialize()

	// Construct filter
	z.constructFilter()

	// Create FilterMetadata
	z.FilterMetadata = filter.FilterMetadata{
		Active:true,
		Name:"BloomFilter",
		Filter:z.Filter,
	}
}

// ConstructFilter initializes a filter by looping over FileMetadata
// and adding each file to the filter
// Algorithm: 
//	- string to hold current path
//	- When encounter startDir, append '/dirname' to string
//	- When encounter file/symlink, hash string + filename into filter
// 	- When encounter endDir, remove previous name from string
func (z *ZarManager) constructFilter() {
	fmt.Println("Constructing Filter")

	var path = ""

	for i:=0; i < len(z.Metadata); i++ {
		name:=z.Metadata[i].Name

		switch MetaType := z.Metadata[i].Type; MetaType {
		case (RegularFile):
			fmt.Println("File")
			// Add to filter
			z.Filter.AddElement([]byte(path + "/" + name))
		case (Directory):
			// Name = ".." means end of directory
			if (name == "..") {
				// Remove dir name at end
				intPath := strings.Split(path, "/")
				intPath = intPath[:len(intPath)-1]
				path = strings.Join(intPath, "/")
			} else {
				path += "/" + name
			}
		case (Symlink):
			// Treat like a file, add and hash
			z.Filter.AddElement([]byte(path + "/" + name))
		}
	}
}

// TODO: Is gob the best choice here?
// TODO: In future, can this be laid out as the struct, and directly mapped into memory?
// WriteHeader implements Manager.WriteHeader
func (z *ZarManager) WriteHeader() error {
        headerLoc := z.Writer.Count     // Offset for Metadata in image file
        fmt.Printf("header location: %v bytes\n", headerLoc)

        mEnc := gob.NewEncoder(z.Writer.W)

        fmt.Println("current Metadata:", z.Metadata)
        mEnc.Encode(z.Metadata)

        // Write location of Metadata to end of file
        z.Writer.WriteInt64(int64(headerLoc))

	// Write filter metadata to file
        filterLoc := z.Writer.Count     // Offset for Metadata in image file
        fmt.Printf("filter location: %v bytes\n", filterLoc)

        fmt.Println("current FilterMetadata:", z.FilterMetadata)
        mEnc.Encode(z.FilterMetadata)

	// Write location of Metadata to end of file
        z.Writer.WriteInt64(int64(filterLoc))

	if err := z.Writer.Close(); err != nil {
                log.Fatalf("can't close zar file: %v", err)
        }
        return nil
}
