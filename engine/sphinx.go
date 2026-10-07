package main

/*
#cgo CFLAGS: -I${SRCDIR}/include -I${SRCDIR}/include/pocketsphinx
#cgo LDFLAGS: -l:libpocketsphinx.so.3 -l:libsphinxbase.so.3
#include "sphinx_bridge.h"
#include <stdlib.h>
*/
import "C"

import (
	"errors"
	"unsafe"
)

const (
	sphinxHMM  = "/usr/share/pocketsphinx/model/en-us/en-us"
	sphinxLM   = "/usr/share/pocketsphinx/model/en-us/en-us.lm.bin"
	sphinxDict = "/usr/share/pocketsphinx/model/en-us/cmudict-en-us.dict"
)

// recognizer is the on-device PocketSphinx decoder (US English).
// It reads 16 kHz mono 16-bit audio and returns words as they are recognized.
type recognizer struct {
	ps *C.ps_decoder_t
}

func openRecognizer() (*recognizer, error) {
	hmm := C.CString(sphinxHMM)
	lm := C.CString(sphinxLM)
	dict := C.CString(sphinxDict)
	defer C.free(unsafe.Pointer(hmm))
	defer C.free(unsafe.Pointer(lm))
	defer C.free(unsafe.Pointer(dict))

	ps := C.meet_ps_open(hmm, lm, dict)
	if ps == nil {
		return nil, errors.New("PocketSphinx could not load the US English model in /usr/share/pocketsphinx/model/en-us")
	}
	return &recognizer{ps: ps}, nil
}

func (r *recognizer) start() error {
	if C.meet_ps_start(r.ps) < 0 {
		return errors.New("PocketSphinx could not start a phrase")
	}
	return nil
}

func (r *recognizer) feed(pcm []byte) {
	if len(pcm) < 2 {
		return
	}
	n := len(pcm) / 2
	C.meet_ps_feed(r.ps, (*C.int16_t)(unsafe.Pointer(&pcm[0])), C.size_t(n))
}

func (r *recognizer) inSpeech() bool {
	return C.meet_ps_in_speech(r.ps) == 1
}

func (r *recognizer) hypothesis() string {
	text := C.meet_ps_hyp(r.ps)
	if text == nil {
		return ""
	}
	return C.GoString(text)
}

func (r *recognizer) end() string {
	C.meet_ps_end(r.ps)
	return r.hypothesis()
}

func (r *recognizer) close() {
	if r.ps != nil {
		C.meet_ps_close(r.ps)
		r.ps = nil
	}
}
