package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"time"
)

const (
	//	ENCODER = "C:\Users\nkark\Downloads\ffmpeg-2022-02-24-git-8ef03c2ff1-full_build\bin\ffmpeg"
	//	ENC_INPUT_PARAM = " -re -framerate 30 -thread_queue_size 10 -start_number 1 "
	//	ENC_OUTPUT_PARAM = " -codec:v libx264 -pix_fmt:v gray   -filter:v fps=24 -b 8M -f \"mpegts\" "4
	FILE_EXTN = ".png"
)

type imageSequence struct {
	path     string
	fileName string
	data     []byte
	seq      int
}

var images map[int]imageSequence
var debugSet bool

// Frame Rate denotes the number of files that will be present in the output dir
// At the frame Rate new files will be inserted and older ones removed
// 30 fps indicates that files were be copied over every 33 msec.
// At the end of input files, they will be looped over

func main() {
	var wg sync.WaitGroup
	var defInputPtr, defOutoutPtr, ffmpegCmd string
	if runtime.GOOS == "windows" {
		defInputPtr = "z:\\images"
		defOutoutPtr = "z:\\loopImages"
		ffmpegCmd = "C:\\Users\\nkark\\Downloads\\ffmpeg-2022-02-24-git-8ef03c2ff1-full_build\\bin\\ffmpeg"
	} else {
		defInputPtr = "/Users/nkarkhan/Documents/GitHub/FrameRateTest/images"
		defOutoutPtr = "/Users/nkarkhan/Documents/GitHub/FrameRateTest/loopImages"
		ffmpegCmd = "ffmpeg"
	}
	inputPtr := flag.String("input", defInputPtr, "Input Source for images")
	d := flag.Bool("debug", false, "Debug tracing")
	pipe := flag.Bool("pipe", false, "pipe to ffmpeg directly")
	flag.Parse()
	debugSet = *d
	outputPtr := flag.String("output", defOutoutPtr, "Output Dir for images")
	frameRatePtr := flag.Int("frameRate", 30, "Frame Rate in Hz")
	images = make(map[int]imageSequence)
	iterate(*outputPtr, true) //clean output files
	iterate(*inputPtr, false)
	log.Println("Number of files ", len(images))
	for index := 1; index <= len(images); index++ {
		if debugSet {
			log.Println("index %d - Image %v", index, images[index])
		}
	}
	wg.Add(1)
	if *pipe {
		fmt.Println("Running the camera stream")
		//C:\Users\nkark\Downloads\ffmpeg-2022-02-24-git-8ef03c2ff1-full_build\bin>ffmpeg -framerate 30 -thread_queue_size 20480 -start_number 1 -i "z:\loopImages\loop%d.png" -codec:v mpeg4    -preset ultrafast  -f mpegts udp://127.0.0.1:5555 -loglevel info

		ffmpegCmd := exec.Command(ffmpegCmd,
			"-loglevel", "debug", "-re", "-f", "image2pipe", "-thread_queue_size", "2048", "-i", "-",
			"-codec:", "v", "mpeg4", "-f", "mpegts", "udp:", "////127.0.0.1:5555",
			"")

		ffpmegStdIn, err := ffmpegCmd.StdinPipe()
		if err != nil {
			return
		}
		defer ffpmegStdIn.Close()
		ffpmegStdOut, err := ffmpegCmd.StderrPipe()
		if err != nil {
			return
		}
		defer ffpmegStdOut.Close()
		err = ffmpegCmd.Start()

		if err != nil {
			log.Fatal(err)
		}
		go func() {
			defer wg.Done()
			stdinWriter := bufio.NewWriter(ffpmegStdIn)
			populateffmpegPipe(*frameRatePtr, stdinWriter, ffpmegStdOut)
		}()
	} else {
		go func() {
			defer wg.Done()
			populateOutput(*frameRatePtr, *outputPtr)
		}()
	}
	wg.Wait()

}
func iterate(path string, del bool) {
	filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			c, _ := os.Getwd()
			fmt.Printf(c)
			log.Fatalf(err.Error() + path)
		}
		var imgData imageSequence
		imgData.path = path
		imgData.fileName = info.Name()
		f := info.Name()
		if len(f) <= 4 {
			return nil
		}
		f = f[:len(f)-4]
		seq, err := strconv.Atoi(f)
		for (err != nil) && (len(f) > 0) {
			f = f[1:]
			seq, err = strconv.Atoi(f)
		}
		if err == nil {
			if del {
				os.Remove(imgData.path)
			} else {
				imgData.seq = seq
				imgData.data, _ = ioutil.ReadFile(imgData.path)
				images[seq] = imgData
			}
		}

		return nil
	})
}

func populateffmpegPipe(frameRate int, filePtr *bufio.Writer, ffmpegout io.ReadCloser) {
	updateTime := 1000 / (frameRate + 3) // in msec, at every updateTime we add a file to ffmpeg stdin
	inputIdx := 1
	for {
		time.Sleep(time.Millisecond * time.Duration(updateTime))
		source, sourceError := os.Open(images[1].path)
		if sourceError != nil {
			log.Fatal(sourceError)
		}
		defer source.Close()
		_, copyError := io.Copy(filePtr, source)
		if copyError != nil {
			var o []byte
			fmt.Println("Error in writing to ffmpeg %s", images[inputIdx].path)

			ffmpegout.Read(o)
			fmt.Println("stdout - %s", o)
			log.Fatal(copyError)
		}
		inputIdx = (inputIdx)%len(images) + 1

	}
}

func populateOutput(frameRate int, outputDir string) {
	updateTime := 1000 / (frameRate + 3) // in msec, at every updateTime we add a file to output and remove 1
	outputIdx := 1
	inputIdx := 1
	for {
		if outputIdx > frameRate {

			time.Sleep(time.Millisecond * time.Duration(updateTime))
		}
		copyToOutput(frameRate, outputDir, inputIdx, outputIdx)
		outputIdx++
		inputIdx = (inputIdx)%len(images) + 1
	}
}

func copyToOutput(frameRate int, outputDir string, inputIdx int, outputIdx int) {
	outputFileStr := outputDir + "loop" + strconv.Itoa(outputIdx) + FILE_EXTN
	source, sourceError := os.Open(images[inputIdx].path)
	if sourceError != nil {
		log.Fatal(sourceError)
	}
	defer source.Close()

	target, targetError := os.OpenFile(outputFileStr, os.O_RDWR|os.O_CREATE, 0666)
	if targetError != nil {
		log.Fatal(targetError)
	}
	defer target.Close()
	_, copyError := io.Copy(target, source)
	if copyError != nil {
		log.Fatal(copyError)
	}
	if outputIdx-(frameRate*45) > 0 { //keep 45 seconds of output
		delFileStr := outputDir + "loop" + strconv.Itoa(outputIdx-(frameRate*45)) + FILE_EXTN
		delError := os.Remove(delFileStr)
		if delError != nil {
			log.Fatal(delError)
		}
	}
}
