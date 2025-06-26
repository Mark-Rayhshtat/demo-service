package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

type Item struct {
	ID    int     `json:"id"`
	Name  string  `json:"name"`
	Price float64 `json:"price"`
}

type App struct {
	DB     *sql.DB
	Router *mux.Router
}

// Initialize initializes the app with predefined configuration
func (app *App) Initialize() {
	// Get PostgreSQL connection details from environment variables
	user := getEnv("POSTGRES_USER", "postgres")
	password := getEnv("POSTGRES_PASSWORD", "postgres")
	dbName := getEnv("POSTGRES_DB", "postgres")
	host := getEnv("POSTGRES_HOST", "localhost")
	port := getEnv("POSTGRES_PORT", "5432")

	// Build PostgreSQL connection string with SSL mode enabled
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=require",
		host, port, user, password, dbName)

	var err error
	// Open a connection to the database
	app.DB, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Error connecting to database: %v", err)
	}

	// Check if the connection is valid
	if err = app.DB.Ping(); err != nil {
		log.Fatalf("Could not ping database: %v", err)
	}

	// Create table if it does not exist
	createTableQuery := `
	CREATE TABLE IF NOT EXISTS items (
		id SERIAL PRIMARY KEY,
		name TEXT NOT NULL,
		price NUMERIC(10,2) NOT NULL
	);`

	_, err = app.DB.Exec(createTableQuery)
	if err != nil {
		log.Fatalf("Could not create table: %v", err)
	}

	// Initialize router
	app.Router = mux.NewRouter()
	app.setRouters()
}

// getEnv gets the environment variable with a default value
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

// setRouters sets up the API routes
func (app *App) setRouters() {
	app.Router.HandleFunc("/api/v1/data", app.getData).Methods("GET")
	app.Router.HandleFunc("/api/v1/data", app.createData).Methods("POST")
	app.Router.HandleFunc("/healthz", app.healthcheck).Methods("GET")
}

// healthcheck provides a simple health check endpoint
func (app *App) healthcheck(w http.ResponseWriter, r *http.Request) {
	// Check if the database connection is alive
	err := app.DB.Ping()
	if err != nil {
		respondWithError(w, http.StatusServiceUnavailable, "Database connection failed")
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// getData retrieves an item by ID
func (app *App) getData(w http.ResponseWriter, r *http.Request) {
	// Get ID from query parameters
	idParam := r.URL.Query().Get("id")
	if idParam == "" {
		respondWithError(w, http.StatusBadRequest, "Missing id parameter")
		return
	}

	id, err := strconv.Atoi(idParam)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid id parameter")
		return
	}

	item := Item{ID: id}
	err = app.DB.QueryRow("SELECT name, price FROM items WHERE id = $1", id).
		Scan(&item.Name, &item.Price)

	if err == sql.ErrNoRows {
		respondWithError(w, http.StatusNotFound, "Item not found")
		return
	} else if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, item)
}

// createData creates a new item
func (app *App) createData(w http.ResponseWriter, r *http.Request) {
	var item Item
	err := json.NewDecoder(r.Body).Decode(&item)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	defer r.Body.Close()

	// Insert item into database
	err = app.DB.QueryRow("INSERT INTO items(name, price) VALUES($1, $2) RETURNING id",
		item.Name, item.Price).Scan(&item.ID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusCreated, item)
}

// respondWithError responds with an error message
func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, map[string]string{"error": message})
}

// respondWithJSON responds with JSON data
func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, _ := json.Marshal(payload)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}

// Run starts the application
func (app *App) Run(addr string) {
	log.Printf("Server starting on %s", addr)
	log.Fatal(http.ListenAndServe(addr, app.Router))
}

func main() {
	app := App{}
	app.Initialize()
	app.Run(":8080")
}
