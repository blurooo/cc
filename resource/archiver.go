package resource

import (
	"errors"
	"reflect"

	"github.com/mholt/archiver/v3"
)

var errorUnsupported = errors.New("unsupported archiver")

// Archiver 参数
type Archiver struct {
	RetainTopFolder bool
}

// UnArchiver 解包
func (a *Archiver) UnArchiver(filename, toPath string) error {
	iua, err := archiver.ByExtension(filename)
	if err != nil {
		return errorUnsupported
	}
	a.patchArchiverSettings(iua)
	u, ok := iua.(archiver.Unarchiver)
	if ok {
		return u.Unarchive(filename, toPath)
	}
	return errorUnsupported
}

func (a *Archiver) patchArchiverSettings(iua interface{}) {
	rua := reflect.ValueOf(iua).Elem()
	if a.RetainTopFolder {
		return
	}
	v := rua.FieldByName("StripComponents")
	if v.CanSet() {
		v.SetInt(1)
	}
}
