package recognizer

type FaceLocation struct {
	Top, Right, Bottom, Left int
}

type FaceRecognition interface {
	DetectFace(imagePath string) (faceLocations []*FaceLocation, err error)
}
