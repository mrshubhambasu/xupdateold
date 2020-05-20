package merge

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"

	"../util"

	"github.com/dsnet/compress/bzip2"
)

//Bytes applies a patch with the oldfile to create the newfile
func Bytes(oldfile, patch []byte) (newfile []byte, err error) {
	return patchb(oldfile, patch)
}

//Reader applies a BSDIFF4 patch (using oldbin and patchf) to create the newbin
func Reader(oldbin io.Reader, newbin io.Writer, patchf io.Reader) error {
	oldbs, err := ioutil.ReadAll(oldbin)
	if err != nil {
		return err
	}
	diffbytes, err := ioutil.ReadAll(patchf)
	if err != nil {
		return err
	}
	newbs, err := patchb(oldbs, diffbytes)
	if err != nil {
		return err
	}
	return util.PutWriter(newbin, newbs)
}

//File applies a BSDIFF4 patch (using oldfile and patchfile) to create the newfile
func File(oldfile, newfile, patchfile string) error {
	oldbs, err := ioutil.ReadFile(oldfile)
	if err != nil {
		return fmt.Errorf("could not read oldfile '%v': %v", oldfile, err.Error())
	}
	patchbs, err := ioutil.ReadFile(patchfile)
	if err != nil {
		return fmt.Errorf("could not read patchfile '%v': %v", patchfile, err.Error())
	}
	newbytes, err := patchb(oldbs, patchbs)
	if err != nil {
		return fmt.Errorf("bspatch: %v", err.Error())
	}
	if err := ioutil.WriteFile(newfile, newbytes, 0644); err != nil {
		return fmt.Errorf("could not create newfile '%v': %v", newfile, err.Error())
	}
	return nil
}

