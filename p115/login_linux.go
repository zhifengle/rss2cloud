package p115

import (
	"errors"
	"os"
	"os/exec"
)

func getValidOpenCommand() string {
	commnads := []string{"xdg-open", "gnome-open", "kde-open"}
	for _, c := range commnads {
		_, err := exec.LookPath(c)
		if err == nil {
			return c
		}
	}
	return ""
}

func DisplayQrcode(img []byte) error {
	imageName := "qrcode115.png"
	// save image
	err := os.WriteFile(imageName, img, 0644)
	if err != nil {
		return err
	}
	openCommand := getValidOpenCommand()
	if openCommand == "" {
		return errors.New("no open command found")
	}
	cmd := exec.Command(openCommand, imageName)
	err = cmd.Start()
	if err != nil {
		return err
	}
	return nil
}

func DisposeQrcode() {
	_ = os.Remove("qrcode115.png")
}
