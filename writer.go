package main

import (
	"bufio"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"encoding/binary"
	"encoding/gob"
	//"io"
	"io/ioutil"
	"log"
	"path"
	"os"
	"syscall"
)

const (
	pageBoundary = 4096	// The boundary that needs to be upheld for page alignment
)

// TODO 
// Add new struct for walking directories
// Can have one that will do a breadth first search based on current implementation
// Can have another that will walk based on a yaml file uploaded
// TODO

// fileWriter struct writes to a file
type fileWriter struct {
	// zarw is used as the writer to the file f
	zarw *bufio.Write

	// Count is the cumulative bytes written to the file f 
	count int64

	// f is the file object that the writer will write to
	f *os.File
}


// Initializes a writer by creating the image file and attaching a writer to it\
//
// Parameter (fn): Name of image file
func (w *fileWriter) Init(fn string) error {
	if w.count != 0 {
		err := "unknown error, writer counter is not 0 when initializing"
		log.Fatalf(err)
		return errors.New(err)
	}

	// Create image file
	f, err := os.Create(fn)
	if err != nil {
		log.Fatalf("can't open zar output file %v, err: %v", fn, err)
		return err
	}

	// Initiaize the buffer writer
	w.f = f
	w.zarw = bufio.NewWriter(f)
	return nil
}

// NOTE: in the new version fileWriter.Writer return the "real" end
func (w *fileWriter) Write(data []byte, pageAlign bool) (int64, error) {
	n, err := w.zarw.Write(data)
	if err != nil {
		return int64(n), err
	}

	n2 := 0
	if pageAlign {
		pad := (align - n % align) % pageBoundary
		fmt.Printf("current write size: %v, padding size: %v\n", n, pad)
		if pad > 0 {
			s := make([]byte, pad)
			n2, err = w.zarw.Write(s)
		}
	}

	//fmt.Printf("Write data %v to file, old count: %v, length: %v\n", data, w.count, n)
	realEnd := w.count + int64(n)
	w.count += int64(n + n2)

	return realEnd, err
}

// WriteInt64 writes a int64 to the fileWriter
//
// parameter (v)	: the value to be written
func (w *fileWriter) WriteInt64(v int64) (int64, error) {
	buf := make([]byte, binary.MaxVarintLen64)
	binary.PutVarint(buf, v)
	n, err := w.Write(buf, false)

	// Extends the file offset
	w.count += int64(n)
	return n, err
}

// Close closes the filewriter by flushing any buffer
func (w *fileWriter) Close() error {
	fmt.Println("Written Bytes: ", w.count)
	w.zarw.Flush()
	return w.f.Close()
}

// zarManager is the main driver of creating the image file. It writes the data and stores metadata.
type zarManager struct {
	// pageAlign indicates whether files will be aligned at page boundaries
	pageAlign bool

	// The fileWriter for this zar image
	writer fileWriter

	// metadata is a list of fileMetadata structs indicating start and end of directories and files
	metadata []fileMetadata
}

// fileMetadata holds information for the location of a file in the image file
type fileMetadata struct {
	// Begin indicates the beginning of a file (pointer) in the file
	Begin int64

	// End indicates the ending of a file (pointer) in the file
	End int64

	// Name indicates the name of a specific file in the file
	Name string
}

// WalkDir recursively traverses each directory below the root director and processes files
// by creating metadata.
//
// Parameter (dir) 		: name of path relative to root dir
// parameter (foldername) 	: name of current folder
// parameter (root)		: whether or not dir is the root dir
// TODO: Change this to breadth first search to see difference (Change to an interface to implement diff types)
func (z *zarManager) WalkDir(dir string, foldername string, root bool) {
	// root dir not marked as directory
	if !root {
		fmt.Printf("including folder: %v, name: %v\n", dir, foldername)
		z.IncludeFolderBegin(foldername)
	}

	// Retrieve all files in current directory
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		log.Fatalf("walk dir unknown err when processing dir %v", dir)
	}

	// Process each file in the directory
	for _, file := range files {
		name := file.Name()
		if !file.IsDir() {
			fmt.Printf("including file: %v\n", name)
			z.IncludeFile(name, dir)
		} else {
			// Depth first search recursively on each directory
			z.WalkDir(path.Join(dir, name), name, false)
		}
	}

	// root dir not marked as directory
	if !root {
		z.IncludeFolderEnd()
	}
}

// TODO: Change to interface for metadata to have diff types of metadata
// IncludeFolderBegin initializes metadata for the beginning of a file
//
// parameter (name)	: name of the file beginning
func (z *zarManager) IncludeFolderBegin(name string) {
	h := &fileMetadata{
			Begin	: -1,
			End	: -1,
			Name	: name,
	}

	// Add to the image's metadata at end
	z.metadata = append(z.metadata, *h)
}

// IncludeFolderEnd initializes metadata for the end of a file
func (z *zarManager) IncludeFolderEnd() {
	h := &fileMetadata{
			Begin	: -1,
			End	: -1,
			Name	: "..",
	}

	// Add to the image's metadata at end
	z.metadata = append(z.metadata, *h)
}

