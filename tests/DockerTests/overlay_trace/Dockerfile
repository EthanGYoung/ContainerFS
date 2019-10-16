FROM python:3.5.2
COPY ./test-trace/img0/ /imgs
COPY ./test-trace/img1/ /imgs
COPY ./test-trace/img2/ /imgs
COPY ./test-trace/img3/ /imgs
COPY ./test-trace/img4/ /imgs
COPY ./test-trace/img5/ /imgs
COPY ./test-trace/img6/ /imgs
COPY ./test-trace/img7/ /imgs
COPY ./test-trace/img8/ /imgs
RUN mkdir /usr/src/python_trace
COPY ./trace_layer.py /usr/src/python_trace
WORKDIR /usr/src/python_trace
CMD [ "python","./trace_layer.py" ]
