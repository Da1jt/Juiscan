package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

func main() {
	shutdown := false
	log := false
	deepscan := false
	flag.BoolVar(&shutdown, "shutdown", shutdown, "关闭")
	flag.BoolVar(&log, "l", log, "日志写入")
	flag.BoolVar(&deepscan, "d", deepscan, "递归扫描")
	url := flag.String("url", "", "目标 URL")
	ports := flag.String("port", "", "端口扫描范围")
	slowscan := flag.Bool("s", false, "慢速扫描")
	help := flag.Bool("h", false, "帮助")
	flag.Parse()

	fmt.Println("\n\n                     /$$                                                            \n                    |__/                                                            \n       /$$ /$$   /$$ /$$  /$$$$$$$  /$$$$$$$  /$$$$$$  /$$$$$$$   /$$$$$$   /$$$$$$ \n      |__/| $$  | $$| $$ /$$_____/ /$$_____/ |____  $$| $$__  $$ /$$__  $$ /$$__  $$\n       /$$| $$  | $$| $$|  $$$$$$ | $$        /$$$$$$$| $$  \\ $$| $$$$$$$$| $$  \\__/\n      | $$| $$  | $$| $$ \\____  $$| $$       /$$__  $$| $$  | $$| $$_____/| $$      \n      | $$|  $$$$$$/| $$ /$$$$$$$/|  $$$$$$$|  $$$$$$$| $$  | $$|  $$$$$$$| $$      \n      | $$ \\______/ |__/|_______/  \\_______/ \\_______/|__/  |__/ \\_______/|__/      \n /$$  | $$                                                                          \n|  $$$$$$/                                                                          \n \\______/                                                                           ")

	if *help {
		helper()
		return
	}
	if *url == "" {
		fmt.Println("[-] 无效的 URL")
		return
	}

	if *slowscan {
		fmt.Println("[+] 慢速扫描已启用")
	}

	portScan := "no"
	GpStart, GpEnd, GPortSingle := 0, 0, 0
	if *ports != "" {
		if strings.Contains(*ports, "-") {
			parts := strings.Split(*ports, "-")
			pstart := parts[0]
			pend := parts[1]
			pstartInt, err := strconv.Atoi(pstart)
			if err != nil {
				fmt.Println("[-] 端口范围输入错误", err)
				return
			}
			pendInt, err := strconv.Atoi(pend)
			if err != nil {
				fmt.Println("[-] 端口范围输入错误", err)
				return
			}
			if isValidPort(pstartInt) && isValidPort(pendInt) {
				fmt.Printf("[+] 端口扫描已启用 %s-%s", pstart, pend)
				portScan = "range"
				GpStart, GpEnd = pstartInt, pendInt

			} else {
				fmt.Println("[-] 端口范围输入错误", err)
				return
			}
		} else {
			portSingle, err := strconv.Atoi(*ports)
			if err != nil {
				fmt.Println("端口范围输入错误", err)
				return
			}
			if isValidPort(portSingle) {
				fmt.Printf("[+] 端口扫描已启用 %d", portSingle)
				portScan = "single"
			}
		}
	}

	if shutdown {
		println("[+] 扫描完成后关机")
	}

	// 处理其他命令行参数
	for i := 1; i < len(os.Args); i++ {
		if os.Args[i] == "-h" {
			helper()
			return
		}
		if os.Args[i] == "-shutdown" {
			shutdown = true
		}
		if os.Args[i] == "-l" {
			log = true
		}
		if os.Args[i] == "-d" {
			deepscan = true
		}
	}
	conn, err := net.Dial("ip4:icmp", *url)
	if err != nil {
		fmt.Println("地址存活性检测失败：", err)
		os.Exit(1)
		return
	}
	defer func(conn net.Conn) {
		err := conn.Close()
		if err != nil {

		}
	}(conn)

	dirPath := "./dictionary/"

	fileList, err := getFileList(dirPath)
	if err != nil {
		fmt.Println("读取目录失败：", err)
		return
	}
	temp, tempall, logfail, logpath := 0, 0, 0, ""
	var wg sync.WaitGroup
	for _, file := range fileList {
		wg.Add(1)
		go func(file string) {
			defer wg.Done()
			temp, tempall, logfail, logpath = processFile(file, *url, *slowscan, log /*, deepscan*/)
		}(file)
	}
	var ipA []string
	var alivePort [][]int
	TotalAlive := 0
	if portScan != "no" {
		if portScan == "range" {
			portlist := []int{GpStart, GpEnd}
			ipA, alivePort, TotalAlive = portScanner(portlist, portScan, *url)
		} else {
			portlist := []int{GPortSingle}
			ipA, alivePort, TotalAlive = portScanner(portlist, portScan, *url)
		}
		for i := 0; i < len(alivePort); i++ {
			for j := 0; j < len(alivePort[i]); j++ {
				Err := logs(strconv.Itoa(alivePort[i][j]), logpath)
				if Err != nil {
					logfail++
				}
			}
		}
	}

	wg.Wait()
	fmt.Println("\n\n[*] 扫描完成")
	fmt.Print("[+] 存在数量路径：")
	fmt.Println(tempall, " / ", temp)
	fmt.Print("[+] 存活端口数量：")
	allPort := (GpEnd - GpStart) * len(ipA)
	fmt.Print(TotalAlive, " / ", allPort, "\n")
	if logfail >= 1 {
		fmt.Printf("[-] 全部或部分日志写入失败，失败数量:%d", logfail)
	} else if logfail == 0 {
		println("\n[+] 日志写入完成")
	}
	fmt.Println()

	if shutdown == true {
		shutdn := exec.Command("shutdown", "/f")
		shutdnt := shutdn.Run()
		if shutdnt != nil {
			fmt.Println("\n[-] 关机失败")
		} else {
			fmt.Println("[+] 关机成功")
		}
	}
}

