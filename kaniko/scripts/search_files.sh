#!/bin/sh

KEYWORDS="wget\|curl\|good.txt\|bye.txt"
TGZ_FILES=$(find ./cache -name '*.tgz')
for f in $TGZ_FILES
do
  echo $f
  tar -tvf $f | grep $KEYWORDS
done