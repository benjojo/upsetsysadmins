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
	tname := flag.String("rename", "", "the name you want everything to be called")
	spareinit := flag.Bool("spareinit", false, "Enable for systems that do not allow init to be touched")
	flag.Parse()

	if *tname == "" {
		flag.Usage() // RTFM
		os.Exit(1)
	}

	pids := GetPids(*spareinit)

	for _, pid := range pids {
		AggressiveRename(pid, *tname)
	}
}

func AggressiveRename(pid int, tname string) {
	attempts := 0
	for {
		RenamePid(pid, tname)
		if strings.HasPrefix(GetCmdLine(pid), tname) {
			break
		}
		attempts++
		if attempts == 10 {
			break
		}
	}
}

func GetPids(avoidinit bool) []int {

	files, err := ioutil.ReadDir("/proc")

	if err != nil {
		log.Fatalf("Okay something is really fucked if I can't read proc | %s", err)
	}

	pids := make([]int, 0)

	for _, v := range files {
		if v.IsDir() {
			if v.Name() == "1" && avoidinit {
				continue
			}

			i, err := strconv.ParseInt(v.Name(), 10, 64)
			if err == nil {
				pids = append(pids, int(i))
			}
		}
	}

	return pids
}

func RenamePid(tpid int, tname string) {
	log.Printf("Attempting to rename pid %d to %s", tpid, tname)
	StackLocation := FindStack(tpid)

	if StackLocation == "" {
		log.Printf("Unable to find stack. this shouldnt happen.")
		return
	}

	cmdline := GetCmdLine(tpid)
	offset := GetRamOffset(tpid, StackLocation, cmdline)

	data := []byte(fmt.Sprint(tname))
	data = append(data, 0x00)

	eh, _ := os.FindProcess(tpid)

	err := syscall.PtraceAttach(eh.Pid)

	if err != nil {
		log.Printf("Could not attach to the PID. Why? %s", err)
		return
	}

	_, err = syscall.PtracePokeData(eh.Pid, uintptr(offset), data)

	if err != nil {
		log.Printf("now I've fucked up! %s is the error", err)
		return
	}

	// fmt.Printf("No idea what it means, but here is 'c' : %d", c)

	err = syscall.PtraceDetach(eh.Pid)

	if err != nil {
		log.Printf("Unable to detach?? Why? %s", err)
		return
	}

	err = syscall.Kill(eh.Pid, syscall.SIGCONT)

	if err != nil {
		log.Printf("Unable to detach?? Why? %s", err)
		return
	}
}

func GetOffsets(stackrange string) (start uint64, end uint64) {
	bits := strings.Split(stackrange, "-")
	start, err := strconv.ParseUint(bits[0], 16, 64)
	if err != nil {
		log.Fatalf("failed to decode hex info on map, this should not happen")
		return
	}

	end, err = strconv.ParseUint(bits[1], 16, 64)
	if err != nil {
		log.Fatalf("failed to decode hex info on map, this should not happen")
		return
	}

	return start, end
}

func GetRamOffset(pid int, stackrange, cmdline string) int {
	file, err := os.Open(fmt.Sprintf("/proc/%d/mem", pid))

	if err != nil {
		log.Fatalf("Unable to access the memory of that PID, possibly due to permissions? '%s'", err)
	}

	start, end := GetOffsets(stackrange)

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
