package main

//
//Copyright 2018 Telenor Digital AS
//
//Licensed under the Apache License, Version 2.0 (the "License");
//you may not use this file except in compliance with the License.
//You may obtain a copy of the License at
//
//http://www.apache.org/licenses/LICENSE-2.0
//
//Unless required by applicable law or agreed to in writing, software
//distributed under the License is distributed on an "AS IS" BASIS,
//WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//See the License for the specific language governing permissions and
//limitations under the License.
//

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io/ioutil"
	"net/http"
	"os"
	"runtime"
	"time"
)

var params struct {
	imeiOne      int64
	imeiTwo      int64
	host         string
	username     string
	password     string
	screenWidth  int
	screenHeight int
	identifier1  string
	identifier2  string
	offset1      image.Point
	offset2      image.Point
	logoffset    image.Point
}

const (
	logo1Dim    = "gfx/telenor_dim.png"
	logo1Bright = "gfx/telenor_col.png"
	logo2Dim    = "gfx/ntnu_dim.png"
	logo2Bright = "gfx/ntnu_col.png"
	finalLogo   = "gfx/iotp_logo.png"
)

var images struct {
	image1Dim    image.Image
	image1Bright image.Image
	image2Dim    image.Image
	image2Bright image.Image
	finalLogo    image.Image
	white        *image.RGBA
}

func init() {
	flag.Int64Var(&params.imeiOne, "imei1", int64(11223344), "First IMEI to listen on")
	flag.Int64Var(&params.imeiTwo, "imei2", int64(55667788), "Second IMEI to listen on")
	flag.StringVar(&params.host, "host", "horde.example.com", "Horde server name")
	flag.StringVar(&params.username, "username", "test", "User name for Horde service")
	flag.StringVar(&params.password, "password", "test", "Password for Horde service")
	flag.IntVar(&params.screenHeight, "height", 984, "Screen height")
	flag.IntVar(&params.screenWidth, "width", 1824, "Screen width")
	flag.StringVar(&params.identifier1, "id1", "Telenor", "Identifier # 1 to listen for")
	flag.StringVar(&params.identifier2, "id2", "NTNU", "Identifier # 2 to listen for")
	flag.IntVar(&params.offset1.X, "x1-offset", 118, "X offset for image 1")
	flag.IntVar(&params.offset1.Y, "y1-offset", 378, "Y offset for image 1")
	flag.IntVar(&params.offset2.X, "x2-offset", 1015, "X offset for image 2")
	flag.IntVar(&params.offset2.Y, "y2-offset", 471, "Y offset for image 2")
	flag.IntVar(&params.logoffset.X, "logo-x-offset", 356, "X offset for logo")
	flag.IntVar(&params.logoffset.Y, "logo-y-offset", 334, "Y offset for logo")
	flag.Parse()
}

// loadFile loads a PNG file from disk into memory
func loadFile(imagefile string) (image.Image, error) {
	fileinfo, err := os.Lstat(imagefile)
	if err != nil {
		return nil, errors.New("unable to stat image file")
	}
	buf := make([]byte, fileinfo.Size())

	file, err := os.Open(imagefile)
	if err != nil {
		return nil, errors.New("unable to open image file")
	}
	defer file.Close()
	n, err := file.Read(buf)
	if err != nil {
		return nil, errors.New("unable to read image file")
	}

	img, err := png.Decode(bytes.NewReader(buf[0:n]))
	if err != nil {
		return nil, errors.New("unable to decode image file")
	}
	return img, nil
}

// loadImages preloads images into memory and creates an all white image to write on
func loadImages() error {
	var err error
	if images.image1Dim, err = loadFile(logo1Dim); err != nil {
		return err
	}
	if images.image1Bright, err = loadFile(logo1Bright); err != nil {
		return err
	}
	if images.image2Dim, err = loadFile(logo2Dim); err != nil {
		return err
	}
	if images.image2Bright, err = loadFile(logo2Bright); err != nil {
		return err
	}
	if images.finalLogo, err = loadFile(finalLogo); err != nil {
		return err
	}
	images.white = image.NewRGBA(image.Rect(0, 0, params.screenWidth, params.screenHeight))
	for x := 0; x < params.screenWidth; x++ {
		for y := 0; y < params.screenHeight; y++ {
			images.white.Set(x, y, color.White)
		}
	}
	return nil
}

var rgbaImage *image.RGBA

// setupBuffer creates a new image buffer
func setupBuffer() {
	rgbaImage = image.NewRGBA(image.Rect(0, 0, params.screenWidth, params.screenHeight))
}

// clearBuffer starts a new screen (without writing)
func clearBuffer() {
	draw.Draw(rgbaImage, rgbaImage.Rect, images.white, rgbaImage.Rect.Min, draw.Over)
}

