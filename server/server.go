package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/zhifengle/rss2cloud/p115"
)

type Server struct {
	Agent *p115.Agent
	Port  int
}

type OfflineTask struct {
	Tasks []string `json:"tasks"`
	Cid   string   `json:"cid"`
}

var mux = http.NewServeMux()
var srv *http.Server

func New(agent *p115.Agent, port int) *Server {
	return &Server{
		Agent: agent,
		Port:  port,
	}
}

func (s *Server) Start(ctx context.Context) error {
	mux.Handle("/add", http.HandlerFunc(s.handleAddTask))
	srv = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.Port),
		Handler: mux,
	}
	fmt.Printf("server started on port %d\n", s.Port)
	return srv.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		return err
	}

	fmt.Printf("server stopped properly\n")
	return nil
}

func (s *Server) handleAddTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Decode the JSON data from the request body
	var task OfflineTask
	err := json.NewDecoder(r.Body).Decode(&task)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.Agent.AddMagnetTask(task.Tasks, task.Cid)

	// Send a response back
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("message success"))
}

func (s *Server) StartServer() {
	ctx, cancel := context.WithCancel(context.Background())

	c := make(chan os.Signal, 1)
	// Trigger graceful shutdown on SIGINT or SIGTERM.
	// The default signal sent by the `kill` command is SIGTERM,
	// which is taken as the graceful shutdown signal for many systems, eg., Kubernetes, Gunicorn.
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		sig := <-c
		fmt.Printf("%s received.\n", sig.String())
		err := s.Shutdown(ctx)
		if err != nil {
			fmt.Printf("failed to shutdown server, error: %+v\n", err)
		}
		cancel()
	}()

	if err := s.Start(ctx); err != nil {
		if err != http.ErrServerClosed {
			fmt.Printf("failed to start server, error: %+v\n", err)
			cancel()
		}
	}

	// Wait for CTRL-C.
	<-ctx.Done()
}
