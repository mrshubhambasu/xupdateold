package diff

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"

	"../util"
	"github.com/dsnet/compress/bzip2"
)

// Bytes takes the old and new byte slices and outputs the diff
func Bytes(oldbs, newbs []byte) ([]byte, error) {
	return diffb(oldbs, newbs)
}

// Reader takes the old and new binaries and outputs to a stream of the diff file
func Reader(oldbin io.Reader, newbin io.Reader, patchf io.Writer) error {
	oldbs, err := ioutil.ReadAll(oldbin)
	if err != nil {
		return err
	}
	newbs, err := ioutil.ReadAll(newbin)
	if err != nil {
		return err
	}
	diffbytes, err := diffb(oldbs, newbs)
	fmt.Println("diffbytes→", diffbytes)
	if err != nil {
		return err
	}
	return util.PutWriter(patchf, diffbytes)
}

// File reads the old and new files to create a diff patch file
func File(oldfile, newfile, patchfile string) error {
	oldbs, err := ioutil.ReadFile(oldfile)
	if err != nil {
		return fmt.Errorf("could not read oldfile '%v': %v", oldfile, err.Error())
	}
	newbs, err := ioutil.ReadFile(newfile)
	if err != nil {
		return fmt.Errorf("could not read newfile '%v': %v", newfile, err.Error())
	}
	diffbytes, err := diffb(oldbs, newbs)
	if err != nil {
		return fmt.Errorf("bsdiff: %v", err.Error())
	}
	if err := ioutil.WriteFile(patchfile, diffbytes, 0644); err != nil {
		return fmt.Errorf("could create patchfile '%v': %v", patchfile, err.Error())
	}
	return nil
}

