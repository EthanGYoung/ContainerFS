package img_test

import (
	"filter"
	"manager"
	"testing"
	"stats"
	"strconv"
)


func TestFilterConstruction(t *testing.T) {
	var DummyMetadata = []manager.FileMetadata {

		// Create dummy metadata (Dont care about Begin, End, Link, ModTime
		manager.FileMetadata{
			Name:"root",
			Type:manager.Directory,
		},
		manager.FileMetadata{
			Name:"apples.txt",
			Type:manager.RegularFile,
		},
		manager.FileMetadata{
			Name:"Groceries",
			Type:manager.Directory,
		},
		manager.FileMetadata{
			Name:"..",
			Type:manager.Directory,
		},
		manager.FileMetadata{
			Name:"OtherApples.txt",
			Type:manager.Symlink,
		},
	}


	// Copying beginning of writeImage in main.go
	var z *manager.ZarManager

	// Initializes all fields to 0
	var stats = &stats.ImgStats{
		NumFiles:2,
		NumSymLinks:1,
		NumDirs:2,
	}
	var filter = &filter.BloomFilter{} // Default to BloomFilter

	z = &manager.ZarManager{
		Statistics	: stats,
		Filter		: filter,
		Metadata	: DummyMetadata,
	}

	// Create the bloom filter
	z.GenerateFilter()

	path:="/root/Apples.txt"
	exp:=true
	if z.Filter.TestElement([]byte(path)) {
		t.Errorf("z.Filter.TestElement([]byte('" + path + " ') return " + strconv.FormatBool(exp) + "  when it should have returned " + strconv.FormatBool(!exp))
	}

}

// TODO: TestStats