// IncludeFile reads the given file, adds it to the file, and creates the metadata.
//
// parameter (fn)	: name of the file to be read
// paramter (basedir)	: name of the current directory relative to root
// return		: new offset into the image file
func (z *zarManager) IncludeFile(fn string, basedir string) (int64, error) {
	content, err := ioutil.ReadFile(path.Join(basedir, fn))
	if err != nil {
		log.Fatalf("can't include file %v, err: %v", fn, err)
		return 0, nil
	}

	// Retrieve the current offset into the file and write the file contents
	oldCounter := z.writer.count
	real_end, err := z.writer.Write(content, z.pageAlign)
	if err != nil {
			log.Fatalf("can't write to file")
			return 0, err
	}

	// Create the file metadata
	h := &fileMetadata{
			Begin	: oldCounter,
			End	: real_end,
			Name	: fn,
	}
	z.metadata = append(z.metadata, *h)

	return real_end, err
}

// TODO: Is gob the best choice here?
// WriterHeader writes the metadata for the imagefile to the end of the image file.
// The location of the beginning of the header is written at the very end as an int64
func (z *zarManager) WriteHeader() error {
	headerLoc := z.writer.count	// Offset for metadata in image file
	fmt.Printf("header location: %v bytes\n", headerLoc)

	mEnc := gob.NewEncoder(z.writer.zarw)

	fmt.Println("current metadata:", z.metadata)
	mEnc.Encode(z.metadata)

	// Write location of metadata to end of file
	z.writer.WriteInt64(int64(headerLoc))

	if err := z.writer.Close(); err != nil {
		log.Fatalf("can't close zar file: %v", err)
	}
	return nil
}

// writeImage acts as the "main" method by creating and initializing the zarManager, 
// beginning the recursive walk of the directories, and writing the metadata header
//
// parameter (dir)	: the root dir name
// parameter (output)	: the name of the image file
// parameter (pageAlign): whether the files in the image will be page aligned
func writeImage(dir string, output string, pageAlign bool) {
	z := &zarManager{pageAlign:pageAlign}
	z.writer.Init(output)

	// Begin recursive walking of directories
	z.WalkDir(dir, dir, true)

	// Write the metadata to end of file
	z.WriteHeader()
}

// TODO: Break up into smaller methods
// readImage will open the given file, extract the metadata, and print out
// the structure and/or data for each file and directory in the image file.
//
// parameter (img)	: name of the image file to be read
// parameter (detail)	: whether to print extra information (file data)
func readImage(img string, detail bool) error {
	f, err := os.Open(img)
	if err != nil {
		log.Fatalf("can't open image file %v, err: %v", img, err)
		return err
	}

	fi, err := f.Stat()
	if err != nil {
		log.Fatalf("can't stat image file %v, err: %v", img, err)
	}

	length := int(fi.Size()) // MMAP limitation. May not support large file in32 bit system
	fmt.Printf("this image file has %v bytes\n", length)

	// mmap image into address space
	mmap, err := syscall.Mmap(int(f.Fd()), 0, length, syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil {
		log.Fatalf("can't mmap the image file, err: %v", err)
	}

	if detail {
		fmt.Println("MMAP data:", mmap)
	}

	// header location is specifed by int64 at last 10 bits (bytes?)
	headerLoc := mmap[length - 10 : length]
	fmt.Println("header data:", headerLoc)

	// Setup reader for header data
	headerReader := bytes.NewReader(headerLoc)
	n, err := binary.ReadVarint(headerReader)
	if err != nil {
		log.Fatalf("can't read header location, err: %v", err)
	}
	fmt.Printf("headerLoc: %v bytes\n", n)

	var metadata []fileMetadata
	header := mmap[int(n) : length - 10]
	fmt.Println("metadata data:", header)

	// Decode the metadata in the header
	metadataReader := bytes.NewReader(header)
	dec := gob.NewDecoder(metadataReader)
	errDec := dec.Decode(&metadata)
	if errDec != nil {
		  log.Fatalf("can't decode metadata data, err: %v", errDec)
			return err
	}
	fmt.Println("metadata data decoded:", metadata)

	// Print the structure (and data) of the image file
	for _, v := range metadata {
		if v.Begin == -1 {
			fmt.Printf("enter folder: %v\n", v.Name)
		} else {
			var fileString string

			// the detail flag will print out file data, too
			if detail {
				fileBytes := mmap[v.Begin : v.End]
				fileString = string(fileBytes)
			} else {
				fileString = "ignored"
			}
			fmt.Printf("file: %v, data: %v\n", v.Name, fileString)
		}
	}
	return nil
}

func main() {
	// TODO: Add config file for version number
	fmt.Println("zar image generator version 1")

	// TODO: Add flag for info logging
	// Handle flags
	dirPtr := flag.String("dir", "./", "select the root dir to generate image")
	imgPtr := flag.String("img", "test.img", "select the image to read")
	outputPtr := flag.String("o", "test.img", "output img name")
	writeModePtr := flag.Bool("w", false, "generate image mode")
	readModePtr := flag.Bool("r", false, "read image mode")
	pageAlignPtr := flag.Bool("pagealign", false, "align the page")
	detailModePtr := flag.Bool("detail", false, "show original context when read")
	flag.Parse()

	if *writeModePtr {
		fmt.Printf("root dir: %v\n", *dirPtr)
		writeImage(*dirPtr, *outputPtr, *pageAlignPtr)
	}

	if (*readModePtr) {
		fmt.Printf("img selected: %v\n", *imgPtr)
		readImage(*imgPtr, *detailModePtr)
	}
}
