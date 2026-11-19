package datacenter

import (
	"bytes"
	"image"
	"image/color"
	stddraw "image/draw"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/golang/freetype"
	"github.com/golang/freetype/truetype"
	"github.com/xuri/excelize/v2"
	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/math/f64"
)

const (
	watermarkTileW   = 520
	watermarkTileH   = 220
	watermarkCanvasW = 1600
	watermarkCanvasH = 1200
	watermarkFontPx  = 20
)

// applySheetBackgroundWatermark 为数据 sheet 设置斜向文字背景水印（无字体时跳过）
func applySheetBackgroundWatermark(f *excelize.File, sheet, wmText string) {
	if wmText == "" {
		return
	}
	pngData, err := buildDiagonalWatermarkPNG(wmText)
	if err != nil || len(pngData) == 0 {
		return
	}
	_ = f.SetSheetBackgroundFromBytes(sheet, ".png", pngData)
}

func buildDiagonalWatermarkPNG(text string) ([]byte, error) {
	font, err := loadWatermarkFont()
	if err != nil {
		return nil, err
	}
	tile := renderWatermarkTile(text, font)
	rotated := rotateImage(tile, -math.Pi/6) // -30°

	canvas := image.NewRGBA(image.Rect(0, 0, watermarkCanvasW, watermarkCanvasH))
	stddraw.Draw(canvas, canvas.Bounds(), image.White, image.Point{}, stddraw.Src)

	b := rotated.Bounds()
	for y := -b.Dy(); y < watermarkCanvasH+b.Dy(); y += b.Dy() * 2 / 3 {
		for x := -b.Dx(); x < watermarkCanvasW+b.Dx(); x += b.Dx() * 2 / 3 {
			stddraw.Draw(canvas, image.Rect(x, y, x+b.Dx(), y+b.Dy()), rotated, b.Min, stddraw.Over)
		}
	}

	buf := new(bytes.Buffer)
	if err := png.Encode(buf, canvas); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func renderWatermarkTile(text string, font *truetype.Font) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, watermarkTileW, watermarkTileH))
	stddraw.Draw(img, img.Bounds(), image.Transparent, image.Point{}, stddraw.Src)

	c := freetype.NewContext()
	c.SetDPI(96)
	c.SetFont(font)
	c.SetFontSize(watermarkFontPx)
	c.SetClip(img.Bounds())
	c.SetDst(img)
	c.SetSrc(image.NewUniform(color.RGBA{R: 0xB0, G: 0xB0, B: 0xB0, A: 0x90}))

	lines := strings.Split(text, "\n")
	y := 48
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		pt := freetype.Pt(16, y)
		if _, err := c.DrawString(line, pt); err != nil {
			return img
		}
		y += int(c.PointToFixed(float64(watermarkFontPx+8)) >> 6)
	}
	return img
}

func rotateImage(src *image.RGBA, angle float64) *image.RGBA {
	b := src.Bounds()
	cx := float64(b.Dx()) / 2
	cy := float64(b.Dy()) / 2
	cos := math.Cos(angle)
	sin := math.Sin(angle)

	// 旋转后包围盒
	w := int(math.Ceil(math.Abs(float64(b.Dx())*cos) + math.Abs(float64(b.Dy())*sin)))
	h := int(math.Ceil(math.Abs(float64(b.Dx())*sin) + math.Abs(float64(b.Dy())*cos)))
	if w < b.Dx() {
		w = b.Dx()
	}
	if h < b.Dy() {
		h = b.Dy()
	}

	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	affine := f64.Aff3{
		cos, -sin, cx - cos*cx + sin*cy + float64(w-b.Dx())/2,
		sin, cos, cy - sin*cx - cos*cy + float64(h-b.Dy())/2,
	}
	xdraw.BiLinear.Transform(dst, affine, src, b, xdraw.Over, nil)
	return dst
}

func loadWatermarkFont() (*truetype.Font, error) {
	candidates := watermarkFontCandidates()
	for _, path := range candidates {
		font, err := parseFontFile(path)
		if err == nil && font != nil {
			return font, nil
		}
	}
	return nil, os.ErrNotExist
}

func watermarkFontCandidates() []string {
	var paths []string
	if runtime.GOOS == "windows" {
		windir := os.Getenv("WINDIR")
		if windir == "" {
			windir = `C:\Windows`
		}
		fontsDir := filepath.Join(windir, "Fonts")
		paths = append(paths,
			filepath.Join(fontsDir, "msyh.ttc"),
			filepath.Join(fontsDir, "msyhbd.ttc"),
			filepath.Join(fontsDir, "simhei.ttf"),
			filepath.Join(fontsDir, "simsun.ttc"),
			filepath.Join(fontsDir, "arial.ttf"),
		)
	}
	paths = append(paths,
		"/usr/share/fonts/truetype/wqy/wqy-microhei.ttc",
		"/usr/share/fonts/opentype/noto/NotoSansCJK-Regular.ttc",
		"/usr/share/fonts/truetype/noto/NotoSansCJK-Regular.ttc",
		"/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf",
	)
	return paths
}

func parseFontFile(path string) (*truetype.Font, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return truetype.Parse(data)
}
