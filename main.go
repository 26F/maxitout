
package main

import (
	_"fmt"
	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/ebitenutil"
	"github.com/26F/maxitout/resources"
	"log"
	"image"
	"bytes"
	_ "image/png"
	"image/color"
	"time"
	"math"
	"math/rand"
	"errors"
	"strconv"
)

// pretend error that tells ebiten to exit
var ExitProgram = errors.New("Fake Error Safe Exit Program")

// Yes, most of the values below are globals. . .
const Width = 1920
const Height = 1080

// dimensions of the road slices graphics art used.
// And other stuff
const roadslicewidth = 150.0
const roadsliceheight = 1.0
const halfwidth = float64(Width) / 2

// center position of the road.
const roadcenterpos = int((halfwidth - (roadslicewidth / 2)))
var mousecenterposv int
var bikeinitpos int
var bikeposoffset float64 = 0
var clickedtobegin = false
var mouseinitpos float64 = 0
var mousechangeamount float64 = 0
var turningspeed float64 = 0.2
var acceleration float64 = 0.1
var crashed = false
const peroid = 3000
var scalefactor float64 = 1.0
var air float64 = 0.0
var currentSpeedStr string
var gamewasjustreset bool = false

type Point struct {
	x, y float64
}

var jumppos = Point{0,0}
var jumping bool = false

type Beziercurve struct {
	p0, p1, p2, p3 Point
}

var beziercurve = Beziercurve{Point{0,0}, Point{0,0}, Point{0,0}, Point{0,0}}

func (bez * Beziercurve) At(t float64) float64 {
	p0x, p1x, p2x, p3x := bez.p0.x, bez.p1.x, bez.p2.x, bez.p3.x
	x := math.Pow((1-t),3)*p0x + 3*(math.Pow((1-t),2)*t*p1x) + 3*((1-t)*math.Pow(t,2)*p2x) + math.Pow(t,3)*p3x
	return x
}

// number of desert trees
const ntrees = 5

// number of cactus
const ncactus = 2

var GlobalVelocity float64 = 0

// desert tree images
var deserttree1 * ebiten.Image 
var deserttree2 * ebiten.Image
var deserttree3 * ebiten.Image

// cactus
var cactus * ebiten.Image

var deserttreeimages = [](* ebiten.Image){
	deserttree1, deserttree2, deserttree3,
}

// desert tree type
type deserttree struct {
	display * ebiten.Image
	x, y   float64
}

var backgrounddesertcolor = color.RGBA{225,191,146, 255}

var alldeserttrees [ntrees]deserttree// array of all of the desert trees
var allcactus [ncactus]deserttree

// road images
var nroadslices = 0
const whitespacing = 100
var CurveSize float64 = 0.9
var bezierfunctioninput float64 = 0
var bikeverticalpos float64 = 0.0

var roadslicescount int = 0

var roadwhite * ebiten.Image
var roadgrey  * ebiten.Image

var roadartwork = [](*ebiten.Image) {
	roadwhite, roadgrey,
}

// font for speedometer
var digits = make([](*ebiten.Image), 10, 10)

var imagedata = [10][]byte{
	resources.Zero, resources.One, resources.Two,
	resources.Three, resources.Four, resources.Five,
	resources.Six, resources.Seven, resources.Eight, 
	resources.Nine,
}


// mouse art
var cursorImage * ebiten.Image

// Superbike
var superbike * ebiten.Image
// superbike crash art
var superbikecrash * ebiten.Image

// these are what make up the road. They are 400x1 pixel slices (WidthxHeight)
// these handle jumps as well
type roadartslice struct {
	display * ebiten.Image
	x, y float64
	jump bool
}

// jump image
var jump * ebiten.Image
var jumponscreen bool = false


// A slice of all of the road slices
var allroadslices []roadartslice

func negposRandom() float64 {
	n := 1.0
	if rand.Int() % 2 == 0 {
		n = -1.0
	}
	return float64(rand.Int() % 2) * n

}

