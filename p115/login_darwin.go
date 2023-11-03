package p115

import (
	"os"
	"os/exec"
)

func DisplayQrcode(img []byte) error {
	imageName := "qrcode115.png"
	// save image
	err := os.WriteFile(imageName, img, 0644)
	if err != nil {
		return err
	}

	cmd := exec.Command("open", imageName)
	err = cmd.Start()
	if err != nil {
		return err
	}
	return nil
}

func DisposeQrcode() {
	_ = os.Remove("qrcode115.png")
}
