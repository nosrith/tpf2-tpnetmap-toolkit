package main

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"os"

	"golang.org/x/image/draw"
	"gopkg.in/yaml.v2"
)

type HSLA struct {
	H float64
	S float64
	L float64
	A float64
}

type HeightColorStop struct {
	Height float64  `yaml:"height"`
	Color  [3]uint8 `yaml:"color"`
}

type Settings struct {
	ImagePath         string            `yaml:"imagePath"`
	MinHeight         float64           `yaml:"minHeight"`
	waterHeight       float64           `yaml:"waterHeight"`
	MaxHeight         float64           `yaml:"maxHeight"`
	HeightPixelRatio  float64           `yaml:"heightPixelRatio"`
	LightPitchDeg     float64           `yaml:"lightPitch"`
	LightYawDeg       float64           `yaml:"lightYaw"`
	NoLightUnderWater bool              `yaml:"noLightUnderWater"`
	HeightColorStops  []HeightColorStop `yaml:"heightColorStops"`
	OutPath           string            `yaml:"outPath"`
	OutScale          float64           `yaml:"outScale"`
}

func minRGB(r uint8, g uint8, b uint8) uint8 {
	if r <= g && r <= b {
		return r
	} else if g <= r && g <= b {
		return g
	} else {
		return b
	}
}

func maxRGB(r uint8, g uint8, b uint8) uint8 {
	if r >= g && r >= b {
		return r
	} else if g >= r && g >= b {
		return g
	} else {
		return b
	}
}

func fromRGBAToHSLA(col color.RGBA) HSLA {
	min, max := minRGB(col.R, col.G, col.B), maxRGB(col.R, col.G, col.B)
	h := float64(0)
	if col.B < col.R && col.B < col.G {
		h = (float64(col.G)-float64(col.R))/float64(max-min) + 1
	} else if col.R < col.G && col.R < col.B {
		h = (float64(col.B)-float64(col.G))/float64(max-min) + 3
	} else if col.G < col.R && col.G < col.B {
		h = (float64(col.R)-float64(col.B))/float64(max-min) + 5
	}
	s := float64(0)
	if max > min {
		s = float64(max-min) / (255 - math.Abs(float64(max)+float64(min)-255))
	}
	return HSLA{
		H: h,
		S: s,
		L: (float64(max) + float64(min)) / 2 / 255,
		A: float64(col.A / 255),
	}
}

func fromHSLAToRGBA(col HSLA) color.RGBA {
	max := col.L + col.S*(1-math.Abs(2*col.L-1))/2
	min := col.L - col.S*(1-math.Abs(2*col.L-1))/2
	r, g, b := float64(0), float64(0), float64(0)
	if col.H < 1 {
		r, g, b = max, min+(max-min)*col.H, min
	} else if col.H < 2 {
		r, g, b = min+(max-min)*(2-col.H), max, min
	} else if col.H < 3 {
		r, g, b = min, max, min+(max-min)*(col.H-2)
	} else if col.H < 4 {
		r, g, b = min, min+(max-min)*(4-col.H), max
	} else if col.H < 5 {
		r, g, b = min+(max-min)*(col.H-4), min, max
	} else {
		r, g, b = max, min, min+(max-min)*(6-col.H)
	}
	return color.RGBA{
		R: uint8(r * 255),
		G: uint8(g * 255),
		B: uint8(b * 255),
		A: uint8(col.A * 255),
	}
}

func loadSettings(path string) Settings {
	settings := Settings{}

	b, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}

	yaml.Unmarshal(b, &settings)
	return settings
}

func loadHeightMap(path string) image.Image {
	f, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	img, err := png.Decode(f)
	if err != nil {
		panic(err)
	}

	return img
}

func calculateHeightColor(heightMapImage image.Image, hillShadeImage *image.RGBA, settings Settings) {
	bounds := heightMapImage.Bounds()
	heightScale := (settings.MaxHeight - settings.MinHeight) / 65536
	heightOffset := settings.MinHeight
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			height := float64(heightMapImage.At(x, y).(color.Gray16).Y)*heightScale + heightOffset
			for i, stop := range settings.HeightColorStops {
				if height <= stop.Height {
					if i == 0 {
						color := color.RGBA{R: stop.Color[0], G: stop.Color[1], B: stop.Color[2], A: 255}
						hillShadeImage.Set(x, y, color)
					} else {
						prevStop := settings.HeightColorStops[i-1]
						prevColorHSL := fromRGBAToHSLA(color.RGBA{R: prevStop.Color[0], G: prevStop.Color[1], B: prevStop.Color[2], A: 255})
						currColorHSL := fromRGBAToHSLA(color.RGBA{R: stop.Color[0], G: stop.Color[1], B: stop.Color[2], A: 255})
						f := float64(stop.Height-height) / float64(stop.Height-prevStop.Height)
						h := float64(0)
						if prevColorHSL.H-currColorHSL.H < -3 {
							h = (prevColorHSL.H+1)*f + currColorHSL.H*(1-f)
						} else if prevColorHSL.H-currColorHSL.H < 3 {
							h = prevColorHSL.H*f + currColorHSL.H*(1-f)
						} else {
							h = prevColorHSL.H*f + (currColorHSL.H+1)*(1-f)
						}
						h = math.Mod(h, 6)
						color := fromHSLAToRGBA(HSLA{
							H: h,
							S: prevColorHSL.S*f + currColorHSL.S*(1-f),
							L: prevColorHSL.L*f + currColorHSL.L*(1-f),
							A: 1.0,
						})
						hillShadeImage.Set(x, y, color)
					}
					break
				} else if i == len(settings.HeightColorStops)-1 {
					color := color.RGBA{R: stop.Color[0], G: stop.Color[1], B: stop.Color[2], A: 255}
					hillShadeImage.Set(x, y, color)
				}
			}
		}
	}
}

