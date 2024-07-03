package main

import (
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	_ "github.com/go-sql-driver/mysql"
)

type Response struct {
	Code int         `json:"code"`
	Data interface{} `json:"data"`
	Msg  string      `json:"msg"`
}

type URLResponse struct {
	Response interface{} `json:"response"`
}

var db *sql.DB

func initDB() {
	var err error
	dsn := "root:root@tcp(db:3306)/url"
	db, err = sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("Error opening database: %v", err)
	}
	if err = db.Ping(); err != nil {
		log.Fatalf("Error connecting to database: %v", err)
	}
}

func getURLs(w http.ResponseWriter, r *http.Request) {
	initDB()
	sql := "SELECT * FROM shortened_url"
	rows, err := db.Query(sql)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var data []map[string]interface{}
	for rows.Next() {
		columns, _ := rows.Columns()
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range columns {
			valuePtrs[i] = &values[i]
		}
		rows.Scan(valuePtrs...)
		entry := make(map[string]interface{})
		for i, col := range columns {
			entry[col] = values[i]
		}
		data = append(data, entry)
	}

	response := Response{Code: 0, Data: data, Msg: "success"}
	json.NewEncoder(w).Encode(response)
	db.Close()
}

func getURL(w http.ResponseWriter, r *http.Request) {
	initDB()
	currURL := r.URL.Path[len("/shorten/"):]
	sha256Hash := sha256.Sum256([]byte(currURL))
	base64Encoded := base64.URLEncoding.EncodeToString(sha256Hash[:])[:6]

	sqll := fmt.Sprintf("SELECT * FROM shortened_url WHERE short_url = '%s'", base64Encoded)
	row := db.QueryRow(sqll)
	//var longURL, shortURL string
	var result map[string]interface{}
	err := row.Scan(&result)
	//err := row.Scan(&longURL, &shortURL)
	if err != nil {
		if err == sql.ErrNoRows {
			json.NewEncoder(w).Encode(URLResponse{Response: "no result"})
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	json.NewEncoder(w).Encode(URLResponse{Response: result})
	db.Close()
}

func createURL(w http.ResponseWriter, r *http.Request) {
	initDB()
	currURL := r.URL.Path[len("/create/"):]
	sha256Hash := sha256.Sum256([]byte(currURL))
	base64Encoded := base64.URLEncoding.EncodeToString(sha256Hash[:])[:6]

	sql := fmt.Sprintf("INSERT INTO shortened_url (long_url, short_url) VALUES ('%s', '%s')", currURL, base64Encoded)
	_, err := db.Exec(sql)
	if err != nil {
		json.NewEncoder(w).Encode(URLResponse{Response: "error inserting"})
		return
	}

	json.NewEncoder(w).Encode(URLResponse{Response: "successful"})
	db.Close()
}

func redirect(w http.ResponseWriter, r *http.Request) {
	initDB()
	shortURL := r.URL.Path[len("/access/"):]
	sqll := fmt.Sprintf("SELECT long_url FROM shortened_url WHERE short_url = '%s'", shortURL)
	row := db.QueryRow(sqll)

	var longURL string
	err := row.Scan(&longURL)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "URL not found", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	http.Redirect(w, r, longURL, http.StatusFound)
	db.Close()
}

func main() {

	http.HandleFunc("/urls", getURLs)
	http.HandleFunc("/shorten/", getURL)
	http.HandleFunc("/create/", createURL)
	http.HandleFunc("/access/", redirect)

	log.Fatal(http.ListenAndServe(":8080", nil))
}
