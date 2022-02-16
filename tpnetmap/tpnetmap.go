package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"unicode"

	"gopkg.in/yaml.v2"
)

type Settings struct {
	MapPath                        string            `yaml:"mapPath"`
	OutPath                        string            `yaml:"outPath"`
	OutSize                        int               `yaml:"outSize"`
	BackgroundPath                 string            `yaml:"backgroundPath"`
	TownLabelAttributes            map[string]string `yaml:"townLabelAttributes"`
	StationsHavingTrackOnly        bool              `yaml:"stationsHavingTrackOnly"`
	SuppressDuplicatedCargoStation bool              `yaml:"suppressDuplicatedCargoStation"`
	StationMarkerAttributes        map[string]string `yaml:"stationMarkerAttributes"`
	StationLabelAttributes         map[string]string `yaml:"stationLabelAttributes"`
	TrackSpeedLimitThreshold       float64           `yaml:"trackSpeedLimitThreshold"`
	TrackPathAttributes            map[string]string `yaml:"trackPathAttributes"`
	StreetWidthThreshold           float64           `yaml:"streetWidthThreshold"`
	StreetPathBaseWidth            float64           `yaml:"streetPathBaseWidth"`
	StreetPathAttributes           map[string]string `yaml:"streetPathAttributes"`
	WideStreetWidthThreshold       float64           `yaml:"wideStreetWidthThreshold"`
	WideStreetPathAttributes       map[string]string `yaml:"wideStreetPathAttributes"`
	WideStreetBorderAttributes     map[string]string `yaml:"wideStreetBorderAttributes"`
}

type Pos3D struct {
	X float64
	Y float64
	Z float64
}

type Station struct {
	Id       int
	Name     string
	Cargo    bool
	HasTrack bool `yaml:"hasTrack"`
	X        float64
	Y        float64
	Z        float64
}

type MapData struct {
	World struct {
		Min Pos3D
		Max Pos3D
	}
	Towns []struct {
		Id   int
		Name string
		X    float64
		Y    float64
		Z    float64
	}
	Stations []Station
	Tracks   struct {
		Paths []struct {
			Nodes      []int
			SpeedLimit float64 `yaml:"speedLimit"`
		}
		Nodes map[int]Pos3D
	}
	Streets struct {
		Paths []struct {
			Nodes      []int
			NumLanes   int `yaml:"numLanes"`
			Width      float64
			SpeedLimit float64 `yaml:"speedLimit"`
		}
		Nodes map[int]Pos3D
	}
}

type Transform struct {
	x func(x float64) float64
	y func(y float64) float64
}

func loadSettings(path string) Settings {
	var settings Settings

	b, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}

	yaml.Unmarshal(b, &settings)
	return settings
}

func loadMapData(path string) MapData {
	var mapData MapData

	b, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}

	if err = yaml.Unmarshal(b, &mapData); err != nil {
		panic(err)
	}

	return mapData
}

var kebabCache = make(map[string]string)

func fromCamelToKebab(s string) string {
	if v, ok := kebabCache[s]; ok {
		return v
	}
	var sb strings.Builder
	sb.Grow(len(s) * 2)
	for _, c := range s {
		if c >= 'A' && c <= 'Z' {
			sb.WriteRune('-')
			sb.WriteRune(unicode.ToLower(c))
		} else {
			sb.WriteRune(c)
		}
	}
	sbs := sb.String()
	kebabCache[s] = sbs
	return sbs
}

func getAttributeString(attrMap ...map[string]string) string {
	totalMap := make(map[string]string)
	for _, m := range attrMap {
		for k, v := range m {
			totalMap[k] = v
		}
	}

	var sb strings.Builder
	for k, v := range totalMap {
		sb.WriteString(` `)
		sb.WriteString(fromCamelToKebab(k))
		sb.WriteString(`="`)
		sb.WriteString(v)
		sb.WriteString(`"`)
	}
	return sb.String()
}

func writePath(nodes []int, nodeMap map[int]Pos3D, tf Transform, attrStr string, writer *bufio.Writer) {
	writer.WriteString(`<path d="`)
	for i, n := range nodes {
		if i == 0 {
			writer.WriteString("M")
		} else {
			writer.WriteString(" L")
		}
		pos := nodeMap[n]
		writer.WriteString(fmt.Sprintf("%f %f", tf.x(pos.X), tf.y(pos.Y)))
	}
	writer.WriteString(fmt.Sprintf(`"%s/>`, attrStr))
}

func writeStreets(mapData MapData, settings Settings, tf Transform, writer *bufio.Writer) {
	attrStr := getAttributeString(
		map[string]string{"strokeWidth": fmt.Sprintf("%f", settings.StreetPathBaseWidth)},
		settings.StreetPathAttributes,
	)
	for _, p := range mapData.Streets.Paths {
		width := p.Width * float64(p.NumLanes)
		if width >= settings.StreetWidthThreshold && width < settings.WideStreetWidthThreshold {
			writePath(p.Nodes, mapData.Streets.Nodes, tf, attrStr, writer)
		}
	}
}

