FROM python:3.5.2
COPY . /
RUN mkdir /usr/src/python_trace
COPY ./trace_layer.py /usr/src/python_trace
WORKDIR /usr/src/python_trace
CMD [ "python","/usr/src/python_trace/trace_layer.py" ]
