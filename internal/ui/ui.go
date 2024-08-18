package ui

import (
	"encoding/json"
	"fmt"
	"github.com/rxxuzi/lumi/internal/core"
	"html/template"
	"log"
	"net"
	"net/http"
	"sync"
)

type WebUI struct {
	Port     int
	Status   string
	Progress *core.Progress
	mu       sync.Mutex
}

func NewWebUI(port int) *WebUI {
	return &WebUI{
		Port:   port,
		Status: "Idle",
	}
}

func (w *WebUI) Start() error {
	http.HandleFunc("/", w.handleIndex)
	http.HandleFunc("/launch", w.handleLaunch)
	http.HandleFunc("/status", w.handleStatus)

	http.Handle("/static/", http.StripPrefix("/static/", GetStaticFilesHandler()))

	localIP := GetLocalIP()
	log.Printf("Server is running on:\n")
	log.Printf("http://localhost:%d\n", w.Port)
	log.Printf("http://%s:%d\n", localIP, w.Port)

	return http.ListenAndServe(fmt.Sprintf(":%d", w.Port), nil)
}

func (w *WebUI) handleIndex(rw http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFS(staticFiles, "static/index.html")
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	tmpl.Execute(rw, nil)
}

func (w *WebUI) handleLaunch(rw http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(rw, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var config core.Lumi
	err := json.NewDecoder(r.Body).Decode(&config)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}

	// Validate and adjust MediaCount
	if config.MediaCount < core.MIN_MEDIA_COUNT {
		config.MediaCount = core.MIN_MEDIA_COUNT
	} else if config.MediaCount > core.MAX_MEDIA_COUNT {
		config.MediaCount = core.MAX_MEDIA_COUNT
	}

	w.setStatus("Running")
	go func() {
		w.Progress = core.Launch(&config)
		w.setStatus("Completed")
	}()

	rw.WriteHeader(http.StatusOK)
	fmt.Fprint(rw, "Lumi process started")
}

func (w *WebUI) handleStatus(rw http.ResponseWriter, r *http.Request) {
	w.mu.Lock()
	status := w.Status
	progress := w.Progress
	w.mu.Unlock()

	response := map[string]interface{}{
		"status": status,
	}

	if progress != nil {
		response["progress"] = map[string]interface{}{
			"totalMedia":        progress.TotalMedia,
			"requestedMedia":    progress.RequestedMedia,
			"downloadedImages":  progress.DownloadedImages,
			"skippedImages":     progress.SkippedImages,
			"currentFileNumber": progress.CurrentFileNumber,
			"terminated":        progress.Terminated,
		}
	}

	json.NewEncoder(rw).Encode(response)
}

func (w *WebUI) setStatus(status string) {
	w.mu.Lock()
	w.Status = status
	w.mu.Unlock()
}

// GetLocalIP returns the non-loopback local IPv4 address
func GetLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "unable to get local IP"
	}
	for _, address := range addrs {
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil && !ipnet.IP.IsLinkLocalUnicast() {
				return ipnet.IP.String()
			}
		}
	}
	return "No suitable IP found"
}
