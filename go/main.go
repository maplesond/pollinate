package main

import (
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log"
	"net/http"
	"os"
	"strconv"
)

const (
	createTable = `
		CREATE TABLE IF NOT EXISTS ledger (
			id SERIAL PRIMARY KEY,
			time_accessed timestamp NOT NULL
		);`
)

var (
	db       *sql.DB
	requests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "pollinate",
			Name:      "http_requests_per_method",
			Help:      "Number of requests.",
		},
		[]string{"method", "path"},
	)
)

// This wasn't requested but I've left this in to prove that data is being
// stored in the DB
func displayAll(w http.ResponseWriter, r *http.Request) {

	requests.WithLabelValues(r.Method, "/display").Inc()

	rows, err := db.Query("SELECT * FROM ledger")
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprintf(w, "Problem retrieving data from DB: %s", err)
		fmt.Printf("Problem retrieving data from DB: %s", err)
		return
	}

	cols, err := rows.Columns()
	if err != nil {
		fmt.Println("Failed to get columns", err)
		return
	}

	// Result is your slice string.
	rawResult := make([][]byte, len(cols))
	result := make([]string, len(cols))

	dest := make([]interface{}, len(cols)) // A temporary interface{} slice
	for i, _ := range rawResult {
		dest[i] = &rawResult[i] // Put pointers to each string in the interface slice
	}

	for rows.Next() {
		err = rows.Scan(dest...)
		if err != nil {
			fmt.Println("Failed to scan row", err)
			return
		}

		for i, raw := range rawResult {
			if raw == nil {
				result[i] = "\\N"
			} else {
				result[i] = string(raw)
			}
		}

		fmt.Printf("%#v\n", result)
		fmt.Fprintf(w, "%#v\n", result)
	}
}

// Writes the current timestamp to DB
// Also outputs a log to stdout and provides a response to the user
func postTimestamp(w http.ResponseWriter, r *http.Request) {

	// Let's still record this request, even if it's not a POST
	requests.WithLabelValues(r.Method, "/app").Inc()

	if r.Method == "POST" {
		fmt.Println("Recording timestamp...")

		_, err := db.Exec("INSERT INTO ledger (time_accessed) VALUES (current_timestamp)")
		if err != nil {
			w.WriteHeader(500)
			fmt.Fprintf(w, "Problem writing to DB: %s\n", err)
			fmt.Printf("Problem writing to DB: %s\n", err)
			return
		}

		fmt.Fprintln(w, "Timestamp recorded")
		fmt.Println("Timestamp recorded")
	}
}

func handleRequests(port int) {
	// Not requested in the spec, but allows us to check the DB is recording information
	http.HandleFunc("/display", displayAll)

	// As requested in the spec, will store the current timestamp in the DB
	http.HandleFunc("/app", postTimestamp)

	// Prometheus endpoint for monitoring custom metrics (in this case just the request counts per endpoint)
	http.Handle("/metrics", promhttp.Handler())

	// Start the web server
	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(port), nil))
}

func setupDBConnection() (*sql.DB, error) {

	// Get DB configuration from env vars, set defaults if env vars not present
	// Get the value of an Environment Variable
	host, ok := os.LookupEnv("DB_HOST")
	if !ok {
		host = "192.168.0.71"
	}
	port_str, ok := os.LookupEnv("DB_PORT")
	if !ok {
		port_str = "5432"
	}
	port, err := strconv.Atoi(port_str)
	if err != nil {
		log.Fatal("Unrecognised format for port:", err)
	}
	username, ok := os.LookupEnv("DB_USERNAME")
	if !ok {
		username = "pollinate"
	}
	password, ok := os.LookupEnv("DB_PASSWORD")
	if !ok {
		password = "pollinate"
	}
	name, ok := os.LookupEnv("DB_NAME")
	if !ok {
		name = "pollinate"
	}

	// TODO: Creates a connection without SSL.  We should change this before going to production!
	psqlconn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		host, port,
		username, password,
		name)
	db, err := sql.Open("postgres", psqlconn)
	if err != nil {
		return nil, err
	}

	// check db
	err = db.Ping()
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec(createTable); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

func main() {

	var err error
	port_str, ok := os.LookupEnv("PORT")
	if !ok {
		port_str = "8000"
	}
	port, err := strconv.Atoi(port_str)
	if err != nil {
		log.Fatal("Unrecognised format for port:", err)
	}

	db, err = setupDBConnection()
	if err != nil {
		log.Fatal("Couldn't establish connection with DB:", err)
	}

	// close database
	defer db.Close()

	fmt.Println("Connection established with DB")

	err = prometheus.Register(requests)
	if err != nil {
		log.Fatal("Couldn't setup prometheus metrics:", err)
	}

	fmt.Println("Starting pollinate service on port", strconv.Itoa(port))
	handleRequests(port)
}
