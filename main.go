package main

import (
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/BHAVIKA44/vibecodefolks/internal/analysis"
	"github.com/BHAVIKA44/vibecodefolks/internal/server"
)

//go:embed web/*
var assets embed.FS

// version is set for releases with -ldflags "-X main.version=vX.Y.Z".
var version = "dev"

func main() {
	noOpen := flag.Bool("no-open", false, "do not open a browser")
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), "Usage: vibecodefolks [--no-open] [project-directory]")
		flag.PrintDefaults()
	}
	flag.Parse()
	if *showVersion {
		fmt.Println("vibecodefolks", version)
		return
	}
	if flag.NArg() > 1 {
		flag.Usage()
		os.Exit(2)
	}
	dir := "."
	if flag.NArg() == 1 {
		dir = flag.Arg(0)
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

	if _, err := analysis.Discover(dir); err != nil {
		log.Fatal(err)
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
	if !*noOpen {
		if err := openBrowser(url); err != nil {
			log.Printf("Could not open browser: %v; open %s manually", err, url)
		}
	}
	log.Fatal(http.Serve(listener, server.Handler(dir, web, listener.Addr().String())))
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
