package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"golang.org/x/sync/semaphore"
)

type Worker struct {
	semaphore *semaphore.Weighted
	busy      bool
}

type WorkerStore struct {
	workers   []*Worker
	semaphore *semaphore.Weighted
}

func (ws *WorkerStore) releaseWorker() {
	ws.semaphore.Release(1)
}

func createWorkerStore(noOfWorkers int) *WorkerStore {
	ws := &WorkerStore{
		workers:   make([]*Worker, noOfWorkers),
		semaphore: semaphore.NewWeighted(int64(noOfWorkers)),
	}

	for i := 0; i < noOfWorkers; i++ {
		ws.workers[i] = &Worker{
			semaphore: ws.semaphore,
			busy:      false,
		}
	}

	return ws
}

func (w *Worker) multiplyMatrices(row []int, col []int) int {
	w.semaphore.Acquire(context.Background(), 1)
	w.busy = true

	defer func() {
		w.semaphore.Release(1)
		w.busy = false
	}()

	result := 0

	for i := 0; i < len(row); i++ {
		result += row[i] * col[i]
	}

	return result
}

func (ws *WorkerStore) getWorker() *Worker {
	for ws.semaphore.TryAcquire(1) == false {

	}
	defer ws.semaphore.Release(1)
	for _, worker := range ws.workers {
		if !worker.busy {
			return worker
		}
	}
	return nil
}

func multiplyMatrices(matrix1 [][]int, matrix2 [][]int, ws *WorkerStore) [][]int {
	n := len(matrix1)
	m := len(matrix2[0])

	result := make([][]int, n)
	for i := 0; i < n; i++ {
		result[i] = make([]int, m)
	}

	for i := 0; i < n; i++ {
		for j := 0; j < m; j++ {
			row := matrix1[i]
			col := make([]int, n)
			for k := 0; k < n; k++ {
				col[k] = matrix2[k][j]
			}

			worker := ws.getWorker()
			result[i][j] = worker.multiplyMatrices(row, col)
		}
	}
	return result
}

func printMatrix(matrix [][]int, name string) {
	fmt.Println()
	fmt.Println("Matrix " + name + ":")
	for _, row := range matrix {
		fmt.Println(row)
	}
}

type Request struct {
	Matrix1 [][]int `json:"matrix1"`
	Matrix2 [][]int `json:"matrix2"`
}

type Response struct {
	Result [][]int `json:"result"`
}

func multiplyHandler(w http.ResponseWriter, r *http.Request, ws *WorkerStore) {

	if r.Method != "POST" {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	decoder := json.NewDecoder(r.Body)

	var request Request

	err := decoder.Decode(&request)

	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	defer r.Body.Close()

	printMatrix(request.Matrix1, "A")
	printMatrix(request.Matrix2, "B")

	result := multiplyMatrices(request.Matrix1, request.Matrix2, ws)

	printMatrix(result, "Result")
	response := Response{Result: result}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(response)
}

func numberHandler(w http.ResponseWriter, r *http.Request) int {
	queryParams := r.URL.Query()
	number := queryParams.Get("number")
	num, _ := strconv.Atoi(number)
	return num
}

func main() {
	ws := createWorkerStore(10)
	http.HandleFunc("/multiply", func(w http.ResponseWriter, r *http.Request) {
		multiplyHandler(w, r, ws)
	})
	http.HandleFunc("/change", func(w http.ResponseWriter, r *http.Request) {
		number := numberHandler(w, r)
		ws = createWorkerStore(number)
		fmt.Println("ws with", number, "workers")
	})
	fmt.Println("Server started on port 8080")
	http.ListenAndServe(":8080", nil)
}