func restartGame() {
	for c := 0; c < nroadslices; c++ {
		allroadslices[c].x = float64(roadcenterpos)
	}
	bezierfunctioninput = 0
	clickedtobegin = false
	crashed = false
	bikeposoffset = 0
	mousechangeamount = 0
	GlobalVelocity = 0.0
	bikeinitpos = roadcenterpos + 60
	gamewasjustreset = true

}

// randomizes the postion and image of a desert tree
func reRandomizeDesertTree(tree * deserttree) {
	tree.display = deserttreeimages[rand.Int() % len(deserttreeimages)]
	tree.x = float64(rand.Int() % Width)
	tree.y = float64(-(rand.Int() % Height)) - 50
}

// repositions the road slice vertically for when it goes off screen below
func RePositionRoadSliceVertically(roadslice * roadartslice, repos float64) {
	pos := float64(-Height) + roadsliceheight + repos
	roadslice.y = float64(pos)
}

// Main game struct which gets methods attached to it 
// which ebiten.RunGame calls
type Game struct {
	count int
}


// Curve generation functions which control the
// curves of the road.


func randomBezierfunction(xval, yval float64) func(c float64) float64 {
	beziercurve.p0 = Point{xval + negposRandom(), yval + negposRandom()}
	beziercurve.p1 = Point{xval, yval}
	npy := float64((rand.Int() % 80) + 30)
	npx := float64(rand.Int() % (Width / 8.0)) -float64(Width / 8.0)
	beziercurve.p2 = Point{npx, npy}
	beziercurve.p3 = Point{npx + negposRandom(), npy + negposRandom()}
	return func(c float64) float64 {
		return beziercurve.At(c)
	}
}


var bezierfunction = randomBezierfunction(float64(bikeinitpos), float64(-Height + roadsliceheight))

func closeEnoughToZero(f float64) bool {
	return int(f * 10) == 0
}

func MakeRoadHaveCurves() float64 {
	bezierfunctioninput += 0.02
	return float64(roadcenterpos) + bezierfunction(float64(bezierfunctioninput / beziercurve.p3.y)) * CurveSize


}

func (g * Game) Update(screen * ebiten.Image) error {
	g.count++
	
	var state error = nil
	if clickedtobegin == true && crashed == false {
		if ebiten.IsKeyPressed(ebiten.KeyW) && jumping == false {
			if GlobalVelocity + acceleration <= 1000000 {
				GlobalVelocity += acceleration
		}
	} else if ebiten.IsKeyPressed(ebiten.KeyS) {
		if GlobalVelocity - acceleration >= 0 {
			GlobalVelocity -= acceleration
			}
		}	
		
		if GlobalVelocity >= 5 {
			x, _ := ebiten.CursorPosition()
			mousechangeamount = float64(x) - mouseinitpos
			bikeposoffset += mousechangeamount * turningspeed	
		}

		if ebiten.IsKeyPressed(ebiten.KeySpace) {
			if GlobalVelocity - 1 >= 0 {
				GlobalVelocity -= 1
			}
		}

		bikex, bikey := float64(bikeinitpos) + bikeposoffset, float64(mousecenterposv) + bikeverticalpos
		if bikex > jumppos.x + 40 && bikex < jumppos.x + 80 && bikey > jumppos.y - 50 && bikey < jumppos.y + 50 {
			jumping = true
		}

		if jumping {
			air += 0.01
			scalefactor = 1 + ((GlobalVelocity / 23) * math.Sin((air * 4) / (GlobalVelocity / 30)))
			//fmt.Println(GlobalVelocity)
			if closeEnoughToZero(scalefactor - 1.0) && air > 0.01 || scalefactor - 1.0 < 0.0 {
				scalefactor = 1.0
				air = 0
				jumping = false
			}
		}

	} else if crashed == true {
		GlobalVelocity = 0.0
		if ebiten.IsKeyPressed(ebiten.KeyEnter) {
			restartGame()
			bezierfunction = randomBezierfunction(float64(bikeinitpos), float64(-Height + roadsliceheight))
		}
	}
	

	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		clickedtobegin = true
		x, _ := ebiten.CursorPosition()
		mouseinitpos = float64(x)
	}
	
	if ebiten.IsKeyPressed(ebiten.KeyEscape) {
		state = ExitProgram
	}

	if ebiten.IsKeyPressed(ebiten.KeyU) {
		scalefactor += 0.05
	} else if ebiten.IsKeyPressed(ebiten.KeyJ) {
		scalefactor -= 0.05
	}

	return state
}

