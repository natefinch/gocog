. ~/golang-crosscompile/crosscompile.bash

build install

gocog @files.txt --eof --startmark={{{ --endmark=}}}

go-linux-amd64 build
mv -f gocog bin/linux64
go-linux-386 build
mv -f gocog bin/linux32
go-darwin-386 build
mv -f gocog bin/darwin32
go-darwin-amd64 build
mv -f gocog bin/darwin64
go-windows-386 build
mv -f gocog.exe bin/win32
go-windows-amd64 build
mv -f gocog.exe bin/win64