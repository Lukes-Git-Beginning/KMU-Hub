package file

import (
	"bytes"
	"context"
	"image"
	"image/jpeg"
	"io"
	"strings"

	"github.com/disintegration/imaging"
)

// ThumbnailGenerator generates thumbnails for supported file types
type ThumbnailGenerator interface {
	Generate(ctx context.Context, reader io.Reader, mimeType string) (io.Reader, error)
	CanGenerate(mimeType string) bool
}

// ImagingGenerator generates thumbnails using the imaging library
type ImagingGenerator struct {
	maxWidth  int
	maxHeight int
}

// NewImagingGenerator creates a new thumbnail generator
func NewImagingGenerator() *ImagingGenerator {
	return &ImagingGenerator{
		maxWidth:  200,
		maxHeight: 200,
	}
}

var supportedImageTypes = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/gif":  true,
	"image/webp": true,
}

// CanGenerate returns true if thumbnails can be generated for this MIME type
func (g *ImagingGenerator) CanGenerate(mimeType string) bool {
	return supportedImageTypes[strings.ToLower(mimeType)]
}

// Generate creates a thumbnail from the image reader
func (g *ImagingGenerator) Generate(_ context.Context, reader io.Reader, _ string) (io.Reader, error) {
	img, _, err := image.Decode(reader)
	if err != nil {
		return nil, err
	}

	thumb := imaging.Fit(img, g.maxWidth, g.maxHeight, imaging.Lanczos)

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, thumb, &jpeg.Options{Quality: 80}); err != nil {
		return nil, err
	}

	return &buf, nil
}
