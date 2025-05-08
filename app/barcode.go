package app

import (
	"bytes"
	"github.com/webitel/storage/model"
	"image"
	"image/color"
	"image/draw"
	"image/png"

	"github.com/makiuchi-d/gozxing"
	"github.com/makiuchi-d/gozxing/oned"

	"golang.org/x/image/font"
	"golang.org/x/image/font/inconsolata"
	"golang.org/x/image/math/fixed"
)

func (a *App) Barcode(text, altText string, width, height int) (model.AppError, *bytes.Buffer) {
	writer := oned.NewCode128Writer()

	h := height

	if altText != "" {
		h += 25
	}

	first, err := writer.Encode(text, gozxing.BarcodeFormat_CODE_128, width, height, nil)
	if err != nil {
		return model.NewInternalError("app.barcode.generate", err.Error()), nil
	}

	b := first.Bounds()
	image3 := image.NewRGBA(image.Rect(0, 0, width, h))
	draw.Draw(image3, b, first, image.ZP, draw.Src)
	if altText != "" {
		addLabel(image3, width, h-7, altText)
	}

	buf := new(bytes.Buffer)
	err = png.Encode(buf, image3)
	if err != nil {
		return model.NewInternalError("app.barcode.generate", err.Error()), nil
	}

	return nil, buf
}

func addLabel(img *image.RGBA, width, y int, label string) {
	col := color.RGBA{A: 255}

	dd := (width / 2) - ((len(label) / 2) * 8)

	point := fixed.Point26_6{X: fixed.I(dd), Y: fixed.I(y)}

	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(col),
		Face: inconsolata.Bold8x16,
		Dot:  point,
	}
	d.DrawString(label)
}