// drawingln
func (g * Game) Draw(screen * ebiten.Image) {
	screen.Fill(color.RGBA{225,191,146, 255})

	// trees
	for c := 0; c < ntrees; c++ {
		op := &ebiten.DrawImageOptions{}

		if alldeserttrees[c].y + GlobalVelocity > float64(Height) {
			reRandomizeDesertTree(&alldeserttrees[c])
		} else {
			alldeserttrees[c].y += GlobalVelocity
		}

		op.GeoM = ebiten.TranslateGeo(alldeserttrees[c].x,
		alldeserttrees[c].y)

		screen.DrawImage(alldeserttrees[c].display, op)
	}

	// cactus
	for c := 0; c < ncactus; c++ {
		op := &ebiten.DrawImageOptions{}
		if allcactus[c].y + GlobalVelocity > float64(Height) {
			allcactus[c].x = float64(rand.Int() % Width)
			allcactus[c].y = float64(-(rand.Int() % Height)) - 50
		} else {
			allcactus[c].y += GlobalVelocity
		}
		op.GeoM = ebiten.TranslateGeo(allcactus[c].x, allcactus[c].y)
		screen.DrawImage(allcactus[c].display, op)
	}

	// road slices
	jop := &ebiten.DrawImageOptions{}
	for c := 0; c < nroadslices; c++ {
		op := &ebiten.DrawImageOptions{}
		posi := allroadslices[c].y + GlobalVelocity
		if posi > float64(Height) {
			RePositionRoadSliceVertically(&allroadslices[c],posi - float64(Height))

			if jumponscreen == true && allroadslices[c].jump == true {
				jumponscreen = false
				allroadslices[c].jump = false
			}

			if jumponscreen == false && rand.Int() % 9000 == 1 {
				allroadslices[c].jump = true
				jumponscreen = true
			}

			if  float64(bezierfunctioninput) / beziercurve.p3.y >= 1.0 || gamewasjustreset {
				bezierfunctioninput = 0.0
				if gamewasjustreset {
					// I don't know why this works and what caused the problem.
					beziercurve.p3.x = float64(roadcenterpos - (Width / 2) + 76.0)
				}
				bezierfunction = randomBezierfunction(beziercurve.p3.x, allroadslices[c].y)
				gamewasjustreset = false
			}
			allroadslices[c].x = MakeRoadHaveCurves()
			roadslicescount++
		}else {
			allroadslices[c].y += GlobalVelocity
		}
			
		op.GeoM = ebiten.TranslateGeo(allroadslices[c].x, allroadslices[c].y)
		if allroadslices[c].y >= float64(mousecenterposv) && allroadslices[c].y <= float64(mousecenterposv) + 40.0 {
			//screen.DrawImage(roadartwork[2], op)
			bpos := float64(bikeinitpos) + bikeposoffset  
			if jumping == false && (bpos < allroadslices[c].x - 25 || bpos > allroadslices[c].x + 140) {
				crashed = true
			} 
		} 
		screen.DrawImage(allroadslices[c].display, op)
		if allroadslices[c].jump {
			jumppos.x = allroadslices[c].x
			jumppos.y = allroadslices[c].y
			jop.GeoM = ebiten.TranslateGeo(allroadslices[c].x + 65.0, allroadslices[c].y)
		}
	}

	if jumponscreen {
		screen.DrawImage(jump, jop)
	}

	// speedometer
	currentSpeedStr = strconv.Itoa(int(GlobalVelocity) * 4)
	if len(currentSpeedStr) == 1 {
		currentSpeedStr = "00" + currentSpeedStr
	} else if len(currentSpeedStr) == 2{
		currentSpeedStr = "0" + currentSpeedStr
	}

	for d := 0; d < 3; d++ {
		cd := currentSpeedStr[d] 
		soIndexis, _ := strconv.Atoi(string(cd))
		op := &ebiten.DrawImageOptions{}
		op.GeoM = ebiten.TranslateGeo(40 + float64(d) * 128, float64(Height) - 256)
		screen.DrawImage(digits[soIndexis], op)

	}

	// superbike
	w, h := superbike.Size()
	op := &ebiten.DrawImageOptions{}
	if crashed == true {
		bikeverticalpos += GlobalVelocity
	} else {
		bikeverticalpos = 0
	}
	op.GeoM.Translate(-float64(w / 2) , -float64(h / 2))
	op.GeoM.Rotate(mousechangeamount * 0.007)
	op.GeoM.Scale(scalefactor, scalefactor)
	scalefactor := 1.0
	op.GeoM.Translate(float64(w / 2) * scalefactor, float64(h / 2) * scalefactor)
	op.GeoM.Translate(float64(bikeinitpos) + bikeposoffset, float64(mousecenterposv) + bikeverticalpos)

	if crashed == true {
		screen.DrawImage(superbikecrash, op)
	}else {
		screen.DrawImage(superbike, op)	
	}
	

	x, y := ebiten.CursorPosition()


	// mouse
	if clickedtobegin == false {
		mop := &ebiten.DrawImageOptions{}
		mop.GeoM = ebiten.TranslateGeo(float64(x), float64(y))
		screen.DrawImage(cursorImage, mop)
	}

	ebitenutil.DebugPrint(screen, "FPS: " + strconv.FormatFloat(ebiten.CurrentFPS(), 'f', 3, 64))

}

