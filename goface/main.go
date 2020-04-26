package main

import (
	"fmt"
	commonrec "go-face-recognition-demo/common/recognizer"
	"go-face-recognition-demo/goface/recognizer"
	"time"
)

const MaxIteration = 10
const ImagePath = "/github.com/idealism-xxm/images/1.jpg"

func main() {
	test(recognizer.NewFaceRecognitionGoFace())
}

func test(faceRecognition commonrec.FaceRecognition) {
	start := time.Now().UnixNano()
	for i := 0; i < MaxIteration; i++ {
		faceLocations, err := faceRecognition.DetectFace(ImagePath)
		fmt.Println(i, len(faceLocations), err)
		if i == 0 {
			for _, faceLocation := range faceLocations {
				fmt.Println(faceLocation.Top, faceLocation.Right, faceLocation.Bottom, faceLocation.Left)
			}
		}
	}
	fmt.Println(time.Now().UnixNano() - start)
}