func writeWideStreets(mapData MapData, settings Settings, tf Transform, writer *bufio.Writer) {
	for _, p := range mapData.Streets.Paths {
		width := p.Width * float64(p.NumLanes)
		if width >= settings.WideStreetWidthThreshold {
			attrStr := getAttributeString(
				map[string]string{"strokeWidth": fmt.Sprintf("%f", width*0.4+settings.StreetPathBaseWidth)},
				settings.WideStreetBorderAttributes,
			)
			writePath(p.Nodes, mapData.Streets.Nodes, tf, attrStr, writer)
		}
	}
	for _, p := range mapData.Streets.Paths {
		width := p.Width * float64(p.NumLanes)
		if width >= settings.WideStreetWidthThreshold {
			attrStr := getAttributeString(
				map[string]string{"strokeWidth": fmt.Sprintf("%f", width*0.3+settings.StreetPathBaseWidth-0.5)},
				settings.WideStreetPathAttributes,
			)
			writePath(p.Nodes, mapData.Streets.Nodes, tf, attrStr, writer)
		}
	}
}

func writeTracks(mapData MapData, settings Settings, tf Transform, writer *bufio.Writer) {
	attrStr := getAttributeString(
		map[string]string{"strokeWidth": "3"},
		settings.TrackPathAttributes,
	)
	for _, p := range mapData.Tracks.Paths {
		if p.SpeedLimit >= settings.TrackSpeedLimitThreshold {
			writePath(p.Nodes, mapData.Tracks.Nodes, tf, attrStr, writer)
		}
	}
}

func writeTownLabel(mapData MapData, settings Settings, tf Transform, writer *bufio.Writer) {
	attrStr := getAttributeString(settings.TownLabelAttributes)
	for _, t := range mapData.Towns {
		writer.WriteString(fmt.Sprintf(
			`<text x="%f" y="%f" fill="#000000" text-anchor="middle" dominant-baseline="central"%s>%s</text>`,
			tf.x(t.X),
			tf.y(t.Y),
			attrStr,
			t.Name,
		))
	}
}

func isTargetStation(s Station, mapData MapData, settings Settings) bool {
	if settings.StationsHavingTrackOnly && !s.HasTrack {
		return false
	}
	if settings.SuppressDuplicatedCargoStation && s.Cargo {
		for _, s2 := range mapData.Stations {
			if s2.Name == s.Name && (s2.Id < s.Id || !s2.Cargo) {
				return false
			}
		}
	}
	return true
}

func writeStationMarker(mapData MapData, settings Settings, tf Transform, writer *bufio.Writer) {
	attrStr := getAttributeString(settings.StationMarkerAttributes)
	for _, s := range mapData.Stations {
		if isTargetStation(s, mapData, settings) {
			writer.WriteString(fmt.Sprintf(
				`<circle cx="%f" cy="%f" r="6" stroke-width="3"%s/>`,
				tf.x(s.X),
				tf.y(s.Y),
				attrStr,
			))
		}
	}
}

func writeStationLabel(mapData MapData, settings Settings, tf Transform, writer *bufio.Writer) {
	attrStr := getAttributeString(settings.StationLabelAttributes)
	for _, s := range mapData.Stations {
		if isTargetStation(s, mapData, settings) {
			writer.WriteString(fmt.Sprintf(
				`<text x="%f" y="%f" fill="#000000" font-size="20"%s>%s</text>`,
				tf.x(s.X)+4,
				tf.y(s.Y)-4,
				attrStr,
				s.Name,
			))
		}
	}
}

func main() {
	var settingsPath string
	if len(os.Args) > 1 {
		settingsPath = os.Args[1]
	} else {
		settingsPath = "tpnetmap_settings.yaml"
	}
	settings := loadSettings(settingsPath)

	mapData := loadMapData(settings.MapPath)

	tf := Transform{
		x: func(x float64) float64 {
			return (x - mapData.World.Min.X) * float64(settings.OutSize) / (mapData.World.Max.X - mapData.World.Min.X)
		},
		y: func(y float64) float64 {
			return (mapData.World.Max.Y - y) * float64(settings.OutSize) / (mapData.World.Max.X - mapData.World.Min.X)
		},
	}
	outHeight := int(float64(settings.OutSize) * (mapData.World.Max.Y - mapData.World.Min.Y) / (mapData.World.Max.X - mapData.World.Min.X))

	f, err := os.Create(settings.OutPath)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	writer := bufio.NewWriter(f)
	writer.WriteString(fmt.Sprintf(
		`<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" width="%d" height="%d" fill="none">`,
		settings.OutSize,
		outHeight,
	))
	if settings.BackgroundPath != "" {
		writer.WriteString(fmt.Sprintf(
			`<image xlink:href="%s" width="%d" height="%d"/>`,
			settings.BackgroundPath,
			settings.OutSize,
			outHeight,
		))
	}
	writeWideStreets(mapData, settings, tf, writer)
	writeStreets(mapData, settings, tf, writer)
	writeTracks(mapData, settings, tf, writer)
	writeStationMarker(mapData, settings, tf, writer)
	writeTownLabel(mapData, settings, tf, writer)
	writeStationLabel(mapData, settings, tf, writer)
	writer.WriteString(`</svg>`)
	writer.Flush()
}
