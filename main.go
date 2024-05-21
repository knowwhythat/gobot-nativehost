package main

import (
	"encoding/binary"
	"gobot-nativehost/log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/websocket"
)

var wsConn *websocket.Conn

func pumpStdin(conn *websocket.Conn) {
	for {
		if conn != nil {
			_, message, err := conn.ReadMessage()
			log.Logger.Info().Msg("read from control:" + string(message))
			if err != nil {
				conn.Close()
				break
			}
			length := len(message)
			buf := make([]byte, 4)
			binary.LittleEndian.PutUint16(buf, uint16(length))
			if _, err := os.Stdout.Write(buf); err != nil {
				conn.Close()
				break
			}
			if _, err := os.Stdout.Write(message); err != nil {
				conn.Close()
				break
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
						wsConn = nil
					}
				}
			} else {
				log.Logger.Error().AnErr("break", err)
				break
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
	http.ResponseWriter.Write(w, []byte("success"))
}

func serveWs(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Logger.Info().Any("upgrade:", err)
		return
	}
	// go pumpStdout(conn)
	go pumpStdin(conn)
	wsConn = conn
}

func main() {
	log.Init()
	log.Logger.Info().Msg("init")
	args := os.Args[:]
	for _, arg := range args {
		log.Logger.Info().Msg(arg)
	}
	go pumpStdout()
	http.HandleFunc("/", serveHome)
	http.HandleFunc("/ws", serveWs)
	server := &http.Server{
		Addr:              "127.0.0.1:8080",
		ReadHeaderTimeout: 3 * time.Second,
	}
	server.ListenAndServe()
}
