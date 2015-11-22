package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
	"syscall"
)

func main() {
	tpid := flag.Int("pid", 0, "the pid you want to fuck with")
	tname := flag.String("rename", "", "the name you want it to be called")
	flag.Parse()

	if *tpid == 0 || *tname == "" {
		flag.Usage() // RTFM
		os.Exit(1)
	}

	StackLocation := FindStack(*tpid)

	if StackLocation == "" {
		log.Fatal("Unable to find stack. this shouldnt happen.")
	}

	log.Printf("Found stack at %s", StackLocation)

	cmdline := GetCmdLine(*tpid)
	offset := GetRamOffset(*tpid, StackLocation, cmdline)

	data := []byte(fmt.Sprint(*tname))
	data = append(data, 0x00)

	eh, _ := os.FindProcess(*tpid)

	err := syscall.PtraceAttach(eh.Pid)

	if err != nil {
		log.Fatalf("Could not attach to the PID. Why? %s", err)
	}

	_, err = syscall.PtracePokeData(eh.Pid, uintptr(offset), data)

	if err != nil {
		log.Fatalf("now I've fucked up! %s is the error", err)
	}

	// fmt.Printf("No idea what it means, but here is 'c' : %d", c)

	err = syscall.PtraceDetach(eh.Pid)

	if err != nil {
		log.Fatalf("Unable to detach?? Why? %s", err)
	}

	err = syscall.Kill(eh.Pid, syscall.SIGCONT)

	if err != nil {
		log.Fatalf("Unable to detach?? Why? %s", err)
	}

}

func GetOffsets(stackrange string) (start uint64, end uint64) {
	bits := strings.Split(stackrange, "-")
	start, err := strconv.ParseUint(bits[0], 16, 64)
	if err != nil {
		log.Fatalf("failed to decode hex info on map, this should not happen")
	}

	end, err = strconv.ParseUint(bits[1], 16, 64)
	if err != nil {
		log.Fatalf("failed to decode hex info on map, this should not happen")
	}

	return start, end
}

func GetRamOffset(pid int, stackrange, cmdline string) int {
	file, err := os.Open(fmt.Sprintf("/proc/%d/mem", pid))

	if err != nil {
		log.Fatalf("Unable to access the memory of that PID, possibly due to permissions? '%s'", err)
	}

	start, end := GetOffsets(stackrange)

	log.Printf("Reading from %d to %d bytes", start, end)

	stack := make([]byte, end-start)

	file.Seek(int64(start), 0)

	n, err := file.Read(stack)

	if err != nil {
		log.Printf("uwot %d", n)
	}

	ptr := strings.LastIndex(string(stack), cmdline)

	file.Close()

	return ptr + int(start)
}

func GetCmdLine(pid int) string {
	cmdline, err := ioutil.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))

	if err != nil {
		log.Fatalf("Unable to access the memory map of that PID, possibly due to permissions? '%s'", err)
	}

	return string(cmdline)
}

func FindStack(pid int) string {
	mapdata, err := ioutil.ReadFile(fmt.Sprintf("/proc/%d/maps", pid))

	if err != nil {
		log.Fatalf("Unable to access the memory map of that PID, possibly due to permissions? '%s'", err)
	}

	lines := strings.Split(string(mapdata), "\n")

	for _, v := range lines {
		if strings.Contains(v, "[stack]") {
			bits := strings.Split(v, " ")
			return bits[0]
		}
	}

	return ""
}
