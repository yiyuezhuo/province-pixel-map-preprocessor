package main

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg" // support jpeg
	"image/png"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Area struct {
	BaseColor  color.Color
	RemapColor color.Color
	Points     int
	X          float64
	Y          float64
	Neighbors  map[*Area]struct{}
}

type AreaReduced struct {
	BaseColor  [4]int
	RemapColor [4]int
	Points     int
	X          float64
	Y          float64
	Neighbors  []([4]int)
}

// Some dumb JSON parser don't like bare list :<
type JsonResult struct {
	Areas []*AreaReduced
}

func Connect(area1 *Area, area2 *Area) {
	area1.Neighbors[area2] = struct{}{}
	area2.Neighbors[area1] = struct{}{}
}

func EncodeColor(c color.Color) [4]int {
	r, g, b, a := c.RGBA()
	return [4]int{int(r >> 8), int(g >> 8), int(b >> 8), int(a >> 8)} // TODO: check correctness?
}

func (area *Area) Reduce() *AreaReduced {
	nei := make([]([4]int), 0)
	for neiArea := range area.Neighbors {
		nei = append(nei, EncodeColor(neiArea.BaseColor))
	}

	baseArr := EncodeColor(area.BaseColor)
	remapArr := EncodeColor(area.RemapColor)

	return &AreaReduced{
		baseArr,
		remapArr,
		area.Points,
		area.X,
		area.Y,
		nei,
	}
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Please specify an image path. (You can drag file into terminal to input path)")
		return
	}
	path := os.Args[1]

	startTime := time.Now()
	// defer fmt.Println("Duration:", time.Since(startTime)) // not working :<

	reader, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	img, _, err := image.Decode(reader)
	if err != nil {
		log.Fatal(err)
	}
	bounds := img.Bounds()
	fmt.Println("bounds:", bounds)

	idx := 0
	areaMap := make(map[color.Color]*Area)

	remapImg := image.NewRGBA(bounds)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			baseColor := img.At(x, y)
			var area *Area
			var ok bool
			if area, ok = areaMap[baseColor]; !ok {
				low := uint8(idx % 256)
				high := uint8(idx / 256) // check type
				remapColor := color.RGBA{low, high, 0, 255}
				area = &Area{baseColor, remapColor, 0, 0, 0, make(map[*Area]struct{})}
				areaMap[baseColor] = area
				idx += 1
			}
			remapImg.Set(x, y, area.RemapColor)
			area.Points += 1
			area.X += float64(x)
			area.Y += float64(y)
		}
	}

	for _, area := range areaMap {
		area.X /= float64(area.Points)
		area.Y /= float64(area.Points)
	}

	if idx > 256*256 {
		log.Fatal("Province color should be less than 256*256")
	}

	fmt.Println("Area size:", len(areaMap))

	name := strings.TrimSuffix(path, filepath.Ext(path)) //  "abc/def/hello.blah" -> "abc/def/hello"

	f, _ := os.Create(name + "_remap.png")
	png.Encode(f, remapImg)

	for y := bounds.Min.Y; y < bounds.Max.Y-1; y++ {
		for x := bounds.Min.X; x < bounds.Max.X-1; x++ {
			c1 := img.At(x, y)
			c2 := img.At(x, y+1)
			c3 := img.At(x+1, y)
			if c1 != c2 {
				Connect(areaMap[c1], areaMap[c2])
			}
			if c1 != c3 {
				Connect(areaMap[c1], areaMap[c3])
			}
		}
	}

	reduceList := []*AreaReduced{}
	for _, area := range areaMap {
		reduceList = append(reduceList, area.Reduce())
	}

	file, err := os.Create(name + "_data.json")
	if err != nil {
		log.Fatal(err)
	}

	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.Encode(JsonResult{reduceList})

	fmt.Println("Duration:", time.Since(startTime))

}
