// dvitype in Go
// 1st working version. Needs TFM in current directory
// License: MIT, more information will follow
// Contact: Patrick Gundlach, gundlach@speedata.de
// Why? Why not.

package dvitype

import (
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
)

func round(f float32) int {
	if f > 0 {
		return int(f + 0.5)
	}
	return int(f - 0.5)
}

type Dvitype struct {
	OutMode    int
	PageSpec   string
	MaxPages   int
	Resolution float32
	dvifile    io.ReadSeeker
	tfmfile    io.Reader
	dvisize    int64
	curloc     int64
}

const (
	set_char_0 = 0
	set1       = 128 // typeset a character and move right
	set_rule   = 132 // typeset a rule and move right
	put1       = 133 // typeset a character
	put_rule   = 137 // typeset a rule
	nop        = 138 // no operation
	bop        = 139 // beginning of page
	eop        = 140 // ending of page
	push       = 141 // save the current positions
	pop        = 142 // restore previous positions
	right1     = 143 // move right
	w0         = 147 // move right by w
	w1         = 148 // move right and set w
	x0         = 152 // move right by x
	x1         = 153 // move right and set x
	down1      = 157 // move down
	y0         = 161 // move down by y
	y1         = 162 // move down and set y
	z0         = 166 // move down by z
	z1         = 167 //  move down and set z
	fnt_num_0  = 171 // set current font to 0
	fnt1       = 235 // set current font
	xxx1       = 239 // extension to DVI primitives
	xxx4       = 242 // potentially long extension to DVI primitives
	fnt_def1   = 243 // the meaning of a font number
	pre        = 247 // preamble
	post       = 248 // postamble beginning
	post_post  = 249 // postamble ending
	undef1     = 250
	undef2     = 251
	undef3     = 252
	undef4     = 253
	undef5     = 254
	undef6     = 255

	ID_BYTE = 2

	max_fonts            = 100   // maximum number of distinct fonts per DVI file
	max_widths           = 10000 // maximum number of different characters among all fonts
	line_length          = 79    // bracketed lines of output will be at most this long
	terminal_line_length = 150   // maximum number of characters input in a single line of input from the terminal
	stack_size           = 100   // DVI files shouldn’t push beyond this depth
	name_size            = 1000  // total length of all font file names
	name_length          = 50    // a file name shouldn’t be longer than this

	first_text_char = 0
	last_text_char  = 127

	errors_only    = 0 // value of out mode when minimal printing occurs
	terse          = 1 // value of out mode for abbreviated output
	mnemonics_only = 2 // value of out mode for medium-quantity output
	verbose        = 3 // value of out mode for detailed tracing
	the_works      = 4 // verbose, plus check of postamble if random reading

	invalid_width = 017777777777
	infinity      = 017777777777
	maxdrift      = 2

	invalid_font = max_fonts
)

var (
	new_mag int // if positive, overrides the postamble’s magnification

	// 72
	h, v, w, x, y, z, hh, vv                       int             // current state values
	hstack, vstack, wstack, xstack, ystack, zstack [stack_size]int // pushed down values in DVI units
	hhstack, vvstack                               [stack_size]int //  pushed down values in pixels

	in_postamble bool

	start_count []int   // count values to select starting page
	start_there []bool  // is the start count value relevant?
	start_vals  uint8   // the last count considered significant
	count       [10]int // the count values on the current page

	// 10
	// xord:array[char]of ASCIIcode;xchr:array[0..255]of char;
	// 22

	curname [name_length + 1]uint8
	// {:24}{25:}
	b0, b1, b2, b3 eightbits
	// {:25}{30:}
	font_num       = make([]int, max_fonts)
	fontname       = make([]int, max_fonts)
	names          = make([]uint8, name_size)
	fontchecksum   = make([]int, max_fonts)
	fontscaledsize = make([]int, max_fonts)
	fontdesignsize = make([]int, max_fonts)
	fontspace      = make([]int, max_fonts+1)
	fontbc         = make([]int, max_fonts)
	fontec         = make([]int, max_fonts)
	widthbase      = make([]int, max_fonts)
	width          = make([]int, max_widths)
	nf             int
	widthptr       int
	// ;{:30}{33:}
	inwidth       [255]int
	tfmchecksum   int
	tfmdesignsize int
	tfmconv       float32
	// {:33}{39:}
	pixelwidth             = make([]int, max_widths)
	conv                   float32
	true_conv              float32
	numerator, denominator int
	mag                    int

	// {:42}{
	// 45
	buffer []uint8
	// termin:textfile;termout:textfile;{:45}
	// 48
	buf_ptr int
	// {57:}inpostamble:boolean;
	// {:64}{67:}textptr:0..linelength;

	// 64
	defaultdirectory []uint8
	// 67
	textptr int
	//
	textbuf [line_length]uint8

	// 73
	maxv                            int
	maxh                            int
	maxs                            int
	maxvsofar, maxhsofar, maxssofar int
	totalpages                      int
	pagecount                       int

	// 78
	s       int
	ss      int
	curfont int
	showing bool

	// 97
	old_backpointer int64
	new_backpointer int64
	started         bool

	// 101
	post_loc int64

	first_backpointer int64
	startloc          int64
	afterpre          int64

	// 108
	m    int
	p, q int64
)

func bad_dvi(s string) {
	fmt.Println("Bad DVI file:", s)
	os.Exit(-1)
}

type (
	textchar  uint8
	eightbits uint8
)

// 11
func init() {

	// 12
	// 31
	// 	nf = 0; width ptr = 0; font name[0] = 1;
	// font space [invalid font ] = 0; { for out space and out vmove } font bc [invalid font ] = 1; font ec [invalid font ] = 0;
	// 43
}

