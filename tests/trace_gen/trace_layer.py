import argparse

parser = argparse.ArgumentParser(description='Open specific file')
parser.add_argument("file", help="file path to open and read")
parser.add_argument("path", help="path for trace")

TRACE_PATH 	= "TRACE-Original_Path-"
LAYER_LOOKUP 	= "layer_lookup"
BF_LOOKUP 	= "bf_lookup"
DIRENT_WALK 	= "dirent_walk"
CACHED_WALK 	= "cached-walk"
GET_FILE 	= "get_file"
LAYER_MATCH	= "lookup_match"


def trace(lines, path):
	# Upper left white space
	print(",", end = '')

	# Row Headers
	print(path, end = '')
	print(path.replace("/",','))
	
	components = path.split("/")

	layers = GetLayers(lines, path)
	
	# Create entries in layers for each component
	for layer in layers:
		layers[layer][path] = ""
		
		for comp in components:
			if (comp == ''):
				continue
			layers[layer][comp] = ""

	layers = GenTrace(lines, layers, path)
	
	PrintTrace(layers, components, path)


def GetLayers(lines, path):
	layers = {}
	layers[path] = {}

	for row in lines:
		if (LAYER_LOOKUP in row or BF_LOOKUP in row):
			sp = row.split("-", 2)
			
			# Don't add if layer already checked
			if (sp[2] not in layers):
				layers[sp[2]] = {}

	return layers
	
def GenTrace(lines, layers, path):
	count = 1
	root = ""
	target = ""

	for line in lines:
		sp = line.split("-", 2)
		
		if (sp[1] == BF_LOOKUP):
			# A Bloom Filter lookup is performed
			layers[sp[2]][path] = count
			count += 1
		elif (sp[1] == LAYER_LOOKUP):
			# Performs a lookup in a specific layer
			layers[sp[2]][target] = count
			count += 1
		elif (sp[1] == DIRENT_WALK or sp[1] == CACHED_WALK):
			# Walks overlay tree with a 'root' and 'target'
			targ = sp[2].split("target=")
			targ = targ[1][:len(targ[1])-1] # Remove ")"
			target = targ
		elif(sp[1] == GET_FILE):
			# Returns the file found
			layers[sp[2]][target] = str(layers[sp[2]][target]) +  "*"

	return layers

def PrintTrace(layers, components, path):
	for layer in layers:
		print(layer + ",", end = '')
		print(str(layers[layer][path]) + ",", end='')
		for comp in components:
			if comp == '':
				continue
			print(layers[layer][comp], end = '')
			print(",", end='')
		print()


if __name__== "__main__":
	print("Hello world from ContainerFS python!")

	# Process args
	args = parser.parse_args()
	print("Opening file with path: " + str(args.file))

	# Open file in specific layer
	f = open(args.file, 'r')

	print("Opened file")

	line = f.readline()
	
	# Read for debugging
	while(line):
		if ((str(TRACE_PATH) + str(args.path)) in line):
			lines = []
			lines.append(line)
			line = f.readline()

			while (TRACE_PATH not in line and line):
				if ("TRACE" in line):
					lines.append(line[:len(line)-1])
				
				line = f.readline() 
			trace(lines, args.path)			
		else:	
			line = f.readline()