// 获取各个字典名称
func getFileList(dirPath string) ([]string, error) {
	var fileList []string

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if filepath.Ext(path) == ".txt" {
			fileList = append(fileList, path)
		}
		return nil
	})

	return fileList, err
}

func isValidPort(port int) bool {
	if port >= 1 && port <= 65535 {
		return true
	}
	return false
}
func portScanner(portlist []int, method string, ip string) ([]string, [][]int, int) {
	var alivePort [][]int
	TotalAlive := 0
	ipA, err := url2ip(ip)
	if err != nil {
		fmt.Println("[-] url解析错误：", err)
		return nil, nil, 0
	}
	for j := 0; j < len(ipA); j++ {
		if method == "single" {
			alivePort[j][0] = subPortscan(portlist[0], ipA[j])
			if alivePort[j][0] != 0 {
				TotalAlive++
				fmt.Printf("[+] %s:%d 端口开放\n", ipA[j], alivePort[j][0])
			}
		} else if method == "range" {
			counter := 0
			for i := portlist[0]; i <= portlist[1]; i++ {
				res := subPortscan(i, ipA[j])
				if res != 0 {
					alivePort[j][counter] = res
					fmt.Printf("[+] %s:%d 端口开放\n", ipA[j], alivePort[j][counter])
					counter++
					TotalAlive++
				}
			}
		}
	}
	return ipA, alivePort, TotalAlive
}

func subPortscan(port int, ip string) int {
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(ip, strconv.Itoa(port)), time.Second)
	if err == nil {
		defer func(conn net.Conn) {
			err := conn.Close()
			if err != nil {
				fmt.Println("[-] 关闭连接失败：", err)
			}
		}(conn)
		return port
	}
	return 0
}