// drawImage draws an image to the screen buffer
func drawImage(i image.Image, pos image.Point) {
	draw.Draw(rgbaImage, image.Rect(pos.X, pos.Y, i.Bounds().Max.X+pos.X, i.Bounds().Max.Y+pos.Y), i, i.Bounds().Min, draw.Src)
}

// showBuffer dumps the output to the screen
func showBuffer() {
	var fbbuf []byte
	if runtime.GOOS != "linux" {
		return
	}
	const fbDevice = "/dev/fb0"
	if fbbuf == nil {
		fbbuf = make([]byte, params.screenWidth*params.screenHeight*4)
	}
	pos := 0
	for y := 0; y < params.screenHeight; y++ {
		for x := 0; x < params.screenWidth; x++ {
			r, g, b, a := rgbaImage.At(x, y).RGBA()
			fbbuf[pos] = byte(b & 0xFF)
			pos++
			fbbuf[pos] = byte(g & 0xFF)
			pos++
			fbbuf[pos] = byte(r & 0xFF)
			pos++
			fbbuf[pos] = byte(a & 0xFF)
			pos++
		}
	}
	if err := ioutil.WriteFile(fbDevice, fbbuf[0:pos], 0600); err != nil {
		fmt.Println("Error writing to frame buffer: ", err)
	}
}

// listenForData checks the backend for new data
func listenForData(address string) (<-chan string, error) {
	ret := make(chan string)
	go func() {
		req, err := http.NewRequest("GET", address, nil)
		if err != nil {
			fmt.Println("Error creating request for ", address, ": ", err)
			return
		}

		req.SetBasicAuth(params.username, params.password)

		lastTimestamp := int64(-1)
		for {
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				fmt.Println("Error creating request for ", address, ": ", err)
				return
			}

			var dataMessage struct {
				IMEI      string `json:"imei"`
				Payload   []byte `json:"payload"`
				Timestamp int64  `json:"timestamp"`
			}

			json.NewDecoder(resp.Body).Decode(&dataMessage)
			if resp.StatusCode == http.StatusOK {
				if lastTimestamp != -1 && lastTimestamp != dataMessage.Timestamp {
					fmt.Printf("Got new data message for %s: %s\n", dataMessage.IMEI, string(dataMessage.Payload))
					ret <- string(dataMessage.Payload)
				}
				if lastTimestamp != dataMessage.Timestamp {
					lastTimestamp = dataMessage.Timestamp
					fmt.Printf("Timestamp for %s updated to %d\n", address, lastTimestamp)

				}
			}
			time.Sleep(time.Second)
		}
	}()
	return ret, nil
}

func main() {
	fmt.Println("Loading images...")
	if err := loadImages(); err != nil {
		fmt.Printf("Got error loading images: %v\n", err)
		return
	}
	setupBuffer()

	clearBuffer()
	drawImage(images.image1Dim, params.offset1)
	drawImage(images.image2Dim, params.offset2)
	showBuffer()

	var ch1, ch2 <-chan string
	var err error
	fmt.Println("Waiting for devices to come online...")
	for ch1 == nil || ch2 == nil {
		ch1, err = listenForData(fmt.Sprintf("http://%s/devices/%d/latest", params.host, params.imeiOne))
		if err != nil {
			fmt.Printf("Error listening on websocket: %v\n", err)
		} else {
			fmt.Println("Device 1 is online!")
		}
		ch2, err = listenForData(fmt.Sprintf("http://%s/devices/%d/latest", params.host, params.imeiTwo))
		if err != nil {
			fmt.Printf("Error listening on websocket: %v\n", err)
		} else {
			fmt.Println("Device 2 is online!")
		}
		if ch1 == nil || ch2 == nil {
			time.Sleep(5 * time.Second)
		}
	}

	var image1 = images.image1Dim
	var image2 = images.image2Dim

	update := false
	for {
		select {
		case msg := <-ch1:
			if msg == params.identifier1 {
				image1 = images.image1Bright
				update = true
			}

		case msg := <-ch2:
			if msg == params.identifier2 {
				image2 = images.image2Bright
				update = true
			}
		default:
			time.Sleep(100 * time.Millisecond)
		}
		if update {
			clearBuffer()
			drawImage(image1, params.offset1)
			drawImage(image2, params.offset2)
			showBuffer()
			if image1 == images.image1Bright && image2 == images.image2Bright {
				time.Sleep(5 * time.Second)
				clearBuffer()
				drawImage(images.finalLogo, params.logoffset)
				showBuffer()
				<-time.After(1 * time.Hour)
			}
			update = false
		}
	}

}
