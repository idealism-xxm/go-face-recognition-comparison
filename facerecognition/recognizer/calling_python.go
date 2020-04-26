package recognizer

import (
	"github.com/DataDog/go-python3"
	"go-face-recognition-demo/common/recognizer"
)

type FaceRecognitionCallingPython struct {
	pyLoadImageFile *python3.PyObject
	pyFaceLocations *python3.PyObject
}

func NewFaceRecognitionCallingPython() *FaceRecognitionCallingPython {
	// 1. 初始化 python
	python3.Py_Initialize()

	// 2. 导入 face_recognition 包
	pyFaceRecognition := python3.PyImport_ImportModule("face_recognition")
	defer pyFaceRecognition.DecRef()

	// 3. 获取需要的方法并进行保存，方便后续处理
	return &FaceRecognitionCallingPython{
		pyLoadImageFile: pyFaceRecognition.GetAttrString("load_image_file"),
		pyFaceLocations: pyFaceRecognition.GetAttrString("face_locations"),
	}
}

func (r *FaceRecognitionCallingPython) DetectFace(imagePath string) (faceLocations []*recognizer.FaceLocation, err error) {
	// 1. imagePath 转换成 python 的类型
	pyFilepath := python3.PyUnicode_FromString(imagePath)
	defer pyFilepath.DecRef()

	// 2. 调用 local_image_file 方法，获得 图片 实例
	pyImage := r.pyLoadImageFile.CallFunctionObjArgs(pyFilepath)
	defer pyImage.DecRef()

	// 3. 调用 face_locations 方法，获得图片中的 人脸位置列表 实例
	pyLocations := r.pyFaceLocations.CallFunctionObjArgs(pyImage)
	//pyNumber := python3.PyLong_FromGoInt(1)
	//defer pyNumber.DecRef()
	//pyModel := python3.PyUnicode_FromString("cnn")
	//defer pyModel.DecRef()
	//pyLocations := r.pyFaceLocations.CallFunctionObjArgs(pyImage, pyNumber, pyModel)
	defer pyLocations.DecRef()

	// 4. 遍历所有 人脸位置，并转换成 []*recognizer.FaceLocation
	for i := python3.PyList_Size(pyLocations) - 1; i >= 0; i-- {
		pyLocation := python3.PyList_GetItem(pyLocations, i)
		defer pyLocation.DecRef()

		// 每一个人脸位置信息转换成 FaceLocation ，并放入切片
		faceLocations = append(faceLocations, &recognizer.FaceLocation{
			Top:    python3.PyLong_AsLong(python3.PyTuple_GetItem(pyLocation, 0)),
			Right:  python3.PyLong_AsLong(python3.PyTuple_GetItem(pyLocation, 1)),
			Bottom: python3.PyLong_AsLong(python3.PyTuple_GetItem(pyLocation, 2)),
			Left:   python3.PyLong_AsLong(python3.PyTuple_GetItem(pyLocation, 3)),
		})
	}

	return faceLocations, nil
}
