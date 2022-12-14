package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
)

type Request struct {
	Message string `json:"message"`
}

func main() {
	fmt.Print("Starting server on port 8080\n")
	server()
}

func writeToBoard(content string) error {
	f, err := os.OpenFile("database.txt", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err = f.WriteString(fmt.Sprintf("%s\n", content)); err != nil {
		return err
	}
	return nil
}

func server() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			req := &Request{}
			err := json.NewDecoder(r.Body).Decode(req)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			err = writeToBoard(req.Message)
			if err != nil {
				fmt.Println(err)
				w.WriteHeader(http.StatusInternalServerError)
			}
			w.WriteHeader(http.StatusCreated)
		} else if r.Method == http.MethodGet {
			content, err := ioutil.ReadFile("database.txt")
			if err != nil {
				panic(err)
			}
			fmt.Fprintf(w, string(content))
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
	})

	if err := http.ListenAndServe(":8080", nil); err != http.ErrServerClosed {
		fmt.Print("Server started on port 8080")
		panic(err)
	}
}
