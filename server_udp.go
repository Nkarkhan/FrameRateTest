package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"time"
)

const (
	CONN_HOST   = "localhost"
	CONN_PORT   = "3333"
	CONN_TYPE   = "udp"
	MAX_WR_SZ   = 1460
	FILE_SZ     = 10000 // 1 Mb files
	FILE_SZ_STR = "File Size:"
)

func main() {
	serverPtr := flag.Bool("server", false, "is this the server")
	addressPtr := flag.String("address", "localhost", "Bind/target address for server/client")
	portPtr := flag.Int("port", 3333, "Bind/target port for server/client")
	fileSzPtr := flag.Int("fSize", 100000, "File Size")
	frameRatePtr := flag.Int("frameRate", 30, "Frame Rate in Hz")

	flag.Parse()
	fmt.Println("ServerMode:", *serverPtr)
	fmt.Println("address:", *addressPtr)
	fmt.Println("port:", *portPtr)
	fmt.Println("fileSz:", *fileSzPtr)
	fmt.Println("Hz:", *frameRatePtr)
	if *serverPtr {
		setupServer(*addressPtr, *portPtr, *fileSzPtr)
	} else {
		setupClient(*addressPtr, *portPtr, *fileSzPtr, *frameRatePtr)
	}
}

// Setup Server
func setupServer(serverAddress string,
	serverPort int,
	fileSize int) {
	// Listen for incoming connections.
	l, err := net.ListenPacket(CONN_TYPE, serverAddress+":"+strconv.Itoa(serverPort))
	if err != nil {
		fmt.Println("Error listening:", err.Error())
		os.Exit(1)
	}
	// Close the listener when the application closes.
	defer l.Close()
	fmt.Println("Listening on " + serverAddress + ":" + strconv.Itoa(serverPort))
	// Handle connections in a new goroutine.
	handleRequest(l, fileSize)
}

func setupClient(serverAddress string,
	serverPort int,
	fileSize int,
	frameRate int) {
	con, err := net.Dial(CONN_TYPE, serverAddress+":"+strconv.Itoa(serverPort))
	if err != nil {
		log.Fatalln(err)
	}
	defer con.Close()
	buffers := make([][]byte, fileSize/MAX_WR_SZ)
	for i := 0; i < fileSize/MAX_WR_SZ; i++ {
		buffers[i] = make([]byte, MAX_WR_SZ)
	}
	hz := 0
	// 30 hz is what we target
	// Time for each frame is 1000/30 msec
	timeForEachFrame := int(1000 / frameRate)
	for {
		t := time.Now()
		for _, buf := range buffers {
			_, err := con.Write(buf)
			if err != nil {
				fmt.Println("Error writing:", err.Error())
				panic(err)
			}
		}
		//		if wrLen != fileSize {
		//			fmt.Println("Wrote :", wrLen)
		//		}
		if time.Since(t) > time.Duration(timeForEachFrame*int(time.Millisecond)) {
			fmt.Println("Falling behind")
		} else {
			for time.Since(t) < time.Duration(timeForEachFrame*int(time.Millisecond)) {
				time.Sleep(time.Millisecond)
			}
		}
		hz = hz + 1
		if hz == (frameRate * 1000) {
			fmt.Println("Sent frames: ", frameRate*1000)
			hz = 0
		}
	}
}

// Handles incoming requests.
func handleRequest(pConn net.PacketConn, fSize int) {
	defer pConn.Close()
	// Make a buffer to hold incoming data.
	buf := make([]byte, fSize+1000)
	// Print chunks of filebuffer received every second
	t := time.Now()
	hz := 0
	totalRead := 0
	for {
		rdLen, _, err := pConn.ReadFrom(buf)
		if err != nil {
			panic(err)
		}
		//		if rdLen != fSize {
		//			fmt.Println("Read bytes:", rdLen)
		//		}
		if rdLen != 0 {
			totalRead = totalRead + rdLen
			framesHere := totalRead / fSize
			totalRead = totalRead % fSize
			hz = hz + framesHere
		}
		if time.Since(t) >= time.Duration(time.Second) {
			fmt.Println("Frame Rate: ", hz)
			t = time.Now()
			hz = 0
		}
	}
	// Close the connection when you're done with it.

}