func (g * Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return Width, Height
}

func main() {
	// seed random
	rand.Seed(time.Now().UnixNano())

	// loading images (desert trees)
	img, _, err1 := image.Decode(bytes.NewReader(resources.DesertTree1))
	if err1 != nil {
		log.Fatal(err1)
	}

	img2, _, err2 := image.Decode(bytes.NewReader(resources.DesertTree2))
	if err2 != nil {
		log.Fatal(err2)
	}

	img3, _, err5 := image.Decode(bytes.NewReader(resources.DesertTree3))
	if err5 != nil {
		log.Fatal(err5)
	}

	// errors for desert trees below
	err3 := error(nil)
	err4 := error(nil)
	err6 := error(nil)

	// errors for road artwork below
	err7 := error(nil)
	err8 := error(nil)
	//err10 := error(nil)


	// setting images (desert trees)
	deserttreeimages[0], err3 = ebiten.NewImageFromImage(img, 
		ebiten.FilterDefault)
	if err3 != nil {
		log.Fatal(err3)
	} 
	deserttreeimages[1], err4 = ebiten.NewImageFromImage(img2, ebiten.FilterDefault)
	if err4 != nil {
		log.Fatal(err4)
	}

	deserttreeimages[2], err6 = ebiten.NewImageFromImage(img3, ebiten.FilterDefault)
	if err6 != nil {
		log.Fatal(err6)
	}

	// loading images road
	roadgreyimg, _, greyerr := image.Decode(bytes.NewReader(resources.RoadGrey))
	if greyerr != nil {
		log.Fatal(greyerr)
	}

	roadwhiteimg, _, whiteerr := image.Decode(bytes.NewReader(resources.RoadWhite))
	if whiteerr != nil {
		log.Fatal(whiteerr)
	}

	// setting road artwork
	roadartwork[0], err7 = ebiten.NewImageFromImage(roadgreyimg, ebiten.FilterDefault)
	if err7 != nil {
		log.Fatal(err7)
	}
	roadartwork[1], err8 = ebiten.NewImageFromImage(roadwhiteimg, ebiten.FilterDefault)
	if err8 != nil {
		log.Fatal(err8)
	}

	// numbers
	for num := 0; num < 10; num++ {
		digit, _, digiterr := image.Decode(bytes.NewReader(imagedata[num]))
		if digiterr != nil {
			log.Fatal(digiterr)
		}
		digits[num], digiterr = ebiten.NewImageFromImage(digit, ebiten.FilterDefault)
		if digiterr != nil {
			log.Fatal(digiterr)
		}
	}

	// mouse
	mouseart, _, mousearterr := image.Decode(bytes.NewReader(resources.Mouseart))
	if mousearterr != nil {
		log.Fatal(mousearterr)
	}
	cursorImage, mousearterr = ebiten.NewImageFromImage(mouseart, ebiten.FilterDefault)
	if mousearterr != nil {
		log.Fatal(mousearterr)
	}

	nroadslices = (2 * Height) / roadsliceheight
	mousecenterposv = (Height / 2) + 120


	// cactus
	cactusimg, _, cactuserr := image.Decode(bytes.NewReader(resources.Cactus))
	if cactuserr != nil {
		log.Fatal(cactuserr)
	}
	cactus, cactuserr = ebiten.NewImageFromImage(cactusimg, ebiten.FilterDefault)
	if cactuserr != nil {
		log.Fatal(cactuserr)
	}

	for c := 0; c < ntrees; c++ {
		alldeserttrees[c] = deserttree{deserttreeimages[rand.Int() % len(deserttreeimages)], float64(rand.Int() % Width), 
		float64(rand.Int() % Height)}
		
	}

	for c := 0; c < ncactus; c++ {
		allcactus[c] = deserttree{cactus, float64(rand.Int() % Width), float64(rand.Int() % Height)}
	}

	// superbike
	bikeimg, _, bikerr := image.Decode(bytes.NewReader(resources.Superbike))
	if bikerr != nil {
		log.Fatal(bikerr)
	}
	superbike, bikerr = ebiten.NewImageFromImage(bikeimg, ebiten.FilterDefault)
	if bikerr != nil {
		log.Fatal(bikerr)
	}

	bcrashimg, _, bcerr := image.Decode(bytes.NewReader(resources.Superbikecrash))
	if bcerr != nil {
		log.Fatal(bcerr)
	}
	superbikecrash, bcerr = ebiten.NewImageFromImage(bcrashimg, ebiten.FilterDefault)
	if bcerr != nil {
		log.Fatal(bcerr)
	}

	// jump
	jumpimg, _, jerr := image.Decode(bytes.NewReader(resources.Jump))
	if jerr != nil {
		log.Fatal(jerr)
	}
	jump, jerr = ebiten.NewImageFromImage(jumpimg, ebiten.FilterDefault)
	if jerr != nil {
		log.Fatal(jerr)
	}

	startat := -Height + roadsliceheight
	allroadslices = make([]roadartslice, nroadslices, nroadslices)
	var whichslice int
	bikeinitpos = roadcenterpos + 60
	for c := 0; c < nroadslices; c++ {
		if c % whitespacing == 0 {
			if whichslice == 1 {
				whichslice = 0
			} else if whichslice == 0 {
				whichslice = 1
			}
		}
		allroadslices[c] = roadartslice{
			roadartwork[whichslice], float64(roadcenterpos),
			float64(startat + (float64(c) * roadsliceheight)), 
			false, 
		}
	}
	ebiten.SetFullscreen(true)
	//ebiten.SetWindowTitle("Game")
	//ebiten.SetWindowSize(Width, Height)
	ebiten.SetCursorVisible(false)
	if err := ebiten.RunGame(&Game{}); err != nil && err != ExitProgram {
		log.Fatal(err)
	}
}