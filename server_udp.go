package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	CONN_HOST   = "127.0.0.1"
	CONN_PORT   = 5555
	CONN_TYPE   = "udp"
	MAX_WR_SZ   = 1460
	FILE_SZ     = 500 * 1024 // 1 Mb files
	FILE_SZ_STR = "File Size:"
)

func main() {
	serverPtr := flag.Bool("server", false, "is this the server")
	addressPtr := flag.String("address", CONN_HOST, "Bind/target address for server/client")
	portPtr := flag.Int("port", CONN_PORT, "Bind/target port for server/client")
	fileSzPtr := flag.Int("fSize", 500*1024, "File Size")
	frameRatePtr := flag.Int("frameRate", 30, "Frame Rate in Hz")

	flag.Parse()
	fmt.Println("ServerMode:", *serverPtr)
	fmt.Println("address:", *addressPtr)
	fmt.Println("port:", *portPtr)
	fmt.Println("fileSz:", *fileSzPtr)
	fmt.Println("Hz:", *frameRatePtr)
	if *serverPtr {
		setupServer(*portPtr, *fileSzPtr)
	} else {
		setupClient(*addressPtr, *portPtr, *fileSzPtr, *frameRatePtr)
	}
}

// GetLocalIP returns the non loopback local IP of the host
func GetLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, address := range addrs {
		// check the address type and if it is not a loopback the display it
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				if strings.Contains(ipnet.IP.String(), "192") {
					return ipnet.IP.String()
				}
			}
		}
	}
	return ""
}

// Setup Server
func setupServer(serverPort int,
	fileSize int) {
	// Listen for incoming connections.
	addr, _ := net.ResolveUDPAddr("udp", GetLocalIP()+":"+strconv.Itoa(serverPort))
	l, err := net.ListenUDP(CONN_TYPE, addr)
	if err != nil {
		fmt.Println("Error listening:", err.Error())
		os.Exit(1)
	}
	// Close the listener when the application closes.
	defer l.Close()
	fmt.Println("Listening on " + GetLocalIP() + ":" + strconv.Itoa(serverPort))
	// Handle connections in a new goroutine.
	handleRequest(l, fileSize)
}

func setupClient(serverAddress string,
	serverPort int,
	fileSize int,
	frameRate int) {
	addr, err := net.ResolveUDPAddr("udp", serverAddress+":"+strconv.Itoa(serverPort))
	//	listenaddr, err := net.ResolveUDPAddr("udp", GetLocalIP()+":"+strconv.Itoa(serverPort+1))
	con, err := net.DialUDP(CONN_TYPE, nil, addr)
	fmt.Println("Sending from " + GetLocalIP() + ":" + strconv.Itoa(serverPort))
	if err != nil {
		log.Fatalln(err)
	}
	err = con.SetWriteBuffer(64 * 1024 * 1024)
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
	counter := byte(0)
	for {
		t := time.Now()
		for _, buf := range buffers {
			counter = counter + 1
			buf[0] = counter
			_, err := con.Write(buf)
			if err != nil {
				fmt.Println("Error writing:", err.Error())
				panic(err)
			}
		}
		if time.Since(t) > time.Duration(timeForEachFrame*int(time.Millisecond)) {
			x := time.Since(t)
			fmt.Println("Falling behind", x)
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
	save_counter := byte(0)
	for {
		rdLen, _, err := pConn.ReadFrom(buf)
		if err != nil {
			panic(err)
		}
		if rdLen != 0 {
			totalRead = totalRead + rdLen
			framesHere := totalRead / fSize
			totalRead = totalRead % fSize
			hz = hz + framesHere
			if buf[0] != save_counter+1 {
				fmt.Println("Missed Packet expected and received", save_counter, buf[0])
			}
			save_counter = buf[0]
		}
		if time.Since(t) >= time.Duration(time.Second) {
			fmt.Println("Frame Rate: ", hz)
			t = time.Now()
			hz = 0
		}
	}
	// Close the connection when you're done with it.

}
