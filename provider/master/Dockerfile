FROM golang:1.13.0-buster

WORKDIR /go/src

# add apt dependencies 
RUN curl -sL https://deb.nodesource.com/setup_10.x | bash -

# install tools
RUN apt-get update \
    && apt-get install -y wget zip unzip \
    && rm -rf /var/lib/apt/lists/* \
    && wget https://storage.googleapis.com/kubernetes-release/release/v1.13.0/bin/linux/amd64/kubectl \
    && mv kubectl /usr/local/bin/kubectl \
    && chmod +x /usr/local/bin/kubectl

# install protocol buffers
RUN curl -OL https://github.com/google/protobuf/releases/download/v3.9.1/protoc-3.9.1-linux-x86_64.zip
RUN unzip protoc-3.9.1-linux-x86_64.zip -d protoc3
RUN mv protoc3/bin/* /usr/local/bin/
RUN mv protoc3/include/* /usr/local/include/
RUN go get -u github.com/golang/protobuf/protoc-gen-go

# install grpc for golang lib
RUN go get -u google.golang.org/grpc

# build synerex daemon
WORKDIR /synerex_simulation
COPY /provider/master ./provider/master
COPY /provider/simutil ./provider/simutil
COPY /util ./util
COPY /api ./api
COPY /nodeapi ./nodeapi
RUN cd provider/master && \
    sed -i 's/\r//' ./entrypoint.sh && \
    chmod +x ./entrypoint.sh

# expose port
#EXPOSE 10000
WORKDIR /synerex_simulation/provider/master
RUN go build master-provider.go
ENTRYPOINT [ "./entrypoint.sh" ]