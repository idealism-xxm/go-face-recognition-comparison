package recognizer

import (
	"go-face-recognition-demo/common/recognizer"
	"os/exec"
	"strconv"
	"strings"
)

type FaceRecognitionRunningCmd struct{}

func NewFaceRecognitionRunningCmd() *FaceRecognitionRunningCmd {
	return &FaceRecognitionRunningCmd{}
}

func (r *FaceRecognitionRunningCmd) DetectFace(imagePath string) (faceLocations []*recognizer.FaceLocation, err error) {
	// 1. 执行对应的 cmd 命令，获取输出结果（每一行为一个人脸位置）
	output, err := exec.Command("face_detection", imagePath).Output()
	//output, err := exec.Command("face_detection", "--model", "cnn", imagePath).Output()
	if err != nil {
		return nil, err
	}

	// 2. 遍历所有 人脸位置，并转换成 []*recognizer.FaceLocation
	for _, line := range strings.Split(string(output), "\n") {
		// 每一行的数据形式： /github.com/idealism-xxm/images/1.jpg,362,266,448,179
		parts := strings.Split(line, ",")
		if len(parts) < 5 {
			continue
		}

		// 将位置信息转换成 int 类型
		top, err := strconv.Atoi(parts[1])
		if err != nil {
			return nil, err
		}
		right, err := strconv.Atoi(parts[2])
		if err != nil {
			return nil, err
		}
		bottom, err := strconv.Atoi(parts[3])
		if err != nil {
			return nil, err
		}
		left, err := strconv.Atoi(parts[4])
		if err != nil {
			return nil, err
		}

		// 每一个人脸位置信息转换成 FaceLocation ，并放入切片
		faceLocations = append(faceLocations, &recognizer.FaceLocation{
			Top:    top,
			Right:  right,
			Bottom: bottom,
			Left:   left,
		})
	}

	return faceLocations, nil
}
