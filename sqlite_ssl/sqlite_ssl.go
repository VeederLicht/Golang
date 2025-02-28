package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"strconv"

	_ "github.com/mattn/go-sqlite3" // Import the SQLite driver
)

type Data struct {
	ID    int    `json:"id"`
	Value string `json:"value"`
}

var db *sql.DB
var httpport = "8080"
var httpsport = "8443"

func main() {
	var err error
	db, err = sql.Open("sqlite3", "./data.db") // Open the SQLite database
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Create a table if it doesn't exist (for testing)
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS mydata (id INTEGER PRIMARY KEY, value TEXT)")
	if err != nil {
		log.Fatal(err)
	}

	// Insert some test data (for testing)
	_, err = db.Exec("INSERT OR IGNORE INTO mydata (id, value) VALUES (1, 'test value'), (2, 'another value')")
	if err != nil {
		log.Fatal(err)
	}

	go func() { // Start an HTTP server to redirect to HTTPS
		log.Println("HTTP server listening on ", httpport)
		err := http.ListenAndServe(":"+httpport, http.HandlerFunc(redirectToHTTPS))
		if err != nil {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	http.HandleFunc("/data/", sqliteHandler)  // Handle requests to /data/ID
	http.HandleFunc("/simple", simpleHandler) // Handle requests to /data/ID
	log.Println("Server listening on ", httpsport)
	err = http.ListenAndServeTLS(":"+httpsport, "server.crt", "server.key", nil) // ListenAndServeTLS: listen tcp :443: bind: permission denied
	if err != nil {
		log.Fatal("ListenAndServeTLS: ", err)
	}
}

func redirectToHTTPS(w http.ResponseWriter, r *http.Request) {
	//https://stackoverflow.com/questions/37536006/how-do-i-rewrite-redirect-from-http-to-https-in-go#answer-53496743
	h, _, _ := net.SplitHostPort(r.Host)
	u := r.URL
	u.Host = net.JoinHostPort(h, httpsport)
	u.Scheme = "https"
	target := u.String()
	log.Println("Redirecting to: ", target)
	http.Redirect(w, r, target, http.StatusMovedPermanently) // or StatusPermanentRedirect
}

func sqliteHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Request received: ", r.URL.Path)
	idStr := r.URL.Path[len("/data/"):] // Extract the ID from the URL
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	row := db.QueryRow("SELECT id, value FROM mydata WHERE id = ?", id)
	var data Data
	err = row.Scan(&data.ID, &data.Value)
	if err == sql.ErrNoRows {
		http.NotFound(w, r)
		return
	} else if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		log.Println(err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(data)
	if err != nil {
		log.Println(err)
	}
}

func simpleHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Request received: ", r.URL.Path)
	w.Write([]byte("This is a simple page!"))
}
