package main

import (
	"bytes"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
)

// When running will create listen for UDP, TCP or REST connections
// and will decode the bytes transferred for a function in the beginning byte
// then seperate the function with a colon and pass the key and if applicable the value
// example (Add function A:YOURKEY:YOURVALUE)
// Functions
// A:KEY:VALUE - Add new pair
// U:KEY:VALUE - Update the specified key with the new value
// D:KEY - Delete the specified key
// R:KEY - Return the value pair of any given key
// I - Return a count of items in the cache
// L:FILENAME - Load and overwrite current cache with saved file
// S:FILENAME - Save the current cache to a specific
// P - Return a json of all keys + values in the cache
//
// With rest it is slightly different:
// If you navigate to ip:port and then
// /addval/KEY/VALUE - Add the pair
// /delval/KEY - Delete the specified key
// /updateval/KEY/VALUE - Update the specified key with the new value
// /retval/KEY - Return the value pair of any given key

func main() {
	// Creates a listener on TCP
	l, err := net.Listen("tcp", "127.0.0.1:1111")
	if err != nil {
		fmt.Print(err)
	}
	// Create a listener on UDP
	addr := net.UDPAddr{
		Port: 1111,
		IP:   net.ParseIP("127.0.0.1"),
	}
	ser, err := net.ListenUDP("udp", &addr)
	if err != nil {
		fmt.Printf("Some error %v\n", err)
		return
	}
	// Create a new cache to store the values
	c := newcache()

	// Run the TCP and UDP as go functions
	go acceptTCP(l, c)
	go acceptUDP(ser, c)

	// Run the REST server
	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/addval/{key}/{value}", c.Add)
	router.HandleFunc("/delval/{key}", c.Delete)
	router.HandleFunc("/retval/{key}", c.Retrieve)
	router.HandleFunc("/updateval/{key}/{value}", c.Update)

	fmt.Println(http.ListenAndServe(":1111", router))
}

// TCP Processor
func acceptTCP(l net.Listener, c *cache) {
	defer l.Close()
	for {
		// Accept the connection
		conn, err := l.Accept()
		if err != nil {
			fmt.Println(err)
		}
		// Create a 2k buffer to hold the message recieved
		buffer := make([]byte, 2048)
		n, _ := conn.Read(buffer)
		v := c.processmsg(buffer[:n])
		if v != nil {
			// If a return message the write back to original connection
			conn.Write(v)
		}
	}
}

// UDP Processor
func acceptUDP(l *net.UDPConn, c *cache) {
	// Create a 2k buffer to hold the message recieved
	buffer := make([]byte, 2048)
	for {
		// Handle UDP slightly differently from TCP
		n, addr, err := l.ReadFromUDP(buffer)
		if err != nil {
			fmt.Printf("Some error %v\n", err)
		}
		v := c.processmsg(buffer[:n])
		if v != nil {
			// If a return message the write back to address readfromudp
			l.WriteToUDP(v, addr)
		}
	}
}

func (c *cache) processmsg(m []byte) []byte {
	// Handle the function being called
	f := m[:1]
	var rv []byte
	switch strings.ToUpper(string(f)) {
	case "A":
		// Add to cache splits on colon and trims the bytes
		s := bytes.Split(m[2:], []byte(":"))
		rv = c.addKV(string(s[0]), s[1][:len(string(s[1]))])
	case "U":
		// Update value splits on colon and trims the bytes
		s := bytes.Split(m[2:], []byte(":"))
		rv = c.updateKV(string(s[0]), s[1][:len(string(s[1]))])
	case "D":
		// Deletes value from cache
		rv = c.delKV(string(m[2:]))
	case "R":
		// Recalls value from cache
		v := c.retKV(string(m[2:]))
		if v != nil {
			rv = []byte(fmt.Sprintf("{%q: %q, %q: %q}", "key", string(m[2:]), "value", string(v)))
		} else {
			rv = []byte(fmt.Sprintf("%q does not exist", string(m[2:])))
		}
	case "I":
		v := c.cachesize()
		rv = []byte(fmt.Sprintf("Cache has %v items", v))
	case "L":
		tf := false
		c, tf = c.LoadFile(string(m[2:]))
		if tf == false {
			rv = []byte("File failed to load please check location")
		} else {
			rv = []byte("File Loaded")
		}
	case "S":
		err := c.SaveFile(string(m[2:]))
		if err != nil {
			rv = []byte("File was not able to save")
		}
		rv = []byte("File Saved")
	case "P":
		rv = c.Printdb()
	}
	return rv
}
