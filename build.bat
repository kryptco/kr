mkdir bin
cd src
go.exe build -v -trimpath -o ../bin/kr.exe ./kr
go.exe build -v -trimpath -o ../bin/krd.exe ./krd
go.exe build -v -trimpath -o ../bin/krssh.exe ./krssh
cd ..
