#!/bin/bash

num_imgs=9
num_files_per_img=1000

for (( i=0; i<$num_imgs; i++ ))
do
	path="test-trace/img$i" 

	rm -rf $path
	mkdir -p $path

	echo "Creating img$i"
	
	# Create pseudo paths in each dir
	for (( k=0; k<$num_imgs; k++))
	do
		for (( j=0; j<$num_files_per_img; j++ ))
		do
			# Intermediate dir name and ending file name
			hash="img$k-$j"
		
			# Generate random content
			txt="File in layer: $i"
			dir=$hash
			
			# Create new file and load contents
			dir_path="$path/$dir/$dir/$dir"
			mkdir -p "$dir_path"

			if [ $i -eq $k ]
			then
				new_path="$dir_path/$hash"
				touch "$new_path"
				echo $txt > $new_path	
			fi
		done
	done
done