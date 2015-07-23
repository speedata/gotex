package dvitype

import (
	"bytes"
	"testing"
)

var a = []byte{1, 2, 3, 4}
var b = []byte{129, 2, 3, 4}

func TestGetByte(t *testing.T) {
	bra := bytes.NewReader(a)
	dta := New(bra)

	brb := bytes.NewReader(b)
	dtb := New(brb)

	res := dta.getbyte()
	exp := 1
	if res != exp {
		t.Errorf("Should be %d, but is %d", exp, res)
	}

	res = dtb.getbyte()
	exp = 129

	if res != exp {
		t.Errorf("Should be %d, but is %d", exp, res)
	}
}

func TestGetTwoBytes(t *testing.T) {
	bra := bytes.NewReader(a)
	dt := New(bra)
	b := dt.gettwobytes()
	exp := 258
	if b != exp {
		t.Errorf("Should be %d, but is %d\n", exp, b)
	}
}

func TestGetSignedByte(t *testing.T) {
	brb := bytes.NewReader(b)
	dt := New(brb)
	b := dt.signedbyte()
	exp := -127
	if b != exp {
		t.Errorf("Should be %d, but is %d\n", exp, b)
	}
}

func TestGetSignedPair(t *testing.T) {
	brb := bytes.NewReader(b)
	dt := New(brb)
	b := dt.signedpair()
	exp := -32510
	if b != exp {
		t.Errorf("Should be %d, but is %d\n", exp, b)
	}
}
func TestGetSignedTrio(t *testing.T) {
	brb := bytes.NewReader(b)
	dt := New(brb)
	b := dt.signedtrio()
	exp := -8322557
	if b != exp {
		t.Errorf("Should be %d, but is %d\n", exp, b)
	}
}

func TestRound(t *testing.T) {
	var a float32
	var exp int

	a = 12.4
	exp = 12
	if res := round(a); res != exp {
		t.Errorf("Should be %d but is %d\n", exp, res)
	}
	a = -12.4
	exp = -12
	if res := round(a); res != exp {
		t.Errorf("Should be %d but is %d\n", exp, res)
	}
}