func calculateHillShade(heightMapImage image.Image, hillShadeImage *image.RGBA, settings Settings) {
	bounds := heightMapImage.Bounds()
	heightScale := (settings.MaxHeight - settings.MinHeight) / 65536
	heightOffset := settings.MinHeight
	zenithRad := (90 - settings.LightPitchDeg) * math.Pi / 180
	azimuthRad := math.Mod(450-settings.LightYawDeg, 360) * math.Pi / 180
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		ty := y
		if y == bounds.Min.Y {
			ty++
		} else if y == bounds.Max.Y-1 {
			ty--
		}
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			tx := x
			if x == bounds.Min.X {
				tx++
			} else if x == bounds.Max.X-1 {
				tx--
			}

			height := float64(heightMapImage.At(tx, ty).(color.Gray16).Y)*heightScale + heightOffset
			if !(settings.NoLightUnderWater && height <= settings.waterHeight) {
				hA := float64(heightMapImage.At(tx-1, ty-1).(color.Gray16).Y)
				hB := float64(heightMapImage.At(tx, ty-1).(color.Gray16).Y)
				hC := float64(heightMapImage.At(tx+1, ty-1).(color.Gray16).Y)
				hD := float64(heightMapImage.At(tx-1, ty).(color.Gray16).Y)
				hF := float64(heightMapImage.At(tx+1, ty).(color.Gray16).Y)
				hG := float64(heightMapImage.At(tx-1, ty+1).(color.Gray16).Y)
				hH := float64(heightMapImage.At(tx, ty+1).(color.Gray16).Y)
				hI := float64(heightMapImage.At(tx+1, ty+1).(color.Gray16).Y)
				dzdx := ((hC + 2*hF + hI) - (hA + 2*hD + hG)) / 8
				dzdy := ((hG + 2*hH + hI) - (hA + 2*hB + hC)) / 8
				slopeRad := math.Atan(heightScale / settings.HeightPixelRatio * settings.OutScale * math.Sqrt(dzdx*dzdx+dzdy*dzdy))
				aspectRad := float64(0)
				if dzdx != 0 {
					aspectRad = math.Atan2(dzdy, -dzdx)
				} else {
					if dzdy > 0 {
						aspectRad = math.Pi / 2
					} else if dzdy < 0 {
						aspectRad = 3 * math.Pi / 2
					}
				}
				hillShade := ((math.Cos(zenithRad) * math.Cos(slopeRad)) + (math.Sin(zenithRad) * math.Sin(slopeRad) * math.Cos(azimuthRad-aspectRad)))
				if hillShade < 0 {
					hillShade = 0
				} else if hillShade > 1 {
					hillShade = 1
				}

				origColor := hillShadeImage.At(x, y).(color.RGBA)
				colorHSL := fromRGBAToHSLA(origColor)
				colorHSL.L *= hillShade
				color := fromHSLAToRGBA(colorHSL)
				hillShadeImage.Set(x, y, color)
			}
		}
	}
}

func writeImage(path string, image image.Image) {
	f, err := os.Create(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	err = png.Encode(f, image)
	if err != nil {
		panic(err)
	}
}

func main() {
	var settingsPath string
	if len(os.Args) > 1 {
		settingsPath = os.Args[1]
	} else {
		settingsPath = "hillshade_settings.yaml"
	}
	settings := loadSettings(settingsPath)

	heightMapImage := loadHeightMap(settings.ImagePath)

	if settings.OutScale != 1.0 {
		bounds := heightMapImage.Bounds()
		scaledBounds := image.Rect(
			0, 0,
			int(float64(bounds.Dx())*settings.OutScale),
			int(float64(bounds.Dy())*settings.OutScale))
		scaledHeightMapImage := image.NewGray16(scaledBounds)
		draw.BiLinear.Scale(scaledHeightMapImage, scaledBounds, heightMapImage, bounds, draw.Over, nil)
		heightMapImage = scaledHeightMapImage
	}

	hillShadeImage := image.NewRGBA(heightMapImage.Bounds())

	calculateHeightColor(heightMapImage, hillShadeImage, settings)

	calculateHillShade(heightMapImage, hillShadeImage, settings)

	writeImage(settings.OutPath, hillShadeImage)
}
