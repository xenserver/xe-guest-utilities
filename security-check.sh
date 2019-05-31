#!/bin/bash
set -x

top_dir=$(pwd)
out_dir=""

if [ ! -z $1 ];then
    mkdir -p $1
    out_dir=$1
fi

tmp_dir=`mktemp -d`
cd $tmp_dir

if [ ! -f ./bin/gosec ];then
    curl -sfL https://raw.githubusercontent.com/securego/gosec/master/install.sh | sh -s 2.0.0
fi

result_file=result.json
issue_file=issues.txt

./bin/gosec -fmt=json -out=${result_file} ${top_dir}/...


python ${top_dir}/analyze.py -i ${result_file} ${issue_file}
ret=$?

rm $result_file
chmod 666 $issue_file
if [ "x" != "x$out_dir" ];then
    mv $issue_file $out_dir
fi
exit $ret
