package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"os"

	"gopkg.in/yaml.v3"
)

// 配置结构体
type Config struct {
	ListenPort   int               `yaml:"listen_port"`
	LoginServer  string            `yaml:"login_server"`
	LogonPort    int               `yaml:"logon_port"`
	ProxyServer  string            `yaml:"proxy_server"`
	ReplaceHosts map[string]string `yaml:"replace_hosts"`
}

// 全局变量存储配置
var config Config

// 读取 YAML 配置
func loadConfig(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(data, &config)
}

// 服务器启动逻辑
func main() {
	// 读取配置文件
	err := loadConfig("config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 启动 TCP 监听
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", config.ProxyServer, config.ListenPort))
	if err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
	defer listener.Close()
	log.Printf("Listening on port %d\n", config.ListenPort)

	// 处理客户端连接
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Error accepting connection: %v", err)
			continue
		}
		log.Printf("New connection from %s\n", conn.RemoteAddr())

		// 启动 Goroutine 处理客户端请求
		go handleClient(conn)
	}
}

// 处理客户端连接
func handleClient(client net.Conn) {
	defer client.Close()

	// 连接登录服务器
	server, err := net.Dial("tcp", fmt.Sprintf("%s:%d", config.LoginServer, config.LogonPort))
	if err != nil {
		log.Printf("Error connecting to login server: %v", err)
		return
	}
	defer server.Close()

	// 使用 Goroutine 进行双向数据转发
	go copyData(client, server, false) // 客户端-> 服务器 （直接转发）
	copyData(server, client, true)     // 服务器-> 客户端 （可能需要替换服务器地址）
}

// 数据转发逻辑（使用 io.Copy）
func copyData(src, dst net.Conn, replace bool) {
	if replace {
		// 需要对 `src -> dst` 方向的流量进行拦截处理
		buffer := make([]byte, 4096)
		for {
			n, err := src.Read(buffer)
			if err != nil {
				if err != io.EOF {
					log.Printf("Read error: %v", err)
				}
				return
			}

			// 如果数据包是 Realm 列表，进行替换
			if n > 0 && buffer[0] == 16 {
				log.Println("Intercepting realm list, replacing hosts")
				newData := replaceRealmHost(buffer[:n])
				fmt.Println("replace")
				dst.Write(newData)
			} else {
				dst.Write(buffer[:n])
			}
		}
	} else {
		// 直接转发数据，不做修改
		_, err := io.Copy(dst, src)
		if err != nil {
			log.Printf("io.Copy error: %v", err)
		}
	}
}

// 替换服务器地址
func replaceRealmHost(data []byte) []byte {
	fmt.Println("start read buffer,", len(data))
	if len(data) < 5 {
		fmt.Println("data count < 5,", len(data))
		return data
	}
	fmt.Println("total", len(data), "row")
	realmCount := data[7]   //1.72版本中 index7 服务器数量
	var output bytes.Buffer //创建临时缓冲区，用于存储拼接的字节流数据
	fmt.Println("set data to buffer,total server:", realmCount)
	output.Write([]byte{16, 0, 0, 0, 0, 0, 0, realmCount}) //写入数据流前8位，第二位为长度，暂时写0，最后更改

	index := 8
	for i := 0; i < int(realmCount); i++ {
		fmt.Println("start get data index", i)
		if index+5 > len(data) {
			fmt.Println("eg 10,return", string(data))
			return data
		}
		fmt.Println("concat data:", string(data[index:index+5]))
		output.Write(data[index : index+5])
		index += 5

		// 解析服务器名称
		nameEnd := bytes.IndexByte(data[index:], 0) + index
		fmt.Println("server name:", string(data[index:nameEnd]))
		if nameEnd < index {
			return data
		}
		output.Write(data[index : nameEnd+1])
		index = nameEnd + 1

		// 解析服务器地址
		fmt.Println("start get server host")
		hostEnd := bytes.IndexByte(data[index:], 0) + index
		fmt.Println("cursor index", index, "host endpoint index", hostEnd)
		if hostEnd < index {
			return data
		}
		oldHost := string(data[index:hostEnd])
		newHost, exists := config.ReplaceHosts[oldHost]
		fmt.Println(oldHost, "-->", newHost)
		if !exists {
			newHost = oldHost
		}

		log.Printf("Replacing host: %s -> %s\n", oldHost, newHost)
		output.Write([]byte(newHost))
		output.WriteByte(0)
		index = hostEnd + 1

		if index+7 > len(data) {
			return data
		}
		output.Write(data[index : index+7])
		index += 7
	}

	// 写入剩余数据
	output.Write(data[index:])
	length := output.Len() - 3
	binary.LittleEndian.PutUint16(output.Bytes()[1:3], uint16(length))
	return output.Bytes()
}
