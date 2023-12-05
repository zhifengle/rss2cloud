package p115

import "errors"

func DisplayQrcode(img []byte) error {
	return errors.New("Qrcode login is not support on Linux")
}

func DisposeQrcode() {
}
