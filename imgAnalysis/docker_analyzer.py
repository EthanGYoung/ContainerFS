import docker
import os
import sys


# argv1: image name
# e.g. sudo python3 docker_analyzer.py ubuntu

# Future work: Enable analysis of cfs images


def num_files(path):
	return sum([len(files) for r, d, files in os.walk(path)])

def num_dirs(path):
	return sum([len(dirs) for r, dirs, f in os.walk(path)]) 

def max_depth(path, depth=0):
    if not os.path.isdir(path): return depth
    maxdepth = depth
    for entry in os.listdir(path):
        fullpath = os.path.join(path, entry)
        maxdepth = max(maxdepth, max_depth(fullpath, depth + 1))
    return maxdepth

def avg_depth(path):
	files = num_files(path)

	if (files == 0):
		return "N/A"
	else:
		return total_depth(path) / files


def total_depth(path, depth=0):
	if not os.path.isdir(path): 
		# File found
		return depth

	tot_depth = 0
	for entry in os.listdir(path):
		fullpath = os.path.join(path, entry)
		tot_depth += total_depth(fullpath, depth + 1)
	return tot_depth

def num_symlinks(path):
	if os.path.islink(path): 
		# File found
		return 1
	elif not os.path.isdir(path):
		return 0

	num_links = 0
	for entry in os.listdir(path):
		fullpath = os.path.join(path, entry)
		num_links += num_symlinks(fullpath)
	return num_links

# Subpath is the part specific to the layer (e.g. /var/lib/docker/overlay2/0b359feaa3cd877bf28e8f2de5a6a729a9d5b920382f94db3339b5541dbc5c48/diff)
def generate_file_paths(subpath, path):
	if not os.path.isdir(path):
		return [path.split(subpath)[1]]

	paths = []
	for entry in os.listdir(path):
		fullpath = os.path.join(path, entry)
		paths += generate_file_paths(subpath,fullpath)
	return paths

def generate_dir_paths(subpath, path):
	dir_paths = []
	if not os.path.isdir(path):
		return [] 
	else:
		dir_paths += [path.split(subpath)[1]]


	for entry in os.listdir(path):
		fullpath = os.path.join(path, entry)
		dir_paths += generate_dir_paths(subpath,fullpath)
	return dir_paths

def similarity(paths, curr_layer, layers_below):
	num_matching = 0
	total_path_length = 0
	total_num_layers = 0

	# Get initial count
	for d in paths[curr_layer]:
		for i in range(0, len(layers_below)):
			if d in paths[layers_below[i]]:
				num_matching += 1
				total_path_length += len(d.split("/"))
				break # Only seeing if match below
	
	# Get path lengths and layer counts
	for d in paths[curr_layer]:
		num_layers = 0
		for i in range(0, len(layers_below)):
			if d in paths[layers_below[i]]:
				total_num_layers += 1

	stats = {}
	stats["Matches"] = num_matching
	if (num_matching == 0):
		stats["Match_length"] = "N/A"
		stats["Match_layers"] = "N/A"
	else:
		stats["Match_length"] = total_path_length/num_matching
		stats["Match_layers"] = total_num_layers/num_matching

	return stats

def percent(num, denom):
	if (num == "N/A" or denom == "N/A" or denom == 0):
		return "N/A"
	else:
		return (num/denom)*100

if __name__ == "__main__":
	# Connect to the image
	cli = docker.APIClient(base_url='unix://var/run/docker.sock')
	image_info = cli.inspect_image(sys.argv[1])

	data = image_info["GraphDriver"]["Data"]
	# print("Image info: " + str(image_info))

	if "LowerDir" in data:
		dirs = data["LowerDir"].split(":")
		# dirs.reverse() # Should we reverse? Want in actual order of how mounted
	else:
		dirs = []
		print("no lower dir in image ", sys.argv[1])

	dirs.insert(0,data["UpperDir"])
	# print("Dirs: " + "\n".join(dirs))
	
	layer_count = 0

	dir_paths = {} 
	for path in dirs:
		dir_paths[path] = generate_dir_paths(path, path)

	# print("Dir_paths:", dir_paths)

	file_paths = {} 
	for path in dirs:
		file_paths[path] = generate_file_paths(path, path)

	# print("File_paths:", file_paths)
	
	print("Looping through layers top to bottom")
	for layer_dir in dirs:
		layer_count += 1
		
		print("Processing layer: " + layer_dir)
		
		print("Summary statistics..")	
		print("	Num files: ", num_files(layer_dir))
		print("	Num dirs: ", num_dirs(layer_dir) + 1) # Account for root
		print("	Max depth: ", max_depth(layer_dir))
		print("	Avg depth: ", avg_depth(layer_dir))
		print("	Num symlinks: ", num_symlinks(layer_dir))
		print()

		print("Similarity..")
		dir_stats = similarity(dir_paths, layer_dir, dirs[layer_count:len(dirs)])
		file_stats = similarity(file_paths, layer_dir, dirs[layer_count:len(dirs)])

		if (file_stats["Match_length"] != "N/A"):
			file_stats["Match_length"] -= 1 # Since extra '/' in file names

		print("	Num dirs with match in lower layer: ", dir_stats["Matches"], " Percent of dirs in this layer with match: ", percent(dir_stats["Matches"], num_dirs(layer_dir) + 1), "%")
		print("		Average path length matched: ", dir_stats["Match_length"])
		print("		Average num layers matched: ", dir_stats["Match_layers"])
		print("	Num files with match in lower layer: ", file_stats["Matches"], " Percent of files in this layer with match: ", percent(file_stats["Matches"], num_files(layer_dir)), "%")
		print("		Average path length matched: ", file_stats["Match_length"]) 
		print("		Average num layers matched: ", file_stats["Match_layers"])
		print()