// 27
func (d *Dvitype) getbyte() int {
	b, err := d.read()
	if err == io.EOF {
		return 0
	} else {
		d.curloc++
		return int(b)
	}
}

func (d *Dvitype) gettwobytes() int {
	a, _ := d.read()
	b, _ := d.read()
	d.curloc += 2
	return int(a)*256 + int(b)
}

func (d *Dvitype) getthreebytes() int {
	a, _ := d.read()
	b, _ := d.read()
	c, _ := d.read()
	d.curloc += 3
	return (int(a)*256+int(b))*256 + int(c)
}

func (d *Dvitype) signedbyte() int {
	b, _ := d.read()
	d.curloc++
	if b < 128 {
		return int(b)
	} else {
		return int(b) - 256
	}
}

func (d *Dvitype) signedpair() int {
	a, _ := d.read()
	b, _ := d.read()

	d.curloc += 2
	if a < 128 {
		return int(a)*256 + int(b)
	} else {
		return (int(a)-256)*256 + int(b)
	}
}

func (d *Dvitype) signedtrio() int {
	a, _ := d.read()
	b, _ := d.read()
	c, _ := d.read()

	d.curloc += 3
	if a < 128 {
		return (int(a)*256+int(b))*256 + int(c)
	} else {
		return ((int(a)-256)*256+int(b))*256 + int(c)
	}
}

func (d *Dvitype) signedquad() int {
	a1, _ := d.read()
	a2, _ := d.read()
	a3, _ := d.read()
	a4, _ := d.read()

	d.curloc += 4
	if a1 < 128 {
		return ((int(a1)*256+int(a2))*256+int(a3))*256 + int(a4)
	} else {
		return (((int(a1)-256)*256+int(a2))*256+int(a3))*256 + int(a4)
	}
}

// 32
func (d *Dvitype) printFont(f int) {
	if f == invalid_font {
		fmt.Print("UNDEFINED!")
	} else {
		for k := fontname[f]; k < fontname[f+1]; k++ {
			fmt.Printf("%c", names[k])
		}
	}
}

// 75
func (d *Dvitype) firstpar(o eightbits) int {
	switch o {
	case set_char_0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38, 39, 40, 41, 42, 43, 44, 45,
		46, 47, 48, 49, 50, 51, 52, 53, 54, 55, 56, 57, 58, 59, 60, 61, 62, 63, 64, 65, 66, 67, 68, 69, 70, 71, 72, 73, 74, 75, 76, 77, 78, 79, 80, 81, 82, 83, 84, 85, 86, 87, 88, 89, 90, 91, 92, 93,
		94, 95, 96, 97, 98, 99, 100, 101, 102, 103, 104, 105, 106, 107, 108, 109, 110, 111, 112, 113, 114, 115, 116, 117, 118, 119, 120, 121, 122, 123, 124, 125, 126, 127:
		return int(o - set_char_0)
	case set1, put1, fnt1, xxx1, fnt_def1:
		return d.getbyte()
	case set1 + 1, put1 + 1, fnt1 + 1, xxx1 + 1, fnt_def1 + 1:
		return d.gettwobytes()
	case set1 + 2, put1 + 2, fnt1 + 2, xxx1 + 2, fnt_def1 + 2:
		return d.getthreebytes()
	case right1, w1, x1, down1, y1, z1:
		return d.signedbyte()
	case right1 + 1, w1 + 1, x1 + 1, down1 + 1, y1 + 1, z1 + 1:
		return d.signedpair()
	case right1 + 2, w1 + 2, x1 + 2, down1 + 2, y1 + 2, z1 + 2:
		return d.signedtrio()
	case set1 + 3, set_rule, put1 + 3, put_rule, right1 + 3, w1 + 3, x1 + 3, down1 + 3, y1 + 3, z1 + 3, fnt1 + 3, xxx1 + 3, fnt_def1 + 3:
		return d.signedquad()
	case nop, bop, eop, push, pop, pre, post, post_post, undef1, undef2, undef3, undef4, undef5, undef6:
		return 0
	case w0:
		return w
	case x0:
		return x
	case y0:
		return y
	case z0:
		return z
	case fnt_num_0, fnt_num_0 + 1, fnt_num_0 + 2, 174, 175, 176, 177, 178, 179, 180, 181, 182, 183, 184, 185, 186,
		187, 188, 189, 190, 191, 192, 193, 194, 195, 196, 197, 198, 199, 200, 201, 202,
		203, 204, 205, 206, 207, 208, 209, 210, 211, 212, 213, 214, 215, 216, 217, 218,
		219, 220, 221, 222, 223, 224, 225, 226, 227, 228, 229, 230, 231, 232, 233, 234:
		return int(o - fnt_num_0)
	}
	return 0
}

func (d *Dvitype) readTFMWord() {
	var err error
	b0, err = d.readFromTFM()
	if err != nil {
		log.Fatal(err)
	}
	b1, err = d.readFromTFM()
	if err != nil {
		log.Fatal(err)
	}
	b2, err = d.readFromTFM()
	if err != nil {
		log.Fatal(err)
	}
	b3, err = d.readFromTFM()
	if err != nil {
		log.Fatal(err)
	}
}

