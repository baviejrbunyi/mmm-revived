package patchserver

import "errors"

type NativeCatalog struct {
	*NativeHandler
}

func OpenNativeCatalog(directory string) (*NativeCatalog, error) {
	handler, err := OpenNative(directory)
	if err != nil {
		return nil, err
	}
	if handler.root == nil {
		handler.Close()
		return nil, errors.New("only retail engine and resource catalogs are supported")
	}
	return &NativeCatalog{handler}, nil
}
