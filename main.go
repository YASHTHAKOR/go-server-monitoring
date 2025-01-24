package main

import (
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
)

var (
	gitInfo = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "git_commit_info",
			Help: "Git commit information",
		},
		[]string{"repository", "hash", "author", "timestamp", "branch", "message"},
	)

	cpuUsage = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "system_cpu_usage",
			Help: "Current CPU usage percentage",
		},
	)

	memoryUsage = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "system_memory_usage",
			Help: "Current memory usage percentage",
		},
	)
)

func registerGitHubMetrics() {
	prometheus.MustRegister(gitInfo)
	go func() {
		for {
			repoPath := os.Getenv("REPO_PATH")
			log.Println(repoPath)
			// Get hash, author, timestamp, message in one command
			logCmd := exec.Command("git", "-C", repoPath, "log", "-1", "--format=%H|%an|%aI|%s")
			info, _ := logCmd.Output()
			parts := strings.Split(strings.TrimSpace(string(info)), "|")
			commitHash := parts[0]
			author := parts[1]
			timestamp := parts[2]
			message := parts[3]

			// Get branch
			branchCmd := exec.Command("git", "-C", repoPath, "rev-parse", "--abbrev-ref", "HEAD")
			branch, _ := branchCmd.Output()
			branchName := strings.TrimSpace(string(branch))

			gitInfo.WithLabelValues("repo-info", commitHash, author, timestamp, branchName, message).Set(1)
			time.Sleep(5 * time.Minute)
		}
	}()
}

func registerSystemMetrics() {
	prometheus.MustRegister(cpuUsage)
	prometheus.MustRegister(memoryUsage)

	go func() {
		for {
			if cpu, err := cpu.Percent(time.Second, false); err == nil {
				cpuUsage.Set(cpu[0])
			}

			if memory, err := mem.VirtualMemory(); err == nil {
				memoryUsage.Set(memory.UsedPercent)
			}

			time.Sleep(15 * time.Second)
		}
	}()
}

func main() {
	log.Println("starting the server")
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}
	registerGitHubMetrics()
	registerSystemMetrics()

	http.Handle("/metrics", promhttp.Handler())
	if err := http.ListenAndServe(":9090", nil); err != nil {
		log.Fatalf("Server failed to start: %v", err) // Log error if server fails to start
	}
}
