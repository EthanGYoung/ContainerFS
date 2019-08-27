FROM gcc:4.9
COPY . /
WORKDIR /
RUN gcc -o myapp -std=gnu99 ./boot.c
ENTRYPOINT ["/imgfs-layer-0/myapp"]
