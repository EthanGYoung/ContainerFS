import argparse
import timeit

parser = argparse.ArgumentParser(description='Open specific file')
parser.add_argument("file", help="file path to open and read")

def open_paths(paths):
	# Open each file
	for path in paths:
		f = open(path, 'r') # Perhaps need to close?

	# Read for debugging
	print(f.readline())

if __name__== "__main__":
	print("Hello world from ContainerFS python!")

	paths = []
	print("Generating paths")
	# Open all files in imgfs layers - In future do random? - Create array with look and then randomize
	for i in range(0,3):
		for j in range(0,1000):
			component = "img" + str(i) + "-" + str(j)
			path = "/" + component + "/" + component + "/" + component + "/" + component
			paths.append(path)

	print("Opening files")
	print(timeit.timeit("open_paths(paths)", 'from __main__ import open_paths, paths', number=1))