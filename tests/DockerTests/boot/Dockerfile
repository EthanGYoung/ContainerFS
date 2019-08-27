FROM gcc:4.9
COPY . /test
WORKDIR /test
RUN gcc -o myapp -std=gnu99 ./boot.c
ENTRYPOINT ["./myapp"]
