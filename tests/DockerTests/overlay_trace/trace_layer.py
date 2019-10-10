import argparse

parser = argparse.ArgumentParser(description='Open specific file')
parser.add_argument("file", help="file path to open and read")

if __name__== "__main__":
	print("Hello world from ContainerFS python!")

	# Process args
	args = parser.parse_args()
	print("Opening file with path: " + str(args.file))

	# Open file in specific layer
	f = open(args.file, 'r')

	print("Opened file")

	# Read for debugging
	print(f.readline())