// 34
func (d *Dvitype) inTFM(z int) bool {
	var (
		// k           int // index for loops
		lh          int // length of header data, in four-byte words
		nw          int // number of words in the width table
		wp          int // new value of width ptr after successful input
		alpha, beta int // quantities used in the scaling computation
	)

	// Read past the header data; goto 9997 if there is a problem 35:
	d.readTFMWord()
	lh = int(b2)*256 + int(b3)
	d.readTFMWord()
	fontbc[nf] = int(b0)*256 + int(b1)
	fontec[nf] = int(b2)*256 + int(b3)
	if fontec[nf] < fontbc[nf] {
		fontbc[nf] = fontec[nf] + 1
	}
	if widthptr+fontec[nf]-fontbc[nf]+1 > max_widths {
		fmt.Println("---not loaded, DVItype needs larger width table")
		return false
	}
	wp = widthptr + fontec[nf] - fontbc[nf] + 1
	d.readTFMWord()
	nw = int(b0)*256 + int(b1)
	if nw == 0 || nw > 256 {
		fmt.Println("--- not loaded, TFM file is bad (3)")
		return false
	}
	for k := 1; k <= 3+lh; k++ {
		// check for eof
		d.readTFMWord()
		if k == 4 {
			if b0 < 128 {
				tfmchecksum = ((int(b0)*256+int(b1))*256+int(b2))*256 + int(b3)
			} else {
				tfmchecksum = (((int(b0)-256)*256+int(b1))*256+int(b2))*256 + int(b3)
			}
		} else if k == 5 {
			if b0 < 128 {
				tfmdesignsize = round(tfmconv * float32(((int(b0)*256+int(b1))*256+int(b2))*256+int(b3)))
			} else {
				fmt.Println("--- not loaded, TFM file is bad (4)")
				return false
			}
		}
	}
	// :35

	// Store character-width indices at the end of the width table 36
	if wp > 0 {
		for k := widthptr; k < wp; k++ {
			d.readTFMWord()
			if int(b0) > nw {
				fmt.Println("--- not loaded, TFM file is bad (5)")
				return false
			}
			width[k] = int(b0)
		}
	}
	// :36

	// Read and convert the width values, setting up the in width table 37
	// ⟨Replace z by z′ and compute α,β 38⟩;
	alpha = 16
	for z >= 040000000 {
		z = z / 2
		alpha = alpha + alpha
	}
	beta = 256 / alpha
	alpha = alpha * z
	// :38

	for k := 0; k <= nw-1; k++ {
		d.readTFMWord()
		inwidth[k] = (((((int(b3) * z) / 0400) + (int(b2) * z)) / 0400) + (int(b1) * z)) / beta
		if b0 > 0 {
			if b0 < 255 {
				fmt.Println("--- not loaded, TFM file is bad (1)")
				return false
			} else {
				inwidth[k] = inwidth[k] - alpha
			}
		}
	}
	// :37
	// Move the widths from in width to width , and append pixel width values 40
	if inwidth[0] != 0 {
		// the first width should be zero
		fmt.Println("--- not loaded, TFM file is bad (2)")
		return false
	}
	widthbase[nf] = widthptr - fontbc[nf]
	if wp > 0 {
		for k := widthptr; k < wp; k++ {
			if width[k] == 0 {
				width[k] = invalid_width
				pixelwidth[k] = 0
			} else {
				width[k] = inwidth[width[k]]
				pixelwidth[k] = round(conv * float32(width[k]))
			}
		}
	}
	// :40
	widthptr = wp
	return true
}

func (d *Dvitype) moveToByte(pos int64) {
	var err error
	d.curloc, err = d.dvifile.Seek(pos, os.SEEK_SET)
	if err != nil {
		log.Fatal(err)
	}
}

func (d *Dvitype) read() (eightbits, error) {
	x := make([]byte, 1)
	_, err := d.dvifile.Read(x)
	return eightbits(x[0]), err
}

func (d *Dvitype) readFromTFM() (eightbits, error) {
	x := make([]byte, 1)
	_, err := d.tfmfile.Read(x)
	return eightbits(x[0]), err
}

