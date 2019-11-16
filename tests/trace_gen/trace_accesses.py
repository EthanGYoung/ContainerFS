import argparse

parser = argparse.ArgumentParser(description='Open specific file')
parser.add_argument("file", help="file path to open and read")

# Pertain to a path
TRACE_PATH 	= "TRACE-Original_Path-"

# Pertain to walking a component of a path
DIRENT_WALK 	= "dirent_walk"
CACHED_WALK 	= "cached_walk"

# Pertain to an individual layer
LAYER_LOOKUP 	= "layer_lookup"
BF_LOOKUP 	= "bf_lookup"
GET_FILE 	= "get_file"
LAYER_MATCH	= "lookup_match"


def AddEntry(dictionary, key, value):
	if (key not in dictionary):
		dictionary[key] = [value]
	else:
		dictionary[key].append(value)

	return dictionary

# Returns dictionary mapping each layer to all accesses at layer
# Includes: bf_lookup, layer_lookup, lookup_match, and get_file
def SplitByLayer(lines):
	layers = {}

	for line in lines:
		sp = line.split("-", 2)
		
		if (sp[1] == BF_LOOKUP or sp[1] == LAYER_LOOKUP or sp[1] == LAYER_MATCH or sp[1] == GET_FILE):
			layers = AddEntry(layers, sp[2], line)

	return layers

# Returns dictionary with num_bf_lookups, num_layer_lookups, num_lookup_matches, and num_get_files
def CountLayerAccesses(layer_lines):
	accesses = {}
	accesses["num_bf_lookups"] = 0
	accesses["num_layer_lookups"] = 0
	accesses["num_lookup_matches"] = 0
	accesses["num_get_files"] = 0

	for line in layer_lines:
		sp = line.split("-", 2)

		if (sp[1] == BF_LOOKUP):
			accesses["num_bf_lookups"] += 1	
		elif (sp[1] == LAYER_LOOKUP):
			accesses["num_layer_lookups"] += 1	
		elif (sp[1] == LAYER_MATCH):
			accesses["num_lookup_matches"] += 1	
		elif (sp[1] == GET_FILE):
			accesses["num_get_files"] += 1	

	return accesses

# Returns dictionary with num_dirent_walks and num_cached_walks
def CountCacheAccesses(lines):
	accesses = {}
	accesses["num_dirent_walks"] = 0
	accesses["num_cached_walks"] = 0

	for line in lines:
		if (DIRENT_WALK in line):
			accesses["num_dirent_walks"] += 1
		elif (CACHED_WALK in line):
			accesses["num_cached_walks"] += 1

	return accesses

# Returns dictionary with num_paths
def CountPaths(lines):
	num_paths = 0

	for line in lines:
		if (TRACE_PATH in line):
			num_paths += 1

	return num_paths

def Aggregate(lines):
	layers = SplitByLayer(lines)
	layer_accesses = {}

	for layer in layers:
		layer_accesses[layer] = CountLayerAccesses(layers[layer])
		print(str(layer.strip('\n')) + "	: " + str(layer_accesses[layer]))

	print("Totals	: " + str(CountLayerAccesses(lines)))

	accesses = CountCacheAccesses(lines)
	print(accesses)

	paths = CountPaths(lines)
	print("paths: " + str(paths))


if __name__== "__main__":
	# Process args
	args = parser.parse_args()
	print("Opening file with path: " + str(args.file))

	# Open file in specific layer
	f = open(args.file, 'r')

	print("Opened file")

	line = f.readline()
	lines = []

	# Get all trace statements
	while(line):
		if ("TRACE" in line):
			lines.append(line)
		
		line = f.readline()

	Aggregate(lines)
