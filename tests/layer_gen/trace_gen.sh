#!/bin/bash

num_imgs=9
num_files_per_img=1000

for (( i=0; i<$num_imgs; i++ ))
do
	path="test-trace/img$i" 

	rm -rf $path
	mkdir $path

	echo "Creating img$i"
	for (( j=0; j<$num_files_per_img; j++ ))
	do
		# Generate random filename
		hash="img$i-$j"
	
		# Generate random content
		txt="Random txt"
		dir="dir1"
			
		if [ "$j" -gt 499 ]
		then
			dir="dir2"
		fi
		
		# Create new file and load contents
		dir_path="$path/$dir"
		mkdir -p "$dir_path"
	
		new_path="$dir_path/$hash"
		touch "$new_path"
		echo $txt > $new_path	
	done
done