// 59
// e is an external font number
func (d *Dvitype) defineFont(e int) {
	var (
		err error
	)
	var f int
	var _p int          //length of the area/directory spec
	var n int           // length of the font name proper
	var c, q, _d, m int // check sum, scaled size, design size, magnification
	var r int           // index into cur name
	var j, k int        // indices into names
	var mismatch bool   //  do names disagree?

	if nf == max_fonts {
		bad_dvi(fmt.Sprintf("DVItype capacity exceeded (max fonts=%d!)", max_fonts))
	}
	font_num[nf] = e
	for font_num[f] != e {
		f++
	}
	// Read the font parameters into position for font nf , and print the font name 61:
	c = d.signedquad()
	fontchecksum[nf] = c
	q = d.signedquad()
	fontscaledsize[nf] = q
	_d = d.signedquad()
	fontdesignsize[nf] = _d
	if (q <= 0) || (_d <= 0) {
		m = 1000
	} else {
		m = round((1000.0 * conv * float32(q)) / (true_conv * float32(_d)))
	}
	_p = d.getbyte()
	n = d.getbyte()
	if fontname[nf]+n+_p > name_size {
		bad_dvi(fmt.Sprintf("DVItype capacity exceeded (name size=%d)!", name_size))
	}
	fontname[nf+1] = fontname[nf] + n + _p
	if showing {
		fmt.Print(": ") // when showing is true, the font number has already been printed
	} else {
		fmt.Printf("Font %d: ", e)
	}
	if n+_p == 0 {
		fmt.Print("null font name!")
	} else {
		for k := fontname[nf]; k < fontname[nf+1]; k++ {
			names[k] = uint8(d.getbyte())
		}
	}
	d.printFont(nf)
	if !showing {
		if m != 1000 {
			fmt.Print(" scaled ", m)
		}
	}
	if ((d.OutMode == the_works) && in_postamble) || ((d.OutMode < the_works) && !in_postamble) {
		if f < nf {
			fmt.Println("---this font was already defined!")
		}
	} else {
		if f == nf {
			fmt.Println("---this font wasn't loaded before!")
		}
	}

	if f == nf {
		// Load the new font, unless there are problems 62
		// 66:
		for k := 1; k <= name_length; k++ {
			curname[k] = ' '
		}
		r = 0
		for k := fontname[nf]; k < fontname[nf+1]; k++ {
			r++
			curname[r] = names[k]
		}
		curname[r+1] = '.'
		curname[r+2] = 't'
		curname[r+3] = 'f'
		curname[r+4] = 'm'
		_fontname := string(curname[1 : r+5])
		// :66
		d.tfmfile, err = os.Open(_fontname)
		if err != nil {
			fmt.Println(err)
			fmt.Print("---not loaded, TFM file can't be opened!")
		} else {
			if (q <= 0) || (q >= 01000000000) {
				fmt.Printf("---not loaded, bad scale (%d)!", q)
			} else if (_d <= 0) || _d >= 01000000000 {
				fmt.Printf("---not loaded, bad design size (%d)!", _d)
			} else if d.inTFM(q) {
				// finish loading the new font info 63
				fontspace[nf] = q / 6 // this is a 3-unit “thin space”

				//font space [nf ] = q div 6; { }
				if (c != 0) && (tfmchecksum != 0) && (c != tfmchecksum) {
					fmt.Println("---beware: check sums do not agree!")
					fmt.Printf("   (%o vs. %o)\n   ", c, tfmchecksum)
				}
				if abs(tfmdesignsize-_d) > 2 {
					fmt.Printf("---beware: design sizes do not agree!\n")
					fmt.Printf("   (%d vs. %d)\n   ", _d, tfmdesignsize)
				}
				fmt.Print("---loaded at size ", q, " DVI units")
				_d = round((100.0 * conv * float32(q)) / (true_conv * float32(_d)))
				if _d != 100 {
					fmt.Printf("\n (this font is magnified  %d%%)", _d)
				}
				nf++ // now the new font is officially present
			}
		}
		if d.OutMode == errors_only {
			fmt.Println(" ")
		}
	} else {
		// Check that the current font definition matches the old one 60
		if fontchecksum[f] != c {
			fmt.Println("---check sum doesn't match previous definition!")
		}
		if fontscaledsize[f] != q {
			fmt.Println("--- scaled size doesn't match previous definition!")
		}
		if fontdesignsize[f] != _d {
			fmt.Println("--- design size doesn't match previous definition!")
		}
		j = fontname[f]
		k = fontname[nf]
		if fontname[f+1]-j != fontname[nf+1]-k {
			mismatch = true
		} else {
			mismatch = false
			for j < fontname[f+1] {
				if names[j] != names[k] {
					mismatch = true
				}
				j++
				k++
			}
		}
		if mismatch {
			fmt.Println("---font name doesn't match previous definition!")
		}
		// :60
	}
}

func abs(j int) int {
	if j < 0 {
		return -1 * j
	} else {
		return j
	}
}

func New(f io.ReadSeeker) *Dvitype {
	d := new(Dvitype)
	d.MaxPages = 1000000
	d.OutMode = 4
	d.PageSpec = "*"
	d.Resolution = 300.0
	d.dvifile = f
	return d
}

func (d *Dvitype) readPostamble() {
	var (
		k int // loop index
	// p, q, m int //general purpose registers
	)
	// showing := false
	post_loc = d.curloc - 5
	fmt.Printf("Postamble starts at byte %d.\n", post_loc)

	if a := d.signedquad(); a != numerator {
		fmt.Println("numerator doesn't match the preamble!")
	}

	if a := d.signedquad(); a != denominator {
		fmt.Println("denominator doesn't match the preamble!")
	}

	if a := d.signedquad(); a != mag {
		if new_mag == 0 {
			fmt.Println("magnification doesn't match the preamble!")
		}
	}
	maxv = d.signedquad()
	maxh = d.signedquad()
	fmt.Printf("maxv=%d, maxh=%d", maxv, maxh)
	maxs = d.gettwobytes()
	totalpages = d.gettwobytes()
	fmt.Printf(", maxstackdepth=%d, totalpages=%d\n", maxs, totalpages)
	if d.OutMode < the_works {
		// Compare the lust parameters with the accumulated facts 104
		if maxv+99 < maxvsofar {
			fmt.Println("warning: observed maxv was", maxvsofar)
		}
		if maxh+99 < maxhsofar {
			fmt.Println("warning: observed maxh was  ", maxhsofar)
		}
		if maxs < maxssofar {
			fmt.Println("warning: observed maxstackdepth was  ", maxssofar)
		}
		if pagecount != totalpages {
			fmt.Println("there are really", pagecount, " pages, not  ", totalpages, "!")
		}
	}
	// Process the font definitions of the postamble 106:
	for {
		k = d.getbyte()
		if k >= fnt_def1 && k < fnt_def1+4 {
			p := d.firstpar(eightbits(k))
			d.defineFont(p)
			fmt.Println()
			k = nop
		}
		if k != nop {
			break
		}
	}
	if k != post_post {
		fmt.Println("byte", d.curloc-1, "is not postpost!")
	}
	// ⟨ Make sure that the end of the file is well-formed 105 ⟩;
	q = int64(d.signedquad())
	if q != post_loc {
		fmt.Println("bad postamble pointer in byte  ", d.curloc-4, "!")
	}
	m = d.getbyte()
	if m != ID_BYTE {
		fmt.Println("identification in byte  ", d.curloc-1, " should be  ", ID_BYTE, "!")
	}
	k = int(d.curloc)
	m = 223

	// while (m = 223) && ¬eof (dvi file) do m = get byte;
	// if ¬eof (dvi file ) then bad dvi ( "signature in byte  ", d.curloc - 1 ,  " should be 223 ") else if d.curloc < k + 4 then
	// fmt.Println ( "not enough signature bytes at end of file ( ", d.curloc - k ,  ") ");
}

