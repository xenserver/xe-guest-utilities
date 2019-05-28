#!/bin/bash
set -x

top_dir=$(pwd)
if [ ! -z $1 ];then
    mkdir -p $1
    cd $1
fi

if [ ! -f ./bin/gosec ];then
    curl -sfL https://raw.githubusercontent.com/securego/gosec/master/install.sh | sh -s 2.0.0
fi

out_file=result.json

./bin/gosec -fmt=json -out=${out_file} ${top_dir}/...

python ${top_dir}/analyze.py ${out_file}
