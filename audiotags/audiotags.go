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
	"errors"
	"image"
	"strings"
	"sync"
	"sync/atomic"
	"unsafe"
)

var ErrNoMetadata = errors.New("no metadata")

type AudioProperties struct {
	LengthMs, BitRate, SampleRate, Channels int
}

func (props *AudioProperties) IsEmpty() bool {
	if props == nil {
		return true
	}
	return props.BitRate == 0 && props.LengthMs == 0 && props.SampleRate == 0 && props.Channels == 0
}

func Read(path string, checkHasImage bool) (tags KeyMap, props *AudioProperties, hasImage bool, err error) {
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	id := mapsNextID.Add(1)
	defer maps.Delete(id)

	maps.Store(id, tags)
	checkImage := 0
	if checkHasImage {
		checkImage = 1
	}
	cMetadata := C.read(cPath, C.int(checkImage))
	if cMetadata == nil {
		return nil, nil, false, ErrNoMetadata
	}
	defer C.free_metadata(cMetadata)

	props = &AudioProperties{
		LengthMs:   int(cMetadata.lengthMs),
		BitRate:    int(cMetadata.bitRate),
		SampleRate: int(cMetadata.sampleRate),
		Channels:   int(cMetadata.channels),
	}

	hasImageInt := cMetadata.hasImage
	if hasImageInt > 0 {
		hasImage = true
	}

	tags = make(KeyMap)

	ctags := cMetadata.tags.tags
	tagCount := cMetadata.tags.size
	tagSlice := unsafe.Slice(ctags, tagCount)

	for i := 0; i < int(tagCount); i++ {
		key := strings.ToUpper(C.GoString(C.KeyValue(tagSlice[i]).key))
		value := C.GoString(C.KeyValue(tagSlice[i]).value)
		tags[key] = append(tags[key], value)
	}

	return tags, props, hasImage, nil
}

func ReadImage(path string) (image.Image, error) {
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	id := mapsNextID.Add(1)
	defer maps.Delete(id)

	C.read_picture(cPath, C.int(id))
	v, ok := maps.Load(id)
	if !ok {
		return nil, nil
	}
	img, _, err := image.Decode(v.(*bytes.Reader))
	return img, err
}

func WriteTag(path string, key string, value string) bool {
	cPath := C.CString(path)
	cKey := C.CString(strings.ToUpper(key))
	cValue := C.CString(value)
	defer C.free(unsafe.Pointer(cPath))
	defer C.free(unsafe.Pointer(cKey))
	defer C.free(unsafe.Pointer(cValue))

	success := int(C.write_tag(cPath, cKey, cValue))
	if success == 0 {
		return false
	}
	return true
}

func RemoveCrossonicTag(path string, instanceID string) bool {
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	cInstanceID := C.CString(strings.ToUpper(instanceID))
	defer C.free(unsafe.Pointer(cInstanceID))

	success := int(C.remove_crossonic_id(cPath, cInstanceID))
	if success == 0 {
		return false
	}
	return true
}

var maps sync.Map
var mapsNextID atomic.Uint64

type KeyMap = map[string][]string

//export goPutImage
func goPutImage(id C.int, data *C.char, size C.int) {
	maps.Store(uint64(id), bytes.NewReader(C.GoBytes(unsafe.Pointer(data), size)))
}