func (d Dvitype) Run() {
	var (
		k int
	)
	var err error
	// 50 dialog()
	fmt.Println("This is DVItype, Version 3.6")
	// Determine the desired out_mode 51

	// Determine the desired start count values 52
	start_there = make([]bool, 0)
	start_count = make([]int, 0)
	for k, v := range strings.Split(d.PageSpec, ".") {
		if v == "*" {
			start_there = append(start_there, false)
			start_count = append(start_count, 0)
		} else {
			k, err = strconv.Atoi(v)
			if err != nil {
				panic("Not a number")
			}
			start_there = append(start_there, true)
			start_count = append(start_count, k)

		}
	}

	fmt.Println("Options selected:")
	fmt.Print("  Starting page = ")
	for k, v := range start_there {
		if v {
			fmt.Print(start_count[k])
		} else {
			fmt.Print("*")
		}
		if k < len(start_count)-1 {
			fmt.Print(".")
		} else {
			fmt.Println()
		}
	}
	fmt.Printf("  Maximum number of pages =  %d\n", d.MaxPages)
	fmt.Printf("  Output level = %d", d.OutMode)
	switch d.OutMode {
	case errors_only:
		fmt.Println(" (showing bops, fonts, and error messages only)")
	case terse:
		fmt.Println(" (terse)")
	case mnemonics_only:
		fmt.Println(" (mnemonics)")
	case verbose:
		fmt.Println(" (verbose)")
	case the_works:
		fmt.Println(" (the works)")
	}
	fmt.Printf("  Resolution = %12.8f pixels per inch\n", d.Resolution)
	if new_mag > 0 {
		fmt.Printf("  New magnification factor =  %8.3f\n", new_mag/1000)
	}
	// end dialog

	// 109 A DVI-reading program that reads the postamble first need not look at the preamble; but DVItype looks at the preamble in order to do error checking, and to display the introductory comment.
	if d.getbyte() != pre {
		bad_dvi("First byte isn't start of preamble!")
	}

	if d.getbyte() != 2 {
		fmt.Printf("identification in byte 1 should be %d!\n", 2)
	}
	// Compute the conversion factors
	numerator = d.signedquad()
	denominator = d.signedquad()

	if numerator <= 0 {
		bad_dvi(fmt.Sprintf("numerator is %d", numerator))
	}
	if numerator <= 0 {
		bad_dvi(fmt.Sprintf("denominator is %d", denominator))
	}
	fmt.Printf("numerator/denominator=%d/%d\n", numerator, denominator)
	tfmconv = (25400000.0 / float32(numerator)) * float32(denominator/473628672) / 16.0
	conv = (float32(numerator) / 254000.0) * (d.Resolution / float32(denominator))
	mag = d.signedquad()
	if new_mag > 0 {
		mag = new_mag
	} else if mag <= 0 {
		bad_dvi(fmt.Sprintf("magnification is %d\n", mag))
	}
	true_conv = conv
	conv = true_conv * (float32(mag) / 1000.0)
	fmt.Printf("magnification=%d; %16.8f pixels per DVI unit\n", mag, conv)

	c := d.getbyte()
	buf := make([]byte, c)
	d.dvifile.Read(buf)
	fmt.Printf("'%s'\n", string(buf))
	d.curloc += int64(c)
	afterpre = d.curloc
	// end 109

	// 	if out_mode=the_works then {|random_reading=true|}
	if d.OutMode == the_works {
		//   begin @<Find the postamble, working back from the end@>;
		pos, err := d.dvifile.Seek(0, os.SEEK_END)
		if err != nil {
			log.Fatal(err)
		}
		d.dvisize = pos
		n := d.dvisize
		if n < 53 {
			bad_dvi(fmt.Sprintf("only %d bytes long", n))
		}
		m := n - 4
		for {
			if m == 0 {
				bad_dvi("all 223s")
			}
			d.moveToByte(m)
			k = d.getbyte()
			m--
			if k != 223 {
				break
			}
		}
		if k != ID_BYTE {
			bad_dvi(fmt.Sprintf("ID byte is %d", k))
		}
		d.moveToByte(m - 3)
		q = int64(d.signedquad())
		if (q < 0) || (q > m-33) {
			bad_dvi(fmt.Sprintf("post pointer %d at byte %d", q, m-3))
		}

		d.moveToByte(int64(q))
		k = d.getbyte()
		if k != post {
			bad_dvi(fmt.Sprintf("byte %d is not post", q))
		}

		post_loc = int64(q)
		first_backpointer = int64(d.signedquad())

		in_postamble = true
		d.readPostamble()
		in_postamble = false

		// Count the pages and move to the starting page 102
		q = post_loc
		p = first_backpointer
		startloc = -1
		if p < 0 {
			in_postamble = true
		} else {
			// now q points to a post or bop command; p >= 0 is prev pointer
			for {
				if p > q-46 {
					bad_dvi(fmt.Sprintf("page link %d after byte %d", p, q))
				}
				q = p
				d.moveToByte(q)
				k = d.getbyte()
				if k == bop {
					pagecount++
				} else {
					bad_dvi(fmt.Sprintf("byte %d is not bop (1)", q))
				}
				for k := 0; k < 10; k++ {
					count[k] = d.signedquad()
				}
				p = int64(d.signedquad())
				if start_match() {
					startloc = q
					old_backpointer = p
				}
				if p < 0 {
					break
				}
			}
			if startloc < 0 {
				bad_dvi("starting page number could not be found!")
			}
			if old_backpointer < 0 {
				startloc = afterpre // we want to check everything
			}
			d.moveToByte(startloc)
		}
		if pagecount != totalpages {
			fmt.Println("there are really ", pagecount, " pages, not ", totalpages, "! ")
		}
		// :102
	}
	d.skip_pages(false)
	if !in_postamble {
		// Translate up to max pages pages 111
		for d.MaxPages > 0 {
			d.MaxPages--
			fmt.Println()
			fmt.Print(d.curloc-45, ": beginning of page ")
			for k := 0; k <= int(start_vals); k++ {
				fmt.Print(count[k])
				if k < int(start_vals) {
					fmt.Print(".")
				} else {
					fmt.Println()
				}
			}
			if !d.doPage() {
				bad_dvi("page ended unexpectedly")
			}
			d.scan_bop()
			if in_postamble {
				goto done
			}
		}
	}
done:
	if d.OutMode < the_works {
		if !in_postamble {
			d.skip_pages(true)
		}
		if int64(d.signedquad()) != old_backpointer {
			fmt.Println("backpointer in byte", d.curloc-4, " should be ", old_backpointer, "!")
		}
		d.readPostamble()

	}
}
func pixelround(a int) int {
	x := conv * float32(a)
	return round(x)
}

