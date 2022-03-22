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
	var defInputPtr, ffmpegCmd string
	if runtime.GOOS == "windows" {
		defInputPtr = "C:\\Users\\nkark\\OneDrive\\Documents\\GitHub\\FrameRateTest\\images"
		ffmpegCmd = "C:\\Users\\nkark\\Downloads\\ffmpeg-2022-02-24-git-8ef03c2ff1-full_build\\bin\\ffmpeg"
	} else {
		defInputPtr = "/Users/nkarkhan/Documents/GitHub/FrameRateTest/images"
		ffmpegCmd = "ffmpeg"
	}
	inputPtr := flag.String("input", defInputPtr, "Input Source for images")
	d := flag.Bool("debug", false, "Debug tracing")
	pipe := flag.Bool("pipe", false, "pipe to ffmpeg directly")
	flag.Parse()
	debugSet = *d
	frameRatePtr := flag.Int("frameRate", 30, "Frame Rate in Hz")
	images = make(map[int]imageSequence)
	iterate(*inputPtr, false)
	log.Println("Number of files ", len(images))
	for index := 1; index <= len(images); index++ {
		if debugSet {
			log.Println("index %d - Image %s", index, images[index].fileName)
		}
	}
	wg.Add(1)
	if *pipe {
		fmt.Println("Running the camera stream")

		ffmpegCmd := exec.Command(ffmpegCmd,
			"-loglevel", "error", "-re", "-hwaccel", "cuda",
			"-max_delay", "0", "-max_probe_packets", "1",
			"-f", "image2pipe", "-framerate", "30", "-i", "-",
			"-codec:v", "rawvideo", "-f", "mpegts", "-muxdelay", "0", "udp://127.0.0.1:5555")
		ffpmegStdIn, err := ffmpegCmd.StdinPipe()
		if err != nil {
			log.Fatal(err)
			return
		}
		defer ffpmegStdIn.Close()
		ffpmegStdOut, err := ffmpegCmd.StderrPipe()
		if err != nil {
			return
		}
		err = ffmpegCmd.Start()
		if err != nil {
			log.Fatal(err)
		}
		stdout := bufio.NewScanner(ffpmegStdOut)
		go func(dumpFfmpeg *bufio.Scanner) {
			fmt.Println("ffmpeg output")
			for dumpFfmpeg.Scan() {
				fmt.Println("-")
				fmt.Println(dumpFfmpeg.Text())
			}
		}(stdout)

		go func() {
			defer wg.Done()
			//			stdinWriter := bufio.NewReader(ffpmegStdIn)
			populateffmpegPipe(*frameRatePtr, ffpmegStdIn)
		}()

		ffmpegCmd.Wait()
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

func populateffmpegPipe(frameRate int, filePtr io.WriteCloser) {
	updateTime := 1000 / (frameRate*2 + 3) // in msec, at every updateTime we add a file to ffmpeg stdin
	inputIdx := 1
	totalBytesWritten := 0
	printTime := true
	for {
		time.Sleep(time.Millisecond * time.Duration(updateTime))
		source := images[inputIdx].data
		writeCnt, err := filePtr.Write(source)
		totalBytesWritten = totalBytesWritten + writeCnt
		if err != nil {
			fmt.Printf("Bytes Written %d on file %d", totalBytesWritten, inputIdx)
			log.Fatal(err)
		}
		if printTime {
			fmt.Println("Time to queue images ", time.Now().String())
			printTime = false
		}
		inputIdx = (inputIdx)%len(images) + 1
	}
}