// 处理字典
func processFile(filePath string, url string, slowmode bool, log bool /*, deepscan bool*/) (int, int, int, string) {
	fmt.Println("[+] 处理字典：", filePath)

	file, err := os.Open(filePath)
	if err != nil {
		fmt.Println("[-] 打开文件失败：", err)
		return 0, 0, 0, "0"
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			fmt.Println("[-] 关闭文件失败：", err)
		}
	}(file)

	var fullc = 0
	var counterc = 0
	/*deepnumb := 0
	deepnexist := 0*/
	dislogcount := 0
	currentTime := time.Now()
	var finalPath = "./log/" + url + " " + currentTime.Format("2006-01-01_15-04-05") + ".txt"
	if log {
		_, err := os.Create(finalPath)
		if err != nil {
			return 0, 0, 0, ""
		}
	}
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		path := scanner.Text()
		result := checkPathExists(url, slowmode, path, log, finalPath)
		fullc++
		if result == "httpnormalisable" || result == "httpsnormalisable" {
			dislogcount++
			counterc++
		}
		if result == "httpsnormal" || result == "httpnormal" {
			counterc++
			/*if deepscan {
				if result == "normal" {
					counterc++
					temppath := url + "/" + path
					if deepscan {
						for scanner.Scan() {
							path := scanner.Text()
							println(temppath)
							deepresult := deepcheck(url+temppath, slowmode, path, log, finalPath, currentTime.String())
							fullc++
							if deepresult == "normal" {
								counterc++
								deepnumb++
							}
							if deepresult == "normalisable" {
								dislogcount++
								counterc++
								deepnumb++
							}
							if deepresult == "none" {
								deepnexist++
								counterc++
							}
						}
						println(deepnumb)
					}
				}
			}*/
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("[-] 读取文件失败：", err)
	}
	if log == false {
		return fullc, counterc, -1, finalPath
	}
	return fullc, counterc, dislogcount, finalPath
}
func url2ip(url string) ([]string, error) {
	addrs, err := net.LookupIP(url)
	if err != nil {
		return nil, err
	}
	var ips []string
	for _, ip := range addrs {
		ips = append(ips, ip.String())
	}
	return ips, nil
}

// 发包
func checkPathExists(url string, slowmode bool, path string, log bool, finalpath string) string {
	if slowmode {
		time.Sleep(1 * time.Second)
	}
	fullURL := url + "/" + path
	response, err := http.Head("https://" + fullURL)
	if err != nil {
		/*fmt.Println("请求发生错误：", err)*/
		return "error"
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Println("关闭响应体时发生错误：", err)
		}
	}(response.Body)

	if response.StatusCode != http.StatusNotFound {
		temppathhttps := "https://" + fullURL
		fmt.Print("[+] ", time.Now().Format("15:04:05"), "  ", response.StatusCode, "  ", temppathhttps, "\n")
		if log {
			if err := logs(temppathhttps, finalpath); err != nil {
				fmt.Println("Error writing to log:", err)
				return "httpsnormalisable"
			}
		}
		return "httpsnormal"
	}

	responser, err := http.Head("http://" + fullURL + "/")
	if err != nil {
		fmt.Println("Error sending HTTP request:", err)
		return "error"
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Println("关闭响应体时发生错误：", err)
		}
	}(responser.Body)

	if responser.StatusCode != http.StatusNotFound {
		empathetic := "http://" + fullURL
		fmt.Print("[+] ", time.Now().Format("15:04:05"), "  ", response.StatusCode, "  ", empathetic, "\n")
		if log {
			if err := logs(empathetic, finalpath); err != nil {
				fmt.Println("Error writing to log:", err)
				return "httpnormalisable"
			}
		}
		return "httpnormal"
	}
	return "none"
}

func logs(exist string, finalPath string) error {
	file, err := os.OpenFile(finalPath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			fmt.Println("关闭文件时发生错误：", err)
		}
	}(file)
	_, err = file.WriteString(exist + "\n")
	if err != nil {
		return err
	}
	return nil
}

// -h
func helper() {
	fmt.Println("<Useage>: \n\n     -url xxx.xxx.xxx\n     -h help\n     -s 慢速扫描\n     -p 目标端口扫描，可以以1~100的格式指定扫描范围或单个扫描     \n     -l 存储存在的路径入日志./log     \n     -shutdown 扫描任务完成后关机\n\n<Examp1e>:\n\n     juiscan.exe -url 127.0.0.1 -s\n     juiscan.exe -url 127.0.0.1 -p 1-100\n     juiscan.exe -url 127.0.0.1 -shutdown -l")
}
