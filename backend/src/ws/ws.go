package ws

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/HarshitNagpal29/replit-clone/backend/src/aws"
	"github.com/HarshitNagpal29/replit-clone/backend/src/fs"
	"github.com/HarshitNagpal29/replit-clone/backend/src/http"

	"github.com/gin-gonic/gin"
	socketio "github.com/googollee/go-socket.io"
)

// Initialize the WebSocket server
func InitWs(httpServer *http.Server) {
	server, err := socketio.NewServer(nil)
	if err != nil {
		log.Fatal(err)
	}

	terminalManager := NewTerminalManager()

	server.OnConnect("/", func(s socketio.Conn) error {
		replId := s.URL().Query().Get("roomId")
		if replId == "" {
			terminalManager.Clear(s.ID())
			s.Close()
			return fmt.Errorf("missing replId")
		}

		go func() {
			localPath := filepath.Join(os.TempDir(), replId)
			if err := aws.FetchS3Folder("code/"+replId, localPath); err != nil {
				log.Println("Failed to fetch S3 folder:", err)
				s.Close()
				return
			}

			rootContent, err := fs.FetchDir(localPath, "")
			if err != nil {
				log.Println("Failed to fetch directory contents:", err)
				s.Close()
				return
			}

			s.Emit("loaded", gin.H{
				"rootContent": rootContent,
			})

			initHandlers(s, replId, terminalManager)
		}()

		return nil
	})

	server.OnError("/", func(s socketio.Conn, e error) {
		log.Println("Socket error:", e)
	})

	server.OnDisconnect("/", func(s socketio.Conn, reason string) {
		log.Println("User disconnected:", reason)
		terminalManager.Clear(s.ID())
	})

	go server.Serve()
	defer server.Close()

	http.Handle("/socket.io/", server)
	http.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("./assets"))))

	log.Println("Serving at localhost:8000...")
	log.Fatal(httpServer.ListenAndServe())
}

// Handler functions for the various socket events
func initHandlers(s socketio.Conn, replId string, terminalManager *TerminalManager) {
	s.OnEvent("/", "fetchDir", func(dir string) ([]File, error) {
		dirPath := filepath.Join(os.TempDir(), replId, dir)
		return FetchDir(dirPath, dir)
	})

	s.OnEvent("/", "fetchContent", func(req struct{ Path string }) (string, error) {
		fullPath := filepath.Join(os.TempDir(), replId, req.Path)
		return fs.FetchFileContent(fullPath)
	})

	s.OnEvent("/", "updateContent", func(req struct {
		Path    string
		Content string
	}) error {
		fullPath := filepath.Join(os.TempDir(), replId, req.Path)
		if err := fs.SaveFile(fullPath, req.Content); err != nil {
			return err
		}
		return aws.SaveToS3("code/"+replId, req.Path, req.Content)
	})

	s.OnEvent("/", "requestTerminal", func() {
		terminalManager.CreatePty(s.ID(), replId, func(data []byte) {
			s.Emit("terminal", gin.H{
				"data": data,
			})
		})
	})

	s.OnEvent("/", "terminalData", func(req struct {
		Data       string
		TerminalId int
	}) {
		terminalManager.Write(s.ID(), req.Data)
	})
}