func diffb(oldbin, newbin []byte) ([]byte, error) {
	bziprule := &bzip2.WriterConfig{
		Level: bzip2.BestCompression,
	}
	iii := make([]int, len(oldbin)+1)
	fmt.Println("old→", string(oldbin))
	qsufsort(iii, oldbin)

	fmt.Println("sorted→", (iii))

	//var db
	var dblen, eblen int

	// create the patch file
	pf := new(util.BufWriter)

	// Header is
	//	0	8	 "BSDIFF40"
	//	8	8	length of bzip2ed ctrl block
	//	16	8	length of bzip2ed diff block
	//	24	8	length of pnew file */
	// File is
	//  0	32	Header
	//  32	??	Bzip2ed ctrl block
	//  ??	??	Bzip2ed diff block
	//  ??	??	Bzip2ed extra block

	newsize := len(newbin)
	oldsize := len(oldbin)

	header := make([]byte, 32)
	buf := make([]byte, 8)

	copy(header, []byte("BSDIFF40"))
	offtout(0, header[8:])
	offtout(0, header[16:])
	offtout(newsize, header[24:])
	// fmt.Println(string(header))
	if _, err := pf.Write(header); err != nil {
		return nil, err
	}
	// Compute the differences, writing ctrl as we go
	pfbz2, err := bzip2.NewWriter(pf, bziprule)
	if err != nil {
		return nil, err
	}
	var scan, ln, lastscan, lastpos, lastoffset int

	var oldscore, scsc int
	var pos int

	var s, Sf, lenf, Sb, lenb int
	var overlap, Ss, lens int

	db := make([]byte, newsize+1)
	eb := make([]byte, newsize+1)

	defer func() {
		if pfbz2 != nil {
			pfbz2.Close()
		}
	}()

	for scan < newsize {
		oldscore = 0

		// scsc = scan += len
		scan += ln
		scsc = scan
		for scan < newsize {
			ln = search(iii, oldbin, newbin[scan:], 0, oldsize, &pos)

			for scsc < scan+ln {
				if scsc+lastoffset < oldsize && oldbin[scsc+lastoffset] == newbin[scsc] {
					oldscore++
				}
				scsc++
			}
			if ln == oldscore && ln != 0 {
				break
			}
			if ln > oldscore+8 {
				break
			}
			if scan+lastoffset < oldsize && oldbin[scan+lastoffset] == newbin[scan] {
				oldscore--
			}
			//
			scan++
		}

		if ln != oldscore || scan == newsize {
			s = 0
			Sf = 0
			lenf = 0
			i := 0
			for lastscan+i < scan && lastpos+i < oldsize {
				if oldbin[lastpos+i] == newbin[lastscan+i] {
					s++
				}
				i++
				if s*2-i > Sf*2-lenf {
					Sf = s
					lenf = i
				}
			}

			lenb = 0
			if scan < newsize {
				s = 0
				Sb = 0
				for i = 1; scan >= lastscan+i && pos >= i; i++ {
					if oldbin[pos-i] == newbin[scan-i] {
						s++
					}
					if s*2-i > Sb*2-lenb {
						Sb = s
						lenb = i
					}
				}
			}

			if lastscan+lenf > scan-lenb {
				overlap = (lastscan + lenf) - (scan - lenb)
				s = 0
				Ss = 0
				lens = 0
				for i = 0; i < overlap; i++ {
					if newbin[lastscan+lenf-overlap+i] == oldbin[lastpos+lenf-overlap+i] {
						s++
					}

					if newbin[scan-lenb+i] == oldbin[pos-lenb+i] {
						s--
					}
					if s > Ss {
						Ss = s
						lens = i + 1
					}
				}

				lenf += lens - overlap
				lenb -= lens
			}

			for i = 0; i < lenf; i++ {
				db[dblen+i] = newbin[lastscan+i] - oldbin[lastpos+i]
			}
			for i = 0; i < (scan-lenb)-(lastscan+lenf); i++ {
				eb[eblen+i] = newbin[lastscan+lenf+i]
			}

			dblen += lenf
			eblen += (scan - lenb) - (lastscan + lenf)

			offtout(lenf, buf)
			if _, err := pfbz2.Write(buf); err != nil {
				return nil, err
			}

			offtout((scan-lenb)-(lastscan+lenf), buf)
			if _, err := pfbz2.Write(buf); err != nil {
				return nil, err
			}

			offtout((pos-lenb)-(lastpos+lenf), buf)
			if _, err := pfbz2.Write(buf); err != nil {
				return nil, err
			}

			lastscan = scan - lenb
			lastpos = pos - lenb
			lastoffset = pos - scan
		}
	}
	if err = pfbz2.Close(); err != nil {
		return nil, err
	}

	// Compute size of compressed ctrl data
	ln = pf.Len()
	offtout(ln-32, header[8:])

	// Write compressed diff data
	pfbz2, err = bzip2.NewWriter(pf, bziprule)
	if err != nil {
		return nil, err
	}
	fmt.Println("real diff data", db[:dblen])
	if _, err = pfbz2.Write(db[:dblen]); err != nil {
		return nil, err
	}

	if err = pfbz2.Close(); err != nil {
		return nil, err
	}
	// Compute size of compressed diff data
	newsize = pf.Len()
	offtout(newsize-ln, header[16:])
	// Write compressed extra data
	pfbz2, err = bzip2.NewWriter(pf, bziprule)
	if err != nil {
		return nil, err
	}
	if _, err = pfbz2.Write(eb[:eblen]); err != nil {
		return nil, err
	}
	if err = pfbz2.Close(); err != nil {
		return nil, err
	}
	// Seek to the beginning, write the header, and close the file
	if _, err = pf.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	if _, err = pf.Write(header); err != nil {
		return nil, err
	}

	return pf.Bytes(), nil
}

func search(iii []int, oldbin []byte, newbin []byte, st, en int, pos *int) int {
	var x, y int
	oldsize := len(oldbin)
	newsize := len(newbin)

	if en-st < 2 {
		x = matchlen(oldbin[iii[st]:], newbin)
		y = matchlen(oldbin[iii[en]:], newbin)

		if x > y {
			*pos = iii[st]
			return x
		}
		*pos = iii[en]
		return y
	}

	x = st + (en-st)/2
	cmpln := util.Min(oldsize-iii[x], newsize)
	if bytes.Compare(oldbin[iii[x]:iii[x]+cmpln], newbin[:cmpln]) < 0 {
		return search(iii, oldbin, newbin, x, en, pos)
	}
	fmt.Println("→→→→", search(iii, oldbin, newbin, st, x, pos))
	return search(iii, oldbin, newbin, st, x, pos)
}

func matchlen(oldbin []byte, newbin []byte) int {
	var i int
	oldsize := len(oldbin)
	newsize := len(newbin)
	for (i < oldsize) && (i < newsize) {
		if oldbin[i] != newbin[i] {
			break
		}
		i++
	}
	return i
}

