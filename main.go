package main

import (
	"encoding/binary"
	"fmt"
	"gobot-nativehost/log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
)

var wsConn *websocket.Conn

func pumpStdin(conn *websocket.Conn) {
	for {
		if conn != nil {
			_, message, err := conn.ReadMessage()
			log.Logger.Debug().Msg("read from control:" + string(message))
			if err != nil {
				err := conn.Close()
				if err != nil {
					log.Logger.Error().AnErr("websocket关闭失败:", err)
				}
				break
			}
			length := len(message)
			buf := make([]byte, 4)
			binary.LittleEndian.PutUint16(buf, uint16(length))
			if _, err := os.Stdout.Write(buf); err != nil {
				log.Logger.Error().AnErr("向浏览器写入消息长度失败:", err)
				continue
			}
			if _, err := os.Stdout.Write(message); err != nil {
				log.Logger.Error().AnErr("向浏览器写入消息失败:", err)
				continue
			}
		} else {
			break
		}
	}
}

func pumpStdout() {
	count := 0
	for {
		buf := make([]byte, 4)
		length, err := os.Stdin.Read(buf)
		if err == nil && length == 4 {
			dataLength := binary.LittleEndian.Uint32(buf)
			data := make([]byte, dataLength)
			if _, err = os.Stdin.Read(data); err == nil {
				log.Logger.Info().Msg("receive from browser:" + string(data))
				if wsConn != nil {
					if err = wsConn.WriteMessage(1, data); err != nil {
						log.Logger.Error().AnErr("向websocket写入消息失败:", err)
						err := wsConn.Close()
						if err != nil {
						}
						wsConn = nil
					}
				}
			} else {
				log.Logger.Error().AnErr("从浏览器读取消息失败", err)
				continue
			}
			count = 0
		} else if length == 0 {
			time.Sleep(time.Millisecond * 200)
			count += 1
			if count > 10 {
				log.Logger.Info().Msg("浏览器关闭")
				os.Exit(0)
			}
		}
	}
}

var upgrader = websocket.Upgrader{CheckOrigin: checkOrigin}

func checkOrigin(r *http.Request) bool {
	return true
}
func serveHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	_, err := http.ResponseWriter.Write(w, []byte("success"))
	if err != nil {
		return
	}
}

func serveWs(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Logger.Info().Any("upgrade:", err)
		return
	}
	// go pumpStdout(conn)
	go pumpStdin(conn)
	if wsConn != nil {
		err := wsConn.Close()
		if err != nil {
		}
	}
	wsConn = conn
}

func PortCheck(port int) bool {
	l, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%s", strconv.Itoa(port)))

	if err != nil {
		return false
	}
	defer func(l net.Listener) {
		_ = l.Close()
	}(l)
	return true
}

func main() {
	log.Init()
	log.Logger.Info().Msg("init")
	args := os.Args[:]
	for _, arg := range args {
		log.Logger.Info().Msg("启动参数:" + arg)
	}
	go pumpStdout()
	port := 3000
	for ; port < 4000; port++ {
		if PortCheck(port) {
			break
		}
	}
	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	exePath := filepath.Dir(ex)
	err = os.WriteFile(exePath+string(os.PathSeparator)+"nativehost_port", []byte(strconv.Itoa(port)), os.ModePerm)
	if err != nil {
		log.Logger.Error().Msg("写入端口号失败")
		return
	}
	http.HandleFunc("/", serveHome)
	http.HandleFunc("/ws", serveWs)
	server := &http.Server{
		Addr:              fmt.Sprintf("127.0.0.1:%s", strconv.Itoa(port)),
		ReadHeaderTimeout: 3 * time.Second,
	}
	err = server.ListenAndServe()
	if err != nil {
		log.Logger.Error().Msg("服务启动失败")
		return
	}
}
