#!/bin/bash
FILE() {
    file_name=("doc" "fasthttp_setting" "http_client_proxy" "http_client" "logger")

    mkdir -p httpclient

    for name in ${file_name[@]}; do
        file_path="https://raw.githubusercontent.com/gogo-lib/httpclient/main/$name.go"
        wget $file_path -O "httpclient/$name.go"
    done
}

FILE