func (d *Dvitype) outText(c uint8) {
	if textptr == line_length-2 {
		d.flushText()
	}
	textptr++
	textbuf[textptr] = c
}

func (d *Dvitype) flushText() {
	if textptr > 0 {
		if d.OutMode > errors_only {
			fmt.Printf("[%s]\n", string(textbuf[1:textptr+1]))
		}
	}
	textptr = 0
}

func (d *Dvitype) show(pos, a interface{}) {
	d.flushText()
	showing = true
	fmt.Printf("%d: %v", pos, a)
}

func (d *Dvitype) major(pos int, a interface{}) {
	if d.OutMode > errors_only {
		d.show(pos, a)
	}
}
func (d *Dvitype) minor(pos int, a interface{}) {
	if d.OutMode > terse {
		showing = true
		fmt.Printf("%d: %v", pos, a)
	}
}
func (d *Dvitype) _error(cmd int, a interface{}) {
	if !showing {
		d.show(cmd, a)
	} else {
		fmt.Print(" ", a)
	}
}

func (d *Dvitype) specialcases(o eightbits, p, a int) bool {
	var (
		q       int  // parameter of the current command
		badchar bool //  has a non-ASCII character code appeared in this xxx ?
		pure    bool // is the command error-free?
		vvv     int  // v, rounded to the nearest pixel
	)
	pure = true
	switch o {
	// 85:
	case down1, down1 + 1, down1 + 2, down1 + 3:
		//outvmove
		if abs(p) >= 5*fontspace[curfont] {
			vv = pixelround(v + p)
		} else {
			vv = vv + pixelround(p)
		}
		d.major(a, fmt.Sprintf("down%d %d", o-down1+1, p))
		goto movedown
	case y0, y1, y1 + 1, y1 + 2, y1 + 3:
		y = p
		if abs(p) >= 5*fontspace[curfont] {
			vv = pixelround(v + p)
		} else {
			vv = vv + pixelround(p)
		}
		d.major(a, fmt.Sprintf("y%d %d", o-y0, p))
		goto movedown
	case z0, z1, z1 + 1, z1 + 2, z1 + 3:
		z = p
		if abs(p) >= 5*fontspace[curfont] {
			vv = pixelround(v + p)
		} else {
			vv = vv + pixelround(p)
		}
		d.major(a, fmt.Sprintf("z%d %d", o-z0, p))
		goto movedown
	// :85
	// 86:
	case fnt_num_0, fnt_num_0 + 1, fnt_num_0 + 2, 174, 175, 176, 177, 178, 179, 180, 181, 182, 183, 184, 185, 186,
		187, 188, 189, 190, 191, 192, 193, 194, 195, 196, 197, 198, 199, 200, 201, 202,
		203, 204, 205, 206, 207, 208, 209, 210, 211, 212, 213, 214, 215, 216, 217, 218,
		219, 220, 221, 222, 223, 224, 225, 226, 227, 228, 229, 230, 231, 232, 233, 234:
		d.major(a, fmt.Sprintf("fntnum%d", p))
		goto changefont
	case fnt1, fnt1 + 1, fnt1 + 2, fnt1 + 3:
		d.major(a, fmt.Sprintf("fnt%d %d", o-fnt1+1, p))
		goto changefont
	case fnt_def1, fnt_def1 + 1, fnt_def1 + 2, fnt_def1 + 3:
		d.major(a, fmt.Sprintf("fntdef%d %d", o-fnt_def1+1, p))
		d.defineFont(p)
		return pure
	// :86
	case xxx1, xxx1 + 1, xxx1 + 2, xxx1 + 3:
		// 87:
		d.major(a, "xxx '")
		badchar = false
		if p < 0 {
			d._error(a, "string of negative length!")
		}
		for k := 1; k <= p; k++ {
			q = d.getbyte()
			if q < ' ' || q > '~' {
				badchar = true
			}
			if showing {
				fmt.Printf("%c", q)
			}
		}
		if showing {
			fmt.Print("'")
		}
		if badchar {
			d._error(a, "non-ASCII character in xxx command!")
		}
		return pure
		// :87
	case pre:
		d._error(a, "preamble command within a page!")
		return false
	case post, post_post:
		d._error(a, "postamble command within a page!")
		return false
	default:
		d._error(a, fmt.Sprintf("undefined command %d!", o))
		return true
	}
movedown:
	// Finish a command that sets v=v+p, then goto done 92⟩;
	if (v > 0) && (p > 0) {
		if v > infinity-p {
			d._error(a, fmt.Sprintf("arithmetic overflow! parameter changed from %d to %d", p, infinity-v))
			p = infinity - v
		}
	}
	if (v < 0) && (p < 0) {
		if -v > p+infinity {
			d._error(a, fmt.Sprintf("arithmetic overflow! parameter changed from %d to %d", p, (-v)-infinity))
			p = (-v) - infinity
		}
	}
	vvv = pixelround(v + p)
	if abs(vvv-vv) > maxdrift {
		if vvv > vv {
			vv = vvv - maxdrift
		} else {
			vv = vvv + maxdrift
		}
	}

	if showing {
		if d.OutMode > mnemonics_only {
			fmt.Print(" v:=", v)
			if p >= 0 {
				fmt.Print("+")
			}
			fmt.Printf("%d=%d, vv:=%d", p, v+p, vv)

		}
	}

	v = v + p

	if abs(v) > maxvsofar {
		if abs(v) > maxv+99 {
			d._error(a, fmt.Sprintf("warning: |v|>%d!", maxv))
			maxv = abs(v)
		}
		maxvsofar = abs(v)
	}
	return pure
	// :92
changefont:
	// ⟨ Finish a command that changes the current font, then goto done 94 ⟩;
	font_num[nf] = p
	curfont = 0
	for font_num[curfont] != p {
		curfont++
	}
	if curfont == nf {
		curfont = invalid_font
		d._error(a, fmt.Sprintf("invalid font selection: font %d was never defined!", p))
	}

	if showing {
		if d.OutMode > mnemonics_only {
			fmt.Print(" current font is  ")
			d.printFont(curfont)
		}
	}
	return pure
	// :94
	return true
}

