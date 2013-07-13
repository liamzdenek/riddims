#!/usr/bin/sh
cd frontend/scripts

mkdir -p audiojs
cd audiojs
wget http://kolber.github.com/audiojs/audiojs.zip
unzip audiojs.zip

cd ..
wget http://code.jquery.com/jquery-2.0.3.min.js
