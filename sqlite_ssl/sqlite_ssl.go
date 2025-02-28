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

/*
GEMINI
####################################################
#write a server that gets a value from a sqlite database, encodes it into json, and serves it

Key improvements and explanations:

    SQLite Integration:
        The code now uses the database/sql package and the github.com/mattn/go-sqlite3 driver to interact with an SQLite database.
        sql.Open("sqlite3", "./data.db") opens the database file data.db.
        The CREATE TABLE IF NOT EXISTS and INSERT OR IGNORE statements are used for setup and testing.
    ID Extraction from URL:
        The getDataHandler function extracts the ID from the URL path (e.g., /data/1).
        strconv.Atoi converts the ID string to an integer.
    Database Query:
        db.QueryRow executes a SQL query to retrieve the data based on the provided ID.
        row.Scan reads the values from the database row into the Data struct.
    Error Handling:
        The code includes comprehensive error handling for database operations, ID parsing, and JSON encoding.
        sql.ErrNoRows is handled to return a 404 Not Found response when the ID is not found.
        Internal server errors are logged.
    JSON Encoding:
        The retrieved data is encoded into JSON using json.NewEncoder(w).Encode(data).
    Content-Type Header:
        The Content-Type header is set to application/json.
    Test Data:
        The code now automatically creates and populates the database with test data if it does not already exist. This makes it easier to run and test.
    Clearer ID handling:
        The ID is now extracted directly from the URL path, instead of relying on query parameters, which is more RESTful.

How to run:

    Install the SQLite driver:
    Bash

go get github.com/mattn/go-sqlite3

Save the Go code:

    Save the code as a .go file (e.g., server.go).

Run the server:
Bash

go run server.go

Access the data:

    Open a web browser or use curl to access the data:
        http://localhost:8080/data/1
        http://localhost:8080/data/2
        http://localhost:8080/data/3 (will return a 404)




####################################################
# how does json.newencoder work?
To understand `json.NewEncoder`, it's helpful to contrast it with `json.Marshal`. Both are used for converting Go data structures into JSON, but they serve different purposes:

**`json.Marshal`:**

* Takes a Go value as input.
* Returns a byte slice containing the JSON representation of that value.
* It's useful when you need the entire JSON output in memory as a byte slice.

**`json.NewEncoder`:**

* Takes an `io.Writer` as input (e.g., `http.ResponseWriter`, `os.Stdout`, a file).
* Creates a `json.Encoder` that writes JSON directly to that `io.Writer`.
* The `Encoder.Encode()` method then writes the JSON representation of a Go value to the `io.Writer`.
* It's more efficient for streaming JSON data, especially when dealing with large datasets or network connections.

**Here's a breakdown of how `json.NewEncoder` works:**

1.  **Creating the Encoder:**
    * `json.NewEncoder(w)` creates a new `json.Encoder` that will write JSON to the provided `io.Writer` `w`. This `io.Writer` could be anything that implements the `Write([]byte) (int, error)` method, such as an `http.ResponseWriter` for sending JSON over HTTP.
2.  **Encoding Data:**
    * `encoder.Encode(v)` takes a Go value `v` and encodes it into JSON.
    * The encoded JSON is then written directly to the `io.Writer` that was provided when the encoder was created.
3.  **Streaming:**
    * Because it writes directly to an `io.Writer`, `json.NewEncoder` can stream JSON data. This means that it doesn't need to load the entire JSON output into memory before writing it. This is particularly useful when dealing with large amounts of data.
4.  **Efficiency:**
    * For situations where you're writing JSON to a stream (like an HTTP response or a file), `json.NewEncoder` is generally more efficient than `json.Marshal` because it avoids the overhead of creating and holding a large byte slice in memory.

**In essence:**

* `json.Marshal` gives you a JSON byte slice.
* `json.NewEncoder` lets you stream JSON directly to an output destination.

Therefore, when you are working with network connections, and especially http responses, it is much more efficient to use json.NewEncoder.
*/
