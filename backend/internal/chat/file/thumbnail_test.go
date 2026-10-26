package file

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func encodedImage(t *testing.T, w, h int, format string) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for x := 0; x < w; x++ {
		for y := 0; y < h; y++ {
			img.Set(x, y, color.RGBA{R: uint8(x % 256), G: uint8(y % 256), B: 120, A: 255})
		}
	}
	var buf bytes.Buffer
	switch format {
	case "png":
		require.NoError(t, png.Encode(&buf, img))
	case "jpeg":
		require.NoError(t, jpeg.Encode(&buf, img, nil))
	case "gif":
		pal := image.NewPaletted(img.Bounds(), color.Palette{color.Black, color.White})
		require.NoError(t, gif.Encode(&buf, pal, nil))
	}
	return buf.Bytes()
}

func TestImagingGenerator_CanGenerate(t *testing.T) {
	t.Parallel()
	g := NewImagingGenerator()

	for _, mime := range []string{"image/jpeg", "image/png", "image/gif", "image/webp", "IMAGE/PNG", "Image/Jpeg"} {
		assert.True(t, g.CanGenerate(mime), mime)
	}
	// SVG is an allowed upload type but cannot be rasterised by imaging, and
	// documents have no preview: Upload relies on this to skip the thumbnail.
	for _, mime := range []string{"image/svg+xml", "application/pdf", "text/plain", "", "image/"} {
		assert.False(t, g.CanGenerate(mime), mime)
	}
}

func TestImagingGenerator_Generate(t *testing.T) {
	t.Parallel()
	g := NewImagingGenerator()

	t.Run("landscape png is fitted into 200x200 and re-encoded as jpeg", func(t *testing.T) {
		t.Parallel()
		out, err := g.Generate(context.Background(), bytes.NewReader(encodedImage(t, 800, 400, "png")), "image/png")
		require.NoError(t, err)

		data, err := io.ReadAll(out)
		require.NoError(t, err)
		cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
		require.NoError(t, err)
		assert.Equal(t, "jpeg", format)
		assert.Equal(t, 200, cfg.Width)
		assert.Equal(t, 100, cfg.Height, "aspect ratio must be kept")
	})

	t.Run("portrait jpeg is bounded by the height", func(t *testing.T) {
		t.Parallel()
		out, err := g.Generate(context.Background(), bytes.NewReader(encodedImage(t, 300, 900, "jpeg")), "image/jpeg")
		require.NoError(t, err)

		data, _ := io.ReadAll(out)
		cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
		require.NoError(t, err)
		assert.Equal(t, 200, cfg.Height)
		assert.Equal(t, 66, cfg.Width)
	})

	t.Run("small image is not enlarged", func(t *testing.T) {
		t.Parallel()
		out, err := g.Generate(context.Background(), bytes.NewReader(encodedImage(t, 50, 40, "gif")), "image/gif")
		require.NoError(t, err)

		data, _ := io.ReadAll(out)
		cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
		require.NoError(t, err)
		assert.Equal(t, 50, cfg.Width)
		assert.Equal(t, 40, cfg.Height)
	})

	t.Run("undecodable bytes are an error", func(t *testing.T) {
		t.Parallel()
		out, err := g.Generate(context.Background(), bytes.NewReader([]byte("definitely not an image")), "image/png")
		require.Error(t, err)
		assert.Nil(t, out)
	})

	t.Run("empty reader is an error", func(t *testing.T) {
		t.Parallel()
		_, err := g.Generate(context.Background(), bytes.NewReader(nil), "image/png")
		require.Error(t, err)
	})
}

func TestNoOpScanner_AcceptsEverything(t *testing.T) {
	t.Parallel()
	var s FileScanner = &NoOpScanner{}
	require.NoError(t, s.Scan(context.Background(), bytes.NewReader([]byte("X5O!P%@AP")), "eicar.txt"))
}
