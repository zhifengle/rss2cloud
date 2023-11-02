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

	cmd := exec.Command("cmd", "/c", "start", imageName)
	err = cmd.Start()
	if err != nil {
		return err
	}
	return nil
}
