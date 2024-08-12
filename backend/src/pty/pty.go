package pty

import (
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/creack/pty"
)

type TerminalSession struct {
	Terminal *os.File
	ReplId   string
	Command  *exec.Cmd
}

type TerminalManager struct {
	sessions map[string]*TerminalSession
	mu       sync.Mutex
}

func NewTerminalManager() *TerminalManager {
	return &TerminalManager{
		sessions: make(map[string]*TerminalSession),
	}
}

func (tm *TerminalManager) CreatePty(id string, replId string, onData func(data []byte, id string)) (*TerminalSession, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Create the shell command
	cmd := exec.Command("bash")

	// Set the working directory
	cmd.Dir = filepath.Join(os.TempDir(), replId)

	// Start the command with a pseudo-terminal
	ptmx, err := pty.Start(cmd)
	if err != nil {
		return nil, err
	}

	// Set up the data handler
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := ptmx.Read(buf)
			if err != nil {
				break
			}
			onData(buf[:n], id)
		}
	}()

	// Store the session
	session := &TerminalSession{
		Terminal: ptmx,
		ReplId:   replId,
		Command:  cmd,
	}
	tm.sessions[id] = session

	// Handle session cleanup on exit
	go func() {
		cmd.Wait()
		tm.mu.Lock()
		defer tm.mu.Unlock()
		delete(tm.sessions, id)
	}()

	return session, nil
}

func (tm *TerminalManager) Write(terminalId string, data string) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	session, exists := tm.sessions[terminalId]
	if !exists {
		return os.ErrNotExist
	}

	_, err := session.Terminal.Write([]byte(data))
	return err
}

func (tm *TerminalManager) Clear(terminalId string) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	session, exists := tm.sessions[terminalId]
	if exists {
		session.Terminal.Close()
		session.Command.Process.Kill()
		delete(tm.sessions, terminalId)
	}
}
