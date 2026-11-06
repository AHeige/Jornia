// First-party format conversion for Wisp's own artwork. No image or UI framework.
package main

import (
	"encoding/binary"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"strconv"
)

func main() {
	if len(os.Args) != 5 {
		panic("usage: addon-art input output width height")
	}
	w, err := strconv.Atoi(os.Args[3])
	if err != nil {
		panic(err)
	}
	h, err := strconv.Atoi(os.Args[4])
	if err != nil {
		panic(err)
	}
	if w < 1 || h < 1 || w > 2048 || h > 2048 || w&(w-1) != 0 || h&(h-1) != 0 {
		panic("dimensions must be powers of two up to 2048")
	}
	in, err := os.Open(os.Args[1])
	if err != nil {
		panic(err)
	}
	img, _, err := image.Decode(in)
	in.Close()
	if err != nil {
		panic(err)
	}
	b := img.Bounds()
	out := make([]byte, 18+w*h*4)
	out[2] = 2
	binary.LittleEndian.PutUint16(out[12:14], uint16(w))
	binary.LittleEndian.PutUint16(out[14:16], uint16(h))
	out[16] = 32
	out[17] = 0x28
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			// Bilinear interpolation in premultiplied RGBA avoids dark transparent edges.
			fx := (float64(x)+.5)*float64(b.Dx())/float64(w) - .5
			fy := (float64(y)+.5)*float64(b.Dy())/float64(h) - .5
			if fx < 0 {
				fx = 0
			}
			if fy < 0 {
				fy = 0
			}
			x0, y0 := int(fx), int(fy)
			x1, y1 := min(x0+1, b.Dx()-1), min(y0+1, b.Dy()-1)
			dx, dy := fx-float64(x0), fy-float64(y0)
			var rgba [4]float64
			for i, p := range [4]image.Point{{x0, y0}, {x1, y0}, {x0, y1}, {x1, y1}} {
				r, g, bl, a := img.At(b.Min.X+p.X, b.Min.Y+p.Y).RGBA()
				weight := []float64{(1 - dx) * (1 - dy), dx * (1 - dy), (1 - dx) * dy, dx * dy}[i]
				for j, v := range []uint32{r, g, bl, a} {
					rgba[j] += float64(v) * weight
				}
			}
			index := 18 + (y*w+x)*4
			a := rgba[3]
			out[index+3] = byte(a/257 + .5)
			if a > 0 {
				out[index] = byte(min(255, rgba[2]/a*255+.5))
				out[index+1] = byte(min(255, rgba[1]/a*255+.5))
				out[index+2] = byte(min(255, rgba[0]/a*255+.5))
			}
		}
	}
	if err = os.WriteFile(os.Args[2], out, 0644); err != nil {
		panic(err)
	}
	fmt.Printf("Wrote %dx%d 32-bit TGA: %s\n", w, h, os.Args[2])
}
