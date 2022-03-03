package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
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
	inputPtr := flag.String("input", "c:\\Users\\nkark\\OneDrive\\Documents\\GitHub\\FrameRateTest\\images", "Input Source for images")
	d := flag.Bool("debug", false, "Debug tracing")
	flag.Parse()
	debugSet = *d
	outputPtr := flag.String("output", "C:\\Users\\nkark\\tmp\\loopImages\\", "Output Dir for images")
	frameRatePtr := flag.Int("frameRate", 30, "Frame Rate in Hz")
	images = make(map[int]imageSequence)
	iterate(*inputPtr)
	log.Println("Number of files ", len(images))
	for index := 1; index <= len(images); index++ {
		if debugSet {
			log.Println("index %d - Image %v", index, images[index])
		}
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		populateOutput(*frameRatePtr, *outputPtr)
	}()
	wg.Wait()
}
func iterate(path string) {
	filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			c, _ := os.Getwd()
			fmt.Printf(c)
			log.Fatalf(err.Error() + path)
		}
		var imgDate imageSequence
		imgDate.path = path
		imgDate.fileName = info.Name()
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
			imgDate.seq = seq
			images[seq] = imgDate
		}

		return nil
	})
}

func populateOutput(frameRate int, outputDir string) {
	updateTime := 1000 / frameRate // in msec, at every updateTime we add a file to output and remove 1
	outputIdx := 1
	inputIdx := 1
	for {
		if outputIdx > frameRate {

			time.Sleep(time.Millisecond * time.Duration(updateTime))
		}
		copyToOutput(frameRate, outputDir, inputIdx, outputIdx)
		outputIdx++
		inputIdx = (inputIdx+1)/len(images) + 1
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
	if outputIdx-(frameRate+1) > 0 {
		delFileStr := outputDir + "loop" + strconv.Itoa(outputIdx-(frameRate+1)) + FILE_EXTN
		delError := os.Remove(delFileStr)
		if delError != nil {
			log.Fatal(delError)
		}
	}
}
