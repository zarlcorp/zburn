package tui

import (
	"bytes"
	_ "embed"
	"image"
	_ "image/png"
)

//go:embed avatar.png
var avatarData []byte

func avatarImage() image.Image {
	img, _, _ := image.Decode(bytes.NewReader(avatarData))
	return img
}
