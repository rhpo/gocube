package main

import (
	"fmt"
	"math"
	"strings"
	"time"

	"golang.org/x/term"
)

type Point struct {
	x, y, z float64
	char    string
}

type Rotation struct {
	A, B, C float64
}

type Shape struct {
	points   []Point
	rotation Rotation
}

const background = " "

var sW, sH int
var screen [][]string
var camera Point
var zBuffer []float64

func clear() {
	fmt.Print("\033[H\033[J")
}

func wipeScreen() {

	zBuffer = make([]float64, sW*sH)
	for i := range zBuffer {
		zBuffer[i] = math.Inf(1)
	}

	screen = make([][]string, sH)
	for i := range screen {
		screen[i] = make([]string, sW)
		for j := range screen[i] {
			screen[i][j] = background
		}
	}

	originX := int(sW/2 - int(camera.x))
	originY := int(sH/2 - int(camera.y))

	for i := 0; i < sH; i++ {
		if originX >= 0 && originX < sW {
			screen[i][originX] = "|"
		}
	}
	for j := 0; j < sW; j++ {
		if originY >= 0 && originY < sH {
			screen[originY][j] = "-"
		}
	}

	if originY >= 0 && originY < sH && originX >= 0 && originX < sW {
		screen[originY][originX] = "+"
	}
}

func init() {

	sW, sH, _ = term.GetSize(0)

	camera = Point{
		x: 0,
		y: 0,
		z: 20,
	}

	wipeScreen()
}

func render() {
	time.Sleep(15 * time.Millisecond)
	clear()

	var innerJoined []string
	for _, innerSlice := range screen {
		innerJoined = append(innerJoined, strings.Join(innerSlice, ""))
	}
	out := strings.Join(innerJoined, "\n")
	fmt.Print(out)

	wipeScreen()
}

func normal(f float64) int {
	return int(math.Round(f))
}

func setPixel(p Point) {

	x := p.x - camera.x
	y := -(p.y - camera.y) / 2
	z := p.z - camera.z

	if z <= 0 {
		return
	}
	scale := 40 / z

	screenX := int(x*scale) + sW/2
	screenY := int(y*scale) + sH/2

	if screenX < 0 || screenX >= sW || screenY < 0 || screenY >= sH {
		return
	}

	index := screenY*sW + screenX

	if z < zBuffer[index] {
		zBuffer[index] = z
		screen[screenY][screenX] = p.char
	}
}

func setShape(s Shape) {

	rotated := Rotate(s)

	for _, p := range rotated.points {
		setPixel(p)
	}
}

func RotatePoint(p Point, rotation Rotation, center Point) Point {
	A := rotation.A * math.Pi / 180
	B := rotation.B * math.Pi / 180
	C := rotation.C * math.Pi / 180

	cA, sA := math.Cos(A), math.Sin(A)
	cB, sB := math.Cos(B), math.Sin(B)
	cC, sC := math.Cos(C), math.Sin(C)

	x := p.x - center.x
	y := p.y - center.y
	z := p.z - center.z

	x1 := x*cC - y*sC
	y1 := x*sC + y*cC
	z1 := z

	x2 := x1*cB + z1*sB
	y2 := y1
	z2 := -x1*sB + z1*cB

	x3 := x2
	y3 := y2*cA - z2*sA
	z3 := y2*sA + z2*cA

	return Point{
		x:    x3 + center.x,
		y:    y3 + center.y,
		z:    z3 + center.z,
		char: p.char,
	}
}

func Rotate(s Shape) Shape {

	var cx, cy, cz float64
	for _, p := range s.points {
		cx += p.x
		cy += p.y
		cz += p.z
	}
	n := float64(len(s.points))
	center := Point{cx / n, cy / n, cz / n, ""}

	result := Shape{
		points:   make([]Point, 0, len(s.points)),
		rotation: s.rotation,
	}

	for _, p := range s.points {
		result.points = append(result.points, RotatePoint(p, s.rotation, center))
	}

	return result
}

func generateCubeSurfaces(size float64, density float64) []Point {

	chars := []string{"#", "@", "+", "x", "o", "%"}
	points := []Point{}

	half := size / 2

	for i, char := range chars {
		switch i {

		case 0:
			for x := -half; x <= half; x += density {
				for y := -half; y <= half; y += density {
					points = append(points, Point{x, y, half, char})
				}
			}

		case 1:
			for x := -half; x <= half; x += density {
				for y := -half; y <= half; y += density {
					points = append(points, Point{x, y, -half, char})
				}
			}

		case 2:
			for z := -half; z <= half; z += density {
				for y := -half; y <= half; y += density {
					points = append(points, Point{half, y, z, char})
				}
			}

		case 3:
			for z := -half; z <= half; z += density {
				for y := -half; y <= half; y += density {
					points = append(points, Point{-half, y, z, char})
				}
			}

		case 4:
			for x := -half; x <= half; x += density {
				for z := -half; z <= half; z += density {
					points = append(points, Point{x, half, z, char})
				}
			}

		case 5:
			for x := -half; x <= half; x += density {
				for z := -half; z <= half; z += density {
					points = append(points, Point{x, -half, z, char})
				}
			}
		}
	}

	return points
}

func generatePyramidSurfaces(size, density float64) []Point {
	points := []Point{}
	half := size / 2

	base := []Point{
		{-half, 0, -half, "#"}, // back-left
		{half, 0, -half, "#"},  // back-right
		{half, 0, half, "#"},   // front-right
		{-half, 0, half, "#"},  // front-left
	}
	apex := Point{0, size, 0, "#"}

	// Base (square)
	for x := -half; x <= half; x += density {
		for z := -half; z <= half; z += density {
			points = append(points, Point{x, 0, z, "#"})
		}
	}

	// 4 triangular faces
	faces := []struct {
		a, b, c Point
		char    string
	}{
		{base[0], base[1], apex, "*"}, // back
		{base[1], base[2], apex, "+"}, // right
		{base[2], base[3], apex, "x"}, // front
		{base[3], base[0], apex, "%"}, // left
	}

	for _, f := range faces {
		for t := 0.0; t <= 1.0; t += density / size {
			for s := 0.0; s <= 1.0-t; s += density / size {
				x := f.a.x + (f.b.x-f.a.x)*t + (f.c.x-f.a.x)*s
				y := f.a.y + (f.b.y-f.a.y)*t + (f.c.y-f.a.y)*s
				z := f.a.z + (f.b.z-f.a.z)*t + (f.c.z-f.a.z)*s
				points = append(points, Point{x, y, z, f.char})
			}
		}
	}

	return points
}

func main() {

	var cubeSize = sW / 5
	cube := Shape{
		points: generateCubeSurfaces(float64(cubeSize), 0.5),
		rotation: Rotation{
			A: 0,
			B: 0,
			C: 0,
		},
	}

	for i := range cube.points {
		cube.points[i].z += float64(cubeSize) * 2
	}

	var speed float64 = 3

	fmt.Printf("Please enter cube rotation speed (0 < r < 40): ")
	fmt.Scanf("%f", &speed)

	for {

		cube.rotation.A += float64(speed)
		cube.rotation.B += float64(speed)
		cube.rotation.C += float64(speed)

		t := float64(time.Now().UnixNano()) / 1e9 // time in seconds
		radius := 50.0
		camSpeed := 1.0 // rotation camSpeed (radians per second)
		min := 20.0

		// make the camera move in a circle around the origin
		camera.z = radius * math.Sin(camSpeed*t)

		if camera.z < min {
			camera.z = min
		}

		setShape(cube)
		render()

	}
}
