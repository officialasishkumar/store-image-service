package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"time"
)

// store_id -> {store_name, area_code}
var StoreMaster = make(map[string]struct {
	StoreName string
	AreaCode  string
})

// represents job to process images
type Job struct {
	JobID  int
	Visits []Visit
	Status string 
	Errors []JobError
	mu     sync.Mutex
}

// represents store visit with image linkss
type Visit struct {
	StoreID   string   `json:"store_id"`
	ImageURLs []string `json:"image_url"`
	VisitTime string   `json:"visit_time"`
}

// represents error for store_id
type JobError struct {
	StoreID string `json:"store_id"`
	Error   string `json:"error"`
}

// represents response for a a post req
type JobResponse struct {
	JobID int `json:"job_id"`
}

// represents response for job status req
type JobStatusResponse struct {
	Status string     `json:"status"`
	JobID  string     `json:"job_id"`
	Errors []JobError `json:"error,omitempty"`
}

var (
	jobs      = make(map[int]*Job)
	jobIDSeq  = 0
	jobsMutex sync.Mutex
)

func main() {
	if err := loadStoreMaster("StoreMasterAssignment.csv"); err != nil {
		fmt.Println("Failed to load StoreMaster:", err)
		return
	}
	http.HandleFunc("/api/submit/", submitJobHandler)
	http.HandleFunc("/api/status", getJobStatusHandler)
	fmt.Println("Server started at :8080")
	http.ListenAndServe(":8080", nil)
}

func loadStoreMaster(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open CSV file: %v", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = 3 

	if _, err := reader.Read(); err != nil {
		return fmt.Errorf("failed to read CSV header: %v", err)
	}

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read CSV record: %v", err)
		}

		areaCode := record[0]
		storeName := record[1]
		storeID := record[2]

		StoreMaster[storeID] = struct {
			StoreName string
			AreaCode  string
		}{
			StoreName: storeName,
			AreaCode:  areaCode,
		}
	}

	return nil
}

func submitJobHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		Count  int     `json:"count"`
		Visits []Visit `json:"visits"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	if request.Count != len(request.Visits) {
		http.Error(w, "Count does not match the number of visits", http.StatusBadRequest)
		return
	}

	jobsMutex.Lock()
	jobIDSeq++
	job := &Job{
		JobID:  jobIDSeq,
		Visits: request.Visits,
		Status: "ongoing",
	}
	jobs[jobIDSeq] = job
	jobsMutex.Unlock()

	go processJob(job)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(JobResponse{JobID: job.JobID})
}

func processJob(job *Job) {
	for _, visit := range job.Visits {
		if _, exists := StoreMaster[visit.StoreID]; !exists {
			job.mu.Lock()
			job.Status = "failed"
			job.Errors = append(job.Errors, JobError{StoreID: visit.StoreID, Error: "Invalid store_id"})
			job.mu.Unlock()
			continue
		}

		for _, url := range visit.ImageURLs {
			img, err := downloadImage(url)
			if err != nil {
				job.mu.Lock()
				job.Status = "failed"
				job.Errors = append(job.Errors, JobError{StoreID: visit.StoreID, Error: fmt.Sprintf("Failed to download image: %s", err)})
				job.mu.Unlock()
				continue
			}

			perimeter := 2 * (img.Bounds().Dx() + img.Bounds().Dy())

			time.Sleep(time.Duration(100+rand.Intn(300)) * time.Millisecond)

			fmt.Printf("Store ID: %s, Image URL: %s, Perimeter: %d\n", visit.StoreID, url, perimeter)
		}
	}

	job.mu.Lock()
	if job.Status != "failed" {
		job.Status = "completed"
	}
	job.mu.Unlock()
}

func downloadImage(url string) (image.Image, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download image: %s", resp.Status)
	}

	img, _, err := image.Decode(resp.Body)
	if err != nil {
		return nil, err
	}

	return img, nil
}

func getJobStatusHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	jobID := r.URL.Query().Get("jobid")
	if jobID == "" {
		http.Error(w, "Missing jobid parameter", http.StatusBadRequest)
		return
	}

	jobsMutex.Lock()
	job, exists := jobs[atoi(jobID)]
	jobsMutex.Unlock()

	if !exists {
		http.Error(w, "Job not found", http.StatusBadRequest)
		return
	}

	response := JobStatusResponse{
		Status: job.Status,
		JobID:  jobID,
	}

	if job.Status == "failed" {
		response.Errors = job.Errors
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func atoi(s string) int {
	var result int
	fmt.Sscanf(s, "%d", &result)
	return result
}