func patchb(oldfile, patch []byte) ([]byte, error) {
	oldsize := len(oldfile)
	var newsize int
	header := make([]byte, 32)
	buf := make([]byte, 8)
	var lenread int
	var i int
	ctrl := make([]int, 3)

	f := bytes.NewReader(patch)

	//	File format:
	//		0	8	"BSDIFF40"
	//		8	8	X
	//		16	8	Y
	//		24	8	sizeof(newfile)
	//		32	X	bzip2(control block)
	//		32+X	Y	bzip2(diff block)
	//		32+X+Y	???	bzip2(extra block)
	//	with control block a set of triples (x,y,z) meaning "add x bytes
	//	from oldfile to x bytes from the diff block; copy y bytes from the
	//	extra block; seek forwards in oldfile by z bytes".

	// Read header
	if n, err := f.Read(header); err != nil || n < 32 {
		if err != nil {
			return nil, fmt.Errorf("corrupt patch %v", err.Error())
		}
		return nil, fmt.Errorf("corrupt patch (n %v < 32)", n)
	}
	// Check for appropriate magic
	if bytes.Compare(header[:8], []byte("BSDIFF40")) != 0 {
		return nil, fmt.Errorf("corrupt patch (header BSDIFF40)")
	}

	// Read lengths from header
	bzctrllen := offtin(header[8:])
	bzdatalen := offtin(header[16:])
	newsize = offtin(header[24:])

	if bzctrllen < 0 || bzdatalen < 0 || newsize < 0 {
		return nil, fmt.Errorf("corrupt patch (bzctrllen %v bzdatalen %v newsize %v)", bzctrllen, bzdatalen, newsize)
	}

	// Close patch file and re-open it via libbzip2 at the right places
	f = nil
	cpf := bytes.NewReader(patch)
	if _, err := cpf.Seek(32, io.SeekStart); err != nil {
		return nil, err
	}
	cpfbz2, err := bzip2.NewReader(cpf, nil)
	if err != nil {
		return nil, err
	}
	dpf := bytes.NewReader(patch)
	if _, err := dpf.Seek(int64(32+bzctrllen), io.SeekStart); err != nil {
		return nil, err
	}
	dpfbz2, err := bzip2.NewReader(dpf, nil)
	if err != nil {
		return nil, err
	}
	epf := bytes.NewReader(patch)
	if _, err := epf.Seek(int64(32+bzctrllen+bzdatalen), io.SeekStart); err != nil {
		return nil, err
	}
	epfbz2, err := bzip2.NewReader(epf, nil)
	if err != nil {
		return nil, err
	}

	pnew := make([]byte, newsize)

	oldpos := 0
	newpos := 0

	for newpos < newsize {
		// Read control data
		for i = 0; i <= 2; i++ {
			lenread, err = zreadall(cpfbz2, buf, 8)
			if lenread != 8 || (err != nil && err != io.EOF) {
				e0 := ""
				if err != nil {
					e0 = err.Error()
				}
				return nil, fmt.Errorf("corrupt patch or bzstream ended: %s (read: %v/8)", e0, lenread)
			}
			ctrl[i] = offtin(buf)
		}
		// Sanity-check
		if newpos+ctrl[0] > newsize {
			return nil, fmt.Errorf("corrupt patch (sanity check)")
		}

		// Read diff string
		// lenread, err = dpfbz2.Read(pnew[newpos : newpos+ctrl[0]])
		lenread, err = zreadall(dpfbz2, pnew[newpos:newpos+ctrl[0]], ctrl[0])
		if lenread < ctrl[0] || (err != nil && err != io.EOF) {
			e0 := ""
			if err != nil {
				e0 = err.Error()
			}
			return nil, fmt.Errorf("corrupt patch or bzstream ended (2): %s", e0)
		}
		// Add pold data to diff string
		for i = 0; i < ctrl[0]; i++ {
			if oldpos+i >= 0 && oldpos+i < oldsize {
				pnew[newpos+i] += oldfile[oldpos+i]
			}
		}

		// Adjust pointers
		newpos += ctrl[0]
		oldpos += ctrl[0]

		// Sanity-check
		if newpos+ctrl[1] > newsize {
			return nil, fmt.Errorf("corrupt patch newpos+ctrl[1] newsize")
		}

		// Read extra string
		// epfbz2.Read was not reading all the requested bytes, probably an internal buffer limitation ?
		// it was encapsulated by zreadall to work around the issue
		lenread, err = zreadall(epfbz2, pnew[newpos:newpos+ctrl[1]], ctrl[1])
		if lenread < ctrl[1] || (err != nil && err != io.EOF) {
			e0 := ""
			if err != nil {
				e0 = err.Error()
			}
			return nil, fmt.Errorf("corrupt patch or bzstream ended (3): %s", e0)
		}
		// Adjust pointers
		newpos += ctrl[1]
		oldpos += ctrl[2]
	}

	// Clean up the bzip2 reads
	if err = cpfbz2.Close(); err != nil {
		return nil, err
	}
	if err = dpfbz2.Close(); err != nil {
		return nil, err
	}
	if err = epfbz2.Close(); err != nil {
		return nil, err
	}
	cpfbz2 = nil
	dpfbz2 = nil
	epfbz2 = nil
	cpf = nil
	dpf = nil
	epf = nil

	return pnew, nil

}

// offtin reads an int64 (little endian)
func offtin(buf []byte) int {

	y := int(buf[7] & 0x7f)
	y = y * 256
	y += int(buf[6])
	y = y * 256
	y += int(buf[5])
	y = y * 256
	y += int(buf[4])
	y = y * 256
	y += int(buf[3])
	y = y * 256
	y += int(buf[2])
	y = y * 256
	y += int(buf[1])
	y = y * 256
	y += int(buf[0])

	if (buf[7] & 0x80) != 0 {
		y = -y
	}
	return y
}

func zreadall(r io.Reader, b []byte, expected int) (int, error) {
	var allread int
	var offset int
	for {
		nread, err := r.Read(b[offset:])
		if nread == expected {
			return nread, err
		}
		if err != nil {
			return allread + nread, err
		}
		allread += nread
		if allread >= expected {
			return allread, nil
		}
		offset += nread
	}
}
