**不关心具体的实现可以跳转至文末看对比。**

#### 背景

最近有一个需求需要用到人脸识别技术，首先去云服务厂商看了相关的服务，发现定价太贵了，无论选择怎么付费都很不划算，所以就考虑使用开源项目，并封装一下形成微服务以供调用。

考虑到我们将部分原有的服务从 `Python` 迁移到 `Go` 后，在巨幅减少服务器的情况下仍能够扛住相同的流量，并且被告知 `Python` 的 `gRPC` 不太好（的确如此，生成的代码竟然还需要手动修改才能使用），所以我们决定使用 `Go` 提供微服务。

以此为前提调研了一些符合我们需求的库：

- [face_recognition](https://github.com/ageitgey/face_recognition): `Python` 的库，支持 `Python` 直接调用和命令行调用，支持 jpg 和 png 格式， 33.8k Star, 9.4k Fork 。
- [go-face](https://github.com/Kagami/go-face): `Go` 的库，支持 `Go` 直接调用，仅支持 jpg 格式， 400 Star, 70 Fork 。

因此产生了几种方法：

- ~~使用 `face_recognition` ，先使用 `Python` 封装一个 `http` 服务，再使用 `Go` 调用~~ （过于麻烦，代码分割在两处，不好维护，且性能堪忧，直接排除）
- 使用 `face_recognition` ， `Go` 使用命令行调用
- 使用 `face_recognition` ， `Go` 使用 `cgo` ，通过 `C` 调用 `Python` 的 api 
- 使用 `go-face` ， `Go` 直接调用 `Go` 的 api

#### 准备

考虑到人脸识别服务可能会由于性能、识别率等原因而更换其他库，所以需要将人脸识别服务的接口封装一下，提供一个公用接口，方便后续直接替换实现，使下游无感知。

调研时仅简单定义一下人脸检测的接口及其返回值：

```go
type FaceLocation struct {
	Top, Right, Bottom, Left int
}

type FaceRecognition interface {
	DetectFace(imagePath string) (faceLocations []*FaceLocation, err error)
}
```

并进行简单的性能测试，我们对同一张图片进行 `10` 次人脸识别：

```go
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
```

#### 使用 `face_recognition`

调研时直接使用 `Docker` 在镜像里测试，既可以避免环境不同而产生的差异，又可以提前研究配置方式并踩坑。

由于需要装的软件很多且比较大，所以先打一个 `base` 镜像，包含所需要的所有依赖，这样我们在更改代码的时候，直接用  `base` 镜像编译即可，可以大幅提高调试和测试的速度。

`base` 镜像的 `Dockerfile`

```dockerfile
# docker build -t facerecognition:debug-base -f facerecognition/Dockerfile.debug.base .

FROM golang:1.14.2

WORKDIR /github.com/idealism-xxm/

COPY go.mod .
COPY go.sum .

RUN go mod download
RUN apt-get update && \
  apt-get install -y cmake python3-pip && \
  pip3 install face_recognition
```

`debug` 镜像的 `Dockerfile`

```dockerfile
# docker build -t facerecognition:debug -f facerecognition/Dockerfile.debug --build-arg repository=facerecognition:debug-base .

# format -> name:tag
ARG repository
FROM $repository

WORKDIR /github.com/idealism-xxm/
COPY . .

RUN mkdir bin
RUN CGO_ENABLED=1 GOOS=linux go build -v -o bin go-face-recognition-demo/facerecognition/...

CMD ["/github.com/idealism-xxm/bin/facerecognition"]
```

##### 命令行调用

使用 `exec.Command` 直接调用即可，不过 `face_recognition` 提供的可执行文件本质上还是 `Python` ，所以需要执行 `import face_recognition` ，而这一行命令比较耗时，所以会大幅拖慢一张图片进行识别的速度。

```go
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
```

可以看到命令行调用比较简单，几行就能完成，但由于返回结果是 `[]byte` 类型，所以需要大量代码转换成返回类型。而且我们并不知道出错时会输出什么，所以对错误处理可能无法太完善，可能在极端情况下返回了错误的数据。

经过简单的测试后得出以下数据：

| | 识别人脸数 | 识别 `10` 次耗时 (s) |
| --- | --- | --- |
| face-recognition (cmd - hog) | 2 | 10 |
| face-recognition (cmd - cnn) | 4 | 20 |

##### 通过 `C` 调用 `Python`

最开始直接在代码里使用原生方式调用，由于没有类型所以经常出错，而且只能等到编译时候才能发现各种问题，并且这样不便于后期维护，所以需要使用类型安全的方式。发现了 [go-python3](https://github.com/DataDog/go-python3) 这个库将 `Python` 的 api 包装了一层，这样我们在业务代码中的所有类型都可以进行推导，可以直接避免大量问题。但是由于我们通过 `C` 进行调用，所以代码没写好就非常容易导致内存泄漏或者空指针异常等问题。

```go
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
			Top: python3.PyLong_AsLong(python3.PyTuple_GetItem(pyLocation, 0)),
			Right: python3.PyLong_AsLong(python3.PyTuple_GetItem(pyLocation, 1)),
			Bottom: python3.PyLong_AsLong(python3.PyTuple_GetItem(pyLocation, 2)),
			Left: python3.PyLong_AsLong(python3.PyTuple_GetItem(pyLocation, 3)),
		})
	}

	return faceLocations, nil
}
```

可以看到通过 `C` 调用 `Python` 时，每次都需要对得到的对象执行 `.DecRef()` 以免内存泄漏。而这个 `.DecRef()` 又很迷，最开始我对每一个获得的对象都执行了 `.DecRef()` ，就会导致 `panic` ，估计是释放内存会同时释放内部的对象造成的。

经过简单的测试后得出以下数据：

| | 识别人脸数 | 识别 `10` 次耗时 (s) |
| --- | --- | --- |
| face-recognition (python - hog) | 3 | 0.76 |
| face-recognition (python - cnn) | 4 | 37.5 |

#### 使用 `go-face`

我们同样使用两个镜像， `base` 镜像包含所有依赖。

`base` 镜像的 `Dockerfile`

```dockerfile
# docker build -t goface:debug-base -f goface/Dockerfile.debug.base .

FROM golang:1.14.2

WORKDIR /github.com/idealism-xxm/

COPY go.mod .
COPY go.sum .

RUN go mod download
RUN apt-get update && \
  apt-get install -y libdlib-dev libopenblas-dev libjpeg62-turbo-dev bzip2

RUN mkdir -p /usr/local/lib/pkgconfig
RUN CONFIG_PATH="/usr/local/lib/pkgconfig/dlib-1.pc" && \
  libdir="/usr/lib/x86_64-linux-gnu" && \
  includedir="/usr/include" && \
  echo "Name: dlib" >> $CONFIG_PATH && \
  echo "Description: Numerical and networking C++ library" >> $CONFIG_PATH && \
  echo "Version: 19.10.0" >> $CONFIG_PATH && \
  echo "Libs: -L${libdir} -ldlib -lblas -llapack" >> $CONFIG_PATH && \
  echo "Cflags: -I${includedir}" >> $CONFIG_PATH && \
  echo "Requires:" >> $CONFIG_PATH
RUN mkdir models && cd models && \
  wget https://github.com/davisking/dlib-models/raw/master/shape_predictor_5_face_landmarks.dat.bz2 && \
  bunzip2 shape_predictor_5_face_landmarks.dat.bz2 && \
  wget https://github.com/davisking/dlib-models/raw/master/dlib_face_recognition_resnet_model_v1.dat.bz2 && \
  bunzip2 dlib_face_recognition_resnet_model_v1.dat.bz2 && \
  wget https://github.com/davisking/dlib-models/raw/master/mmod_human_face_detector.dat.bz2 && \
  bunzip2 mmod_human_face_detector.dat.bz2
```

`debug` 镜像的 `Dockerfile`

```dockerfile
# docker build -t goface:debug -f goface/Dockerfile.debug --build-arg repository=goface:debug-base .

# format -> name:tag
ARG repository
FROM $repository

WORKDIR /github.com/idealism-xxm/
COPY . .

RUN mkdir bin
RUN CGO_ENABLED=1 GOOS=linux go build -v -o bin go-face-recognition-demo/goface/...

CMD ["/github.com/idealism-xxm/bin/goface"]
```

##### 直接调用

`go-face` 直接提供了各种人脸检测和识别的函数，方便直接在 `Go` 中处理，由于没有在 `Go` 中与 `Python` 交互，所以能较为安全的进行人脸识别，但是编译时较慢（不包括下载依赖）。

```go
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
			Top: curFace.Rectangle.Min.Y,
			Right: curFace.Rectangle.Max.X,
			Bottom: curFace.Rectangle.Max.Y,
			Left: curFace.Rectangle.Min.X,
		})
	}

	return faceLocations, nil
}
```

可以发现 `go-face` 调用非常方便，我们只需提取所需的信息进行转换即可，不过需要注意是它仅支持 jpg 格式，并且目前没有太多人关注，已经长时间未更新。

经过简单的测试后得出以下数据：

| | 识别人脸数 | 识别 `10` 次耗时 (s) |
| --- | --- | --- |
| go-face (hog) | 2 | 0.51 |
| go-face (cnn) | 4 | 4 |

#### 总结

电脑配置： i9-9900K, 无独立显卡 

操作系统： macOS Mojave

测试图片：

![1.png](images/1.jpg)

测试结果及体验如下：

| | 识别人脸数 | 识别 `10` 次耗时 (s) | 支持图片格式 | 镜像包大小 (GB) | 编译速度 | 是否容易出错 |
| --- | --- | --- | --- | --- | --- | --- |
| face-recognition (cmd - hog) | 2 | 10 | jpg,png | 1.49 | 较快 | √ |
| face-recognition (cmd - cnn) | 4 | 20 | jpg,png | 1.49 |  较快 | √ |
| face-recognition (python - hog) | 3 | 0.76 | jpg,png | 1.49 | 较快 | √ |
| face-recognition (python - cnn) | 4 | 37.5 | jpg,png | 1.49 | 较快 | √ |
| go-face (hog) | 2 | 0.51 | jpg | 1.17 | 较慢 | ✕ |
| go-face (cnn) | 4 | 4 | jpg | 1.17 | 较慢 | ✕ |

目前只是针对人脸检测进行了对比，如果只需要人脸检测的话，有其他的 纯 `Go` 实现的库能够达到更快的检测速度。后续还需要对人脸识别进行对比，以便最终确定使用哪个库提供服务。
