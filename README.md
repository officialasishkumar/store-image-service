# Store Image Processing Service

Github link: https://github.com/officialasishkumar/store-image-service

## Description
A Go-based web service for processing store visits and images. The service allows:
- Submitting jobs with store visits and image URLs
- Tracking job status
- Validating store IDs against a master list
- Downloading and analyzing images

## Setup and Installation

### Docker Deployment

1. Place `StoreMasterAssignment.csv` in the project directory

2. Build and run the Docker container:
   ```bash
   docker build -t store-image-service .
   docker run -p 8080:8080 store-image-service
   ```

### Manual Setup
1. Go 1.20+
2. Run:
   ```bash
   go mod tidy
   go run main.go
   ```

## Endpoints
- `/api/submit/` (POST): Submit job
- `/api/status` (GET): Check job status

## Potential Improvements
- Add basic logging
- Implement configurable job timeout
- Create basic input validation middleware
- Implement graceful server shutdown