func rulepixels(x int) int {
	var n int
	n = int(conv * float32(x))
	if float32(n) < conv*float32(x) {
		return n + 1
	} else {
		return n
	}
}

// 79 doPage()
func (d *Dvitype) doPage() bool {
	var o eightbits        //  operation code of the current command
	var p, q int           // parameters of the current command
	var a int              // byte number of the current command
	var hhh int            // h, rounded to the nearest pixel
	curfont = invalid_font // set current font undefined
	s = 0
	h = 0
	v = 0
	x = 0
	y = 0
	z = 0
	hh = 0
	vv = 0 // initialize the state variables
	for {
		//  Translate the next command in the DVI file; goto 9999 with do page = true if it was eop ; goto 9998 if premature termination is needed 80:
		a = int(d.curloc)
		showing = false
		o = eightbits(d.getbyte())
		p = d.firstpar(o)

		// if eof (dvi file ) then bad dvi ( "the file ended prematurely")

		// Start translation of command o and goto the appropriate label to finish the job 81:
		if o < set_char_0+128 {
			//  Translate a set char command 88:
			if o > ' ' && o <= '~' {
				d.outText(uint8(p))
				d.minor(a, fmt.Sprintf("setchar%d", p))
			} else {
				d.major(a, fmt.Sprintf("setchar%d", p))
			}
			goto finset
			// :88
		} else {
			switch o {
			case set1, set1 + 1, set1 + 2, set1 + 3:
				d.major(a, fmt.Sprintf("set%d %d", o-set1+1, p))
				goto finset
			case put1, put1 + 1, put1 + 2, put1 + 3:
				d.major(a, fmt.Sprintf("put%d %d", o-set1+1, p))
				goto finset
			case set_rule:
				d.major(a, "setrule")
				goto finrule
			case put_rule:
				d.major(a, "putrule")
				goto finrule
			// 83:
			case nop:
				d.minor(a, "nop")
				goto done
			case bop:
				d._error(a, "bop occurred before eop!")
				goto l9998
			case eop:
				d.major(a, "eop")

				if s != 0 {
					d._error(a, fmt.Sprintf("stack not empty at end of page (level %d)!", s))
				}
				fmt.Println()
				return true
			case push:
				d.major(a, "push")
				if s == maxhsofar {
					maxssofar = s + 1
					if s == maxs {
						d._error(a, "deeper than claimed in postamble!")
					}
					if s == stack_size {
						d._error(a, fmt.Sprintf("DVItype capacity exceeded (stack size= %d) ", stack_size))
						goto l9998
					}
				}
				hstack[s] = h
				vstack[s] = v
				wstack[s] = w
				xstack[s] = x
				ystack[s] = y
				zstack[s] = z
				hhstack[s] = hh
				vvstack[s] = vv
				s++
				ss = s - 1
				goto showstate
			case pop:
				d.major(a, "pop")
				if s == 0 {
					d._error(a, "(illegal at level zero)! ")
				} else {
					s--
					hh = hhstack[s]
					vv = vvstack[s]
					h = hstack[s]
					v = vstack[s]
					w = wstack[s]
					x = xstack[s]
					y = ystack[s]
					z = zstack[s]
				}

				ss = s
				goto showstate
				// :83
				// 84:
			case right1, right1 + 1, right1 + 2, right1 + 3:
				// outspace
				if (p >= fontspace[curfont]) || (p <= -4*fontspace[curfont]) {
					d.outText(' ')
					hh = pixelround(h + p)
				} else {
					hh = hh + pixelround(p)
				}
				d.minor(a, fmt.Sprintf("right%d %d", o-right1+1, p))
				q = p
				goto moveright
			case w0, w1, w1 + 1, w1 + 2, w1 + 3:
				w = p
				// outspace
				if (p >= fontspace[curfont]) || (p <= -4*fontspace[curfont]) {
					d.outText(' ')
					hh = pixelround(h + p)
				} else {
					hh = hh + pixelround(p)
				}
				d.minor(a, fmt.Sprintf("w%d %d", int(o)-w0, p))
				q = p
				goto moveright
			case x0, x1, x1 + 1, x1 + 2, x1 + 3:
				x = p
				// outspace
				if (p >= fontspace[curfont]) || (p <= -4*fontspace[curfont]) {
					d.outText(' ')
					hh = pixelround(h + p)
				} else {
					hh = hh + pixelround(p)
				}
				d.minor(a, fmt.Sprintf("x%d %d", int(o)-x0, p))
				q = p
				goto moveright
				// :84
			default:
				if d.specialcases(o, p, a) {
					goto done
				} else {
					goto l9998
				}
			}

		}
		//:81

	finset: //  Finish a command that either sets or puts a character, then goto move right or done 89 ⟩
		if p < 0 {
			p = 255 - ((-1 - p) % 256)
		} else if p >= 256 {
			p = p % 256 // width computation for oriental fonts
		}
		if (p < fontbc[curfont]) || p > fontec[curfont] {
			q = invalid_width
		} else {
			q = width[widthbase[curfont]+p]
		}
		if q == invalid_width {
			d._error(a, fmt.Sprintf("character %d invalid in font", p))
			d.printFont(curfont)
			if curfont != invalid_font {
				fmt.Print("!") // the invalid font has ‘!’ in its name
			}
		}
		if o >= put1 {
			goto done
		}
		if q == invalid_width {
			q = 0
		} else {
			hh = hh + pixelwidth[widthbase[curfont]+p]
		}
		goto moveright
		// :89
	finrule: // Finish a command that either sets or puts a rule, then goto move right or done 90 ⟩
		q = d.signedquad()
		if showing {
			fmt.Printf(" height %d, width %d", p, q)
			if d.OutMode > mnemonics_only {
				if p <= 0 || q <= 0 {
					fmt.Print(" (invisible) ")
				} else {
					fmt.Printf(" (%dx%d pixels)", rulepixels(p), rulepixels(q))
				}
			}
		}
		if o == put_rule {
			goto done
		}
		if showing {
			if d.OutMode > mnemonics_only {
				fmt.Println()
			}
		}
		hh = hh + rulepixels(q)
		goto moveright
		// :90
	moveright: // Finish a command that sets h = h + q, then goto done 91
		if h > 0 && q > 0 {
			if h > infinity-q {
				d._error(a, fmt.Sprintf("arithmetic overflow! parameter changed from  %d to %d", q, infinity-h))
				q = infinity - h
			}
		}
		if h < 0 && q < 0 {
			if -h > q+infinity {
				d._error(a, fmt.Sprintf("arithmetic overflow! parameter changed from  %d to %d", q, (-h)-infinity))
				q = (-h) - infinity
			}
		}
		hhh = pixelround(h + q)
		if abs(hhh-hh) > maxdrift {
			if hhh > hh {
				hh = hhh - maxdrift
			} else {
				hh = hhh + maxdrift
			}
		}
		if showing {
			if d.OutMode > mnemonics_only {
				fmt.Printf(" h:=%d", h)
				if q >= 0 {
					fmt.Print("+")
				}
				fmt.Printf("%d=%d, hh:=%d", q, h+q, hh)
			}
		}
		h = h + q
		if abs(h) > maxhsofar {
			if abs(h) > maxh+99 {
				d._error(a, fmt.Sprintf("warning: |h|>%d!", maxh))
				maxh = abs(h)
			}
			maxhsofar = abs(h)
		}
		goto done
		// :91
	showstate: // Show the values of ss, h, v, w, x, y, z, hh, and vv then goto done 93⟩
		// 	if showing then
		if showing {
			if d.OutMode > mnemonics_only {
				fmt.Println()
				fmt.Printf("level %d:(h=%d,v=%d,w=%d,x=%d,y=%d,z=%d,hh=%d,vv=%d)", ss, h, v, w, x, y, z, hh, vv)
			}
		}
		goto done
		// :93
	done:
		if showing {
			fmt.Println()
		}
		//:80
	}
l9998:
	fmt.Println("!")
	return false
}

