package preview

import (
	"bytes"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	"github.com/qeesung/image2ascii/convert"
)

// renderImage converts a raster image to colored ASCII art via image2ascii.
func renderImage(data []byte, name string, width, height int) (result string, err error) {
	// Recover from any panics in the resize/conversion step.
	defer func() {
		if r := recover(); r != nil {
			result = ""
			err = fmt.Errorf("image decode failed: %v", r)
		}
	}()

	// Decode the image from raw bytes using the standard library.
	// This returns a proper error instead of calling log.Fatal like
	// image2ascii's ImageFile2ASCIIString does.
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("image decode: %w", err)
	}

	opts := convert.DefaultOptions
	opts.Colored = true
	opts.FitScreen = false
	if width > 0 {
		opts.FixedWidth = width
	}

	if height > 0 {
		opts.FixedHeight = height
	}

	c := convert.NewImageConverter()
	return c.Image2ASCIIString(img, &opts), nil
}
