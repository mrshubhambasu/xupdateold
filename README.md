# xupdate example.

To generate the diffrence file (difference between two files)

→ go run gendiff.go oldfile newfile difffile

    ↑ it will generate the difffile
_____________________________________________________________________

To merge the patching file with the old file to generate updated file

→ go run applydiff.go oldfile newfile difffile

    ↑ it will generate the newfile
