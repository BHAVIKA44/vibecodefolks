package main

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

//go:embed web/*
var assets embed.FS

func main() {
	if len(os.Args) > 2 {
		log.Fatal("usage: vibecodefolks [project-directory]")
	}
	dir := "."
	if len(os.Args) == 2 {
		dir = os.Args[1]
	}
	dir, err := filepath.Abs(dir)
	if err != nil {
		log.Fatal(err)
	}
	info, err := os.Stat(dir)
	if err != nil {
		log.Fatal(err)
	}
	if !info.IsDir() {
		log.Fatal("project path must be a directory")
	}

	web, err := fs.Sub(assets, "web")
	if err != nil {
		log.Fatal(err)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatal(err)
	}
	url := "http://" + listener.Addr().String()
	fmt.Printf("Project: %s\nDashboard: %s\nPress Ctrl+C to stop.\n", dir, url)
	if err := openBrowser(url); err != nil {
		log.Printf("Could not open browser: %v; open %s manually", err, url)
	}
	log.Fatal(http.Serve(listener, http.FileServer(http.FS(web))))
}

func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}