// offtout puts an int64 (little endian) to buf
func offtout(x int, buf []byte) {
	var y int
	if x < 0 {
		y = -x
	} else {
		y = x
	}
	buf[0] = byte(y % 256)
	y -= int(buf[0])
	y = y / 256
	buf[1] = byte(y % 256)
	y -= int(buf[1])
	y = y / 256
	buf[2] = byte(y % 256)
	y -= int(buf[2])
	y = y / 256
	buf[3] = byte(y % 256)
	y -= int(buf[3])
	y = y / 256
	buf[4] = byte(y % 256)
	y -= int(buf[4])
	y = y / 256
	buf[5] = byte(y % 256)
	y -= int(buf[5])
	y = y / 256
	buf[6] = byte(y % 256)
	y -= int(buf[6])
	y = y / 256
	buf[7] = byte(y % 256)

	if x < 0 {
		buf[7] |= 0x80
	}
}

// to be changed asap
func qsufsort(iii []int, buf []byte) {
	buckets := make([]int, 256)
	vvv := make([]int, len(iii))
	var i, h, ln int
	bufzise := len(buf)

	for i = 0; i < bufzise; i++ {
		buckets[buf[i]]++
	}

	for i = 1; i < 256; i++ {
		buckets[i] += buckets[i-1]
	}

	for i = 255; i > 0; i-- {
		buckets[i] = buckets[i-1]
	}
	buckets[0] = 0

	for i = 0; i < bufzise; i++ {
		buckets[buf[i]]++
		iii[buckets[buf[i]]] = i
	}
	iii[0] = bufzise

	for i = 0; i < bufzise; i++ {
		vvv[i] = buckets[buf[i]]
	}
	vvv[bufzise] = 0

	for i = 1; i < 256; i++ {
		if buckets[i] == buckets[i-1]+1 {
			iii[buckets[i]] = -1
		}
	}
	iii[0] = -1

	for h = 1; iii[0] != -(bufzise + 1); h += h {
		ln = 0

		i = 0
		for i < bufzise+1 {
			if iii[i] < 0 {
				ln -= iii[i]
				i -= iii[i]
			} else {
				if ln != 0 {
					iii[i-ln] = -ln
				}
				ln = vvv[iii[i]] + 1 - i
				split(iii, vvv, i, ln, h)
				i += ln
				ln = 0
			}
		}
		if ln != 0 {
			iii[i-ln] = -ln
		}
	}

	for i = 0; i < bufzise+1; i++ {
		iii[vvv[i]] = i
	}
}

func split(iii, vvv []int, start, ln, h int) {
	var i, j, k, x int

	if ln < 16 {
		for k = start; k < start+ln; k += j {
			j = 1
			x = vvv[iii[k]+h]
			for i = 1; k+i < start+ln; i++ {
				if vvv[iii[k+i]+h] < x {
					x = vvv[iii[k+i]+h]
					j = 0
				}
				if vvv[iii[k+i]+h] == x {
					iii[k+j], iii[k+i] = iii[k+i], iii[k+j]
					j++
				}
			}
			for i = 0; i < j; i++ {
				vvv[iii[k+i]] = k + j - 1
			}
			if j == 1 {
				iii[k] = -1
			}
		}
		return
	}
	x = vvv[iii[start+(ln/2)]+h]
	var jj, kk int
	for i = start; i < start+ln; i++ {
		if vvv[iii[i]+h] < x {
			jj++
		} else if vvv[iii[i]+h] == x {
			kk++
		}
	}
	jj += start
	kk += jj

	i = start
	j = 0
	k = 0
	for i < jj {
		if vvv[iii[i]+h] < x {
			i++
		} else if vvv[iii[i]+h] == x {
			iii[i], iii[jj+j] = iii[jj+j], iii[i]
			j++
		} else {
			iii[i], iii[kk+k] = iii[kk+k], iii[i]
			k++
		}
	}
	for jj+j < kk {
		if vvv[iii[jj+j]+h] == x {
			j++
		} else {
			iii[jj+j], iii[kk+k] = iii[kk+k], iii[jj+j]
			k++
		}
	}
	if jj > start {
		split(iii, vvv, start, jj-start, h)
	}

	for i = 0; i < kk-jj; i++ {
		vvv[iii[jj+i]] = kk - 1
	}
	if jj == kk-1 {
		iii[jj] = -1
	}

	if start+ln > kk {
		split(iii, vvv, kk, start+ln-kk, h)
	}
}
