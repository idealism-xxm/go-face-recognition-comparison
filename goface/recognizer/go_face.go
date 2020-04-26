package recognizer

import (
	"github.com/Kagami/go-face"
	"go-face-recognition-demo/common/recognizer"
)

const modelPath = "/github.com/idealism-xxm/models"

type FaceRecognitionGoFace struct {
	rec *face.Recognizer
}

func NewFaceRecognitionGoFace() *FaceRecognitionGoFace {
	// 创建一个 go-face 的 recognizer
	rec, err := face.NewRecognizer(modelPath)
	if err != nil {
		panic("can not new recognizer")
	}
	return &FaceRecognitionGoFace{
		rec: rec,
	}
}

func (r *FaceRecognitionGoFace) DetectFace(imagePath string) (faceLocations []*recognizer.FaceLocation, err error) {
	// 1. 调用 go-face 的识别方法，获得图片中的人脸位置切片
	faces, err := r.rec.RecognizeFile(imagePath)
	//faces, err := r.rec.RecognizeFileCNN(imagePath)
	if err != nil {
		return nil, err
	}

	// 2. 将人脸信息转换成 []*recognizer.FaceLocation
	for _, curFace := range faces {
		faceLocations = append(faceLocations, &recognizer.FaceLocation{
			Top:    curFace.Rectangle.Min.Y,
			Right:  curFace.Rectangle.Max.X,
			Bottom: curFace.Rectangle.Max.Y,
			Left:   curFace.Rectangle.Min.X,
		})
	}

	return faceLocations, nil
}
