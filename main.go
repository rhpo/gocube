package main

/*
#include <unistd.h>
static long gethz(void) { return sysconf(_SC_CLK_TCK); }
*/
import "C"

import (
	"fmt"
	"github.com/fatih/color"
	"golang.org/x/term"
	"math"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
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

const windows = runtime.GOOS == "windows"

func clear() {
	// Cross-platform clear
	if windows {
		cmd := exec.Command("cmd", "/c", "cls")
		cmd.Stdout = os.Stdout
		_ = cmd.Run()
	} else {
		// ANSI clear for Unix terminals
		fmt.Print("\033[H\033[J")
	}
}

func drawAxis() {
	originX := int(float64(sW)/2.0 - camera.x)
	originY := int(float64(sH)/2.0 - camera.y)

	// vertical axis (|) along columns at originX
	for i := 0; i < sH; i++ {
		if originX >= 0 && originX < sW {
			screen[i][originX] = "|"
		}
	}

	// horizontal axis (-) along row originY
	for j := 0; j < sW; j++ {
		if originY >= 0 && originY < sH {
			screen[originY][j] = "-"
		}
	}

	// origin point
	if originY >= 0 && originY < sH && originX >= 0 && originX < sW {
		screen[originY][originX] = "+"
	}
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
}

func initTerminalSize() {
	// Try to get terminal size in a cross-platform way; fallback to defaults.
	w, h, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || w == 0 || h == 0 {
		// fallback defaults
		w, h = 120, 40
	}
	sW, sH = w, h
}

func render() {
	// target ~60 FPS
	hz := C.gethz()
	hzInt := int(hz)

	frameDelay := time.Duration(1/hzInt) * time.Second

	time.Sleep(frameDelay)

	clear()

	var innerJoined []string
	for _, innerSlice := range screen {
		innerJoined = append(innerJoined, strings.Join(innerSlice, ""))
	}
	out := strings.Join(innerJoined, "\n")
	fmt.Print(out)

	// prepare for next frame
	wipeScreen()
	drawAxis()
}

func setPixel(p Point) {
	x := p.x - camera.x
	y := -(p.y - camera.y) / 2.0 // squash Y for aspect
	z := p.z - camera.z

	if z <= 0 {
		return
	}

	scale := 40.0 / z

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
	A := rotation.A * math.Pi / 180.0
	B := rotation.B * math.Pi / 180.0
	C := rotation.C * math.Pi / 180.0

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
	// Using background colors from fatih/color
	red := color.New(color.BgHiRed).SprintFunc()
	blue := color.New(color.BgHiBlue).SprintFunc()
	white := color.New(color.BgHiWhite).SprintFunc()
	green := color.New(color.BgHiGreen).SprintFunc()
	yellow := color.New(color.BgHiYellow).SprintFunc()
	magenta := color.New(color.BgHiMagenta).SprintFunc()
	char := " "

	chars := []string{
		red(char),     // front
		blue(char),    // back
		green(char),   // right
		yellow(char),  // left
		white(char),   // top
		magenta(char), // bottom
	}

	points := []Point{}
	half := size / 2.0

	for i, ch := range chars {
		switch i {
		case 0: // front z = +half
			for x := -half; x <= half; x += density {
				for y := -half; y <= half; y += density {
					points = append(points, Point{x, y, half, ch})
				}
			}
		case 1: // back z = -half
			for x := -half; x <= half; x += density {
				for y := -half; y <= half; y += density {
					points = append(points, Point{x, y, -half, ch})
				}
			}
		case 2: // right x = +half
			for z := -half; z <= half; z += density {
				for y := -half; y <= half; y += density {
					points = append(points, Point{half, y, z, ch})
				}
			}
		case 3: // left x = -half
			for z := -half; z <= half; z += density {
				for y := -half; y <= half; y += density {
					points = append(points, Point{-half, y, z, ch})
				}
			}
		case 4: // top y = +half
			for x := -half; x <= half; x += density {
				for z := -half; z <= half; z += density {
					points = append(points, Point{x, half, z, ch})
				}
			}
		case 5: // bottom y = -half
			for x := -half; x <= half; x += density {
				for z := -half; z <= half; z += density {
					points = append(points, Point{x, -half, z, ch})
				}
			}
		}
	}

	return points
}

func generatePyramidSurfaces(size, density float64) []Point {
	points := []Point{}
	half := size / 2.0

	// base square on y=0
	for x := -half; x <= half; x += density {
		for z := -half; z <= half; z += density {
			points = append(points, Point{x, 0, z, "#"})
		}
	}

	// triangular faces
	base := []Point{
		{-half, 0, -half, "#"}, // back-left
		{half, 0, -half, "#"},  // back-right
		{half, 0, half, "#"},   // front-right
		{-half, 0, half, "#"},  // front-left
	}
	apex := Point{0, size, 0, "#"}

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
	// encourage color output on Windows terminals (helps PowerShell/Windows Terminal)
	color.NoColor = false

	// determine terminal size
	initTerminalSize()
	wipeScreen()

	// set initial camera
	camera = Point{
		x: 0,
		y: 0,
		z: 20,
	}

	// build cube
	var cubeSize = sW / 5
	cube := Shape{
		points: generateCubeSurfaces(float64(cubeSize), 0.2),
		rotation: Rotation{
			A: 0,
			B: 0,
			C: 0,
		},
	}

	// push cube forward a bit so it's visible
	for i := range cube.points {
		cube.points[i].z += float64(cubeSize) * 2.0
	}

	var speed float64 = 3.0
	fmt.Printf("Please enter cube rotation speed (0 < r < 40) [default 3]: ")
	_, err := fmt.Scanf("%f", &speed)
	if err != nil {
		speed = 3.0
	}

	for {
		// update rotation
		cube.rotation.A += 3.0 * speed
		cube.rotation.B += 1.0 * speed
		cube.rotation.C += 1.0 * speed

		// camera movement (circular-ish)
		t := float64(time.Now().UnixNano()) / 1e9
		radius := 50.0
		camSpeed := 1.0 // radians per second
		minZ := 20.0

		camera.z = radius * math.Abs(math.Sin(camSpeed*t)) // keep positive
		if camera.z < minZ {
			camera.z = minZ
		}

		setShape(cube)
		render()
	}
}
