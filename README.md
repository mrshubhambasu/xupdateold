# xupdate

To generate the patching file (difference between two files)

→ go run diff.go oldfile newfile difffile

   it will generate the difffile


To merge the patching file with the old file to generate updated file

→ go run merge.go oldfile newfile difffile

    it will generate the newfile