// :79

// 95:
func (d *Dvitype) skip_pages(bop_seen bool) {
	var (
		_p int       // a parameter
		k  eightbits // command code
	)
	showing = false
	for {
		if !bop_seen {
			d.scan_bop()
			if in_postamble {
				return
			}
			if !started {
				if start_match() {
					started = true
					return
				}
			}
		}
		// skip until finding eop 96
		for {
			// if eof -> error
			k = eightbits(d.getbyte())
			_p = d.firstpar(k)
			switch k {
			case set_rule, put_rule:
				d.signedquad() // ignore
			case fnt_def1, fnt_def1 + 1, fnt_def1 + 2, fnt_def1 + 3:
				d.defineFont(_p)
				fmt.Println()
			case xxx1, xxx1 + 1, xxx1 + 2, xxx1 + 3:
				d.getbyte() // ignore
				_p--
			case bop, pre, post, post_post, undef1, undef2, undef3, undef4, undef5, undef6:
				bad_dvi(fmt.Sprintf("illegal command at byte %d", d.curloc))
			}
			if k == eop {
				break
			}
		}
		// :96
		bop_seen = false
	}

}

// :95

// 99
func (d *Dvitype) scan_bop() {
	var k eightbits
	for {
		//  if eof (dvi file ) then bad dvi ("the file ended prematurely");
		k = eightbits(d.getbyte())
		if k >= fnt_def1 && k < fnt_def1+4 {
			d.defineFont(d.firstpar(k))
			k = nop
		}
		if k != nop {
			break
		}
	}
	if k == post {
		in_postamble = true
	} else {
		if k != bop {
			bad_dvi(fmt.Sprintf("byte %d is not bop (2)", d.curloc))
		}
		new_backpointer = d.curloc - 1
		pagecount++
		for k := 0; k < 10; k++ {
			count[k] = d.signedquad()
		}
		if x := int64(d.signedquad()); x != old_backpointer {
			fmt.Println("backpointer in byte", d.curloc-4, "should be", old_backpointer, ", but is", x, "!")
		}
		old_backpointer = new_backpointer
	}
}

// does count match the starting spec?
func start_match() bool {
	var match bool // does everything match so far?
	match = true
	for k := 0; k <= int(start_vals); k++ {
		if start_there[k] && (start_count[k] != count[k]) {
			match = false
		}
	}
	return match
}
