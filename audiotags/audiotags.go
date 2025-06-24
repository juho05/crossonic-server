/***************************************************************************
   copyright            : (C) 2014 by Nick Sellen
   email                : code@nicksellen.co.uk
***************************************************************************/

/***************************************************************************
 *   This library is free software; you can redistribute it and/or modify  *
 *   it  under the terms of the GNU Lesser General Public License version  *
 *   2.1 as published by the Free Software Foundation.                     *
 *                                                                         *
 *   This library is distributed in the hope that it will be useful, but   *
 *   WITHOUT ANY WARRANTY; without even the implied warranty of            *
 *   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU     *
 *   Lesser General Public License for more details.                       *
 *                                                                         *
 *   You should have received a copy of the GNU Lesser General Public      *
 *   License along with this library; if not, write to the Free Software   *
 *   Foundation, Inc., 59 Temple Place, Suite 330, Boston, MA  02111-1307  *
 *   USA                                                                   *
 ***************************************************************************/

package audiotags

/*
#cgo pkg-config: taglib
#cgo LDFLAGS: -lstdc++
#include "audiotags.h"
#include <stdlib.h>
*/
import "C"
import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"strings"
	"sync"
	"sync/atomic"
	"unsafe"
)

const (
	JPEG = iota
	PNG  = iota
)

type File C.TagLib_FileRefRef

type AudioProperties struct {
	Length, LengthMs, Bitrate, Samplerate, Channels int
}

func (f *File) HasMedia() bool {
	return !f.ReadAudioProperties().isEmpty()
}

func (props *AudioProperties) isEmpty() bool {
	return props.Bitrate == 0 && props.LengthMs == 0 && props.Length == 0 && props.Samplerate == 0 && props.Channels == 0
}

func Open(filename string) (*File, error) {
	fp := C.CString(filename)
	defer C.free(unsafe.Pointer(fp))
	f := C.audiotags_file_new(fp)
	if f == nil {
		return nil, fmt.Errorf("cannot process %s", filename)
	}
	return (*File)(f), nil
}

func (f *File) Close() {
	C.audiotags_file_close((*C.TagLib_FileRefRef)(f))
}

func (f *File) ReadTags() KeyMap {
	id := mapsNextID.Add(1)
	defer maps.Delete(id)

	m := KeyMap{}
	maps.Store(id, m)
	C.audiotags_file_properties((*C.TagLib_FileRefRef)(f), C.int(id))
	return m
}

func (f *File) WriteTags(tagMap KeyMap) bool {
	if len(tagMap) == 0 {
		return bool(C.audiotags_clear_properties((*C.TagLib_FileRefRef)(f)))
	}

	tagFields := make([]*C.char, len(tagMap))
	tagValues := make([]*C.char, len(tagMap))
	var i int
	for field, values := range tagMap {
		fieldC := C.CString(field)
		tagFields[i] = fieldC

		valueC := C.CString(strings.Join(values, "\003"))
		tagValues[i] = valueC
		i++
	}
	defer func() {
		for i := 0; i < len(tagMap); i++ {
			C.free(unsafe.Pointer(tagFields[i]))
			C.free(unsafe.Pointer(tagValues[i]))
		}
	}()

	return bool(C.audiotags_write_properties((*C.TagLib_FileRefRef)(f), C.uint(len(tagMap)), &tagFields[0], &tagValues[0]))
}

func (f *File) ReadAudioProperties() *AudioProperties {
	ap := C.audiotags_file_audioproperties((*C.TagLib_FileRefRef)(f))
	if ap == nil {
		return nil
	}
	p := AudioProperties{}
	p.Length = int(C.audiotags_audioproperties_length(ap))
	p.LengthMs = int(C.audiotags_audioproperties_length_ms(ap))
	p.Bitrate = int(C.audiotags_audioproperties_bitrate(ap))
	p.Samplerate = int(C.audiotags_audioproperties_samplerate(ap))
	p.Channels = int(C.audiotags_audioproperties_channels(ap))
	return &p
}

func (f *File) ReadImage() (image.Image, error) {
	id := mapsNextID.Add(1)
	defer maps.Delete(id)

	C.audiotags_read_picture((*C.TagLib_FileRefRef)(f), C.int(id))
	v, ok := maps.Load(id)
	if !ok {
		return nil, nil
	}
	img, _, err := image.Decode(v.(*bytes.Reader))
	return img, err
}

func (f *File) WriteImage(img image.Image, format int) error {
	i, ok := img.(*image.NRGBA)
	if !ok {
		return fmt.Errorf("can't get convert image")
	}

	buff := bytes.NewBuffer([]byte{})
	switch format {
	case JPEG:
		if err := jpeg.Encode(buff, i.SubImage(i.Rect), &jpeg.Options{Quality: 65}); err != nil {
			return err
		}
	case PNG:
		if err := png.Encode(buff, i.SubImage(i.Rect)); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsuppported image format")
	}

	data := buff.Bytes()
	if len(data) == 0 {
		return fmt.Errorf("can't write empty image")
	}

	if !f.WriteImageData(data, img.Bounds().Size().X, img.Bounds().Size().Y, format) {
		return fmt.Errorf("can't write image")
	}

	return nil
}

func (f *File) WriteImageData(data []byte, fmt, w, h int) bool {
	if len(data) == 0 {
		return false
	}

	return bool(C.audiotags_write_picture((*C.TagLib_FileRefRef)(f), (*C.char)(unsafe.Pointer(&data[0])), C.uint(len(data)), C.int(w), C.int(h), C.int(fmt)))
}

func (f *File) RemovePictures() bool {
	return bool(C.audiotags_remove_pictures((*C.TagLib_FileRefRef)(f)))
}

var maps sync.Map
var mapsNextID atomic.Uint64

type KeyMap = map[string][]string

//export goTagPut
func goTagPut(id C.int, key *C.char, val *C.char) {
	r, _ := maps.Load(uint64(id))
	m := r.(KeyMap)
	k := strings.ToLower(C.GoString(key))
	v := C.GoString(val)
	m[k] = append(m[k], v)
}

//export goPutImage
func goPutImage(id C.int, data *C.char, size C.int) {
	maps.Store(uint64(id), bytes.NewReader(C.GoBytes(unsafe.Pointer(data), size)))
}
