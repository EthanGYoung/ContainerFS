import docker
import os
import sys


# argv1: image name
# e.g. sudo python3 docker_analyzer.py ubuntu

# Future work: Enable analysis of cfs images

def get_depth(path, depth=0):
    if not os.path.isdir(path): return depth
    maxdepth = depth
    for entry in os.listdir(path):
        fullpath = os.path.join(path, entry)
        maxdepth = max(maxdepth, get_depth(fullpath, depth + 1))
    return maxdepth

if __name__ == "__main__":
	# Connect to the image
	cli = docker.APIClient(base_url='unix://var/run/docker.sock')
	image_info = cli.inspect_image(sys.argv[1])

	data = image_info["GraphDriver"]["Data"]
	print("Image info: " + str(image_info))

	if "LowerDir" in data:
		dirs = data["LowerDir"].split(":")
		# dirs.reverse() # Should we reverse? Want in actual order of how mounted
	else:
		dirs = []
		print("no lower dir in image ", sys.argv[1])

	dirs.insert(0,data["UpperDir"])
	print("Dirs: " + "\n".join(dirs))
	
	layer_count = 0
	
	print("Looping through layers top to bottom")
	for layer_dir in dirs:
		print("Processing layer: " + layer_dir)
		num_files = sum([len(files) for r, d, files in os.walk(layer_dir)])
		num_dirs = sum([len(dirs) for r, dirs, f in os.walk(layer_dir)]) 
		print("Num files: ", num_files)
		print("Num dirs: ", num_dirs)
		print("Max depth: ", get_depth(layer_dir))
		layer_count += 1



