package companion

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	lua "github.com/yuin/gopher-lua"
)

func TestLuaQRInteroperatesWithIndependentDecoder(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Apple Vision interoperability check requires macOS")
	}
	clang, err := exec.LookPath("clang")
	if err != nil {
		t.Skip("clang unavailable")
	}
	temp := t.TempDir()
	decoder := filepath.Join(temp, "decode")
	build := exec.Command(clang, "-fobjc-arc", "../../scripts/experiments/qr/decode_fixture.m", "-framework", "Foundation", "-framework", "AppKit", "-framework", "Vision", "-o", decoder)
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("decoder: %v\n%s", err, out)
	}
	vm := lua.NewState()
	defer vm.Close()
	if err := vm.DoFile("../../apps/addon/Jornia/QR.lua"); err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{"WBACK 12345678 12715 001 0008 0000000000 00000", "Where now? åäö 中文 🧙", strings.Repeat("A", 584)} {
		vm.SetGlobal("PAYLOAD", lua.LString(text))
		if err := vm.DoString(`M,N=JorniaQR.Encode(PAYLOAD)`); err != nil {
			t.Fatal(err)
		}
		n := int(vm.GetGlobal("N").(lua.LNumber))
		matrix := vm.GetGlobal("M").(*lua.LTable)
		for _, pitch := range []int{2, 4} {
			side := (n + 8) * pitch
			img := image.NewGray(image.Rect(0, 0, side, side))
			for y := 0; y < side; y++ {
				for x := 0; x < side; x++ {
					shade := uint8(255)
					mx, my := x/pitch-4, y/pitch-4
					if mx >= 0 && my >= 0 && mx < n && my < n && matrix.RawGet(lua.LNumber(my)).(*lua.LTable).RawGet(lua.LNumber(mx)) == lua.LTrue {
						shade = 0
					}
					img.SetGray(x, y, color.Gray{Y: shade})
				}
			}
			file := filepath.Join(temp, "fixture.png")
			f, err := os.Create(file)
			if err != nil {
				t.Fatal(err)
			}
			err = png.Encode(f, img)
			f.Close()
			if err != nil {
				t.Fatal(err)
			}
			out, err := exec.Command(decoder, file).Output()
			if err != nil || string(out) != text {
				t.Fatalf("version modules=%d, pitch=%d, bytes=%d: mismatch (%v)", n, pitch, len(text), err)
			}
		}
	}
	if err := vm.DoString(`assert(not pcall(JorniaQR.Encode,string.rep('a',585)))`); err != nil {
		t.Fatal(err)
	}
}
