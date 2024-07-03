package main

import (
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"log"
	"time"
	"net/http"
	"context"
	"regexp"

	"github.com/go-redis/redis/v8"

	_ "github.com/go-sql-driver/mysql"
)

// lru cache usage
const (
    CacheKey  = "lru_cache_keys"
    MaxCacheSize = 5 
)

// general response structure
type Response struct {
	Code int         `json:"code"`
	Data interface{} `json:"data"`
	Msg  string      `json:"msg"`
}

// url response structure
type URLResponse struct {
	Response interface{} `json:"response"`
}

// global var for storage connection
var db *sql.DB
var rdb *redis.Client
var ctx context.Context

// configuration imports
type Config struct {
    Username string `json:"username"`
    Password string `json:"password"`
    Protocol string `json:"protocol"`
    Host     string `json:"host"`
    Port     int    `json:"port"`
    Database string `json:"database"`
	Redis string `json:"redisInstance"`
	RedisPort string `json:"redisPort"`
}

var config Config


// DB connection initialization
func initDB() {
	var err error
	//dsn := "root:root@tcp(db:3306)/url"
	dsn := fmt.Sprintf("%s:%s@%s(%s:%d)/%s", config.Username, config.Password, config.Protocol, config.Host, config.Port, config.Database)
	db, err = sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("Error opening database: %v", err)
	}
	if err = db.Ping(); err != nil {
		log.Fatalf("Error connecting to database: %v", err)
	}
}

// Cache connection initialization
func initRedis() {
	address := fmt.Sprintf("%s:%s", config.Redis, config.RedisPort)
	rdb = redis.NewClient(&redis.Options{
        Addr:     address, 
        Password: "",               
        DB:       0,                
    })
    _, err := rdb.Ping(rdb.Context()).Result()
	ctx = context.Background()
    if err != nil {
        log.Fatalf("Could not connect to Redis: %v", err)
    }
}

// get all urls
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
		values := make([]string, len(columns))
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

// convert a long url while not inserting into storage
func getURL(w http.ResponseWriter, r *http.Request) {
	currURL := r.URL.Path[len("/shorten/"):]
	urlPattern := `^(?:http(?:s)?://)?(?:www\.)?[a-zA-Z0-9-]+\.[a-zA-Z]{2,}$`
	regex := regexp.MustCompile(urlPattern)
	if !regex.MatchString(currURL) {
		json.NewEncoder(w).Encode(URLResponse{Response: "url not valid"})
		return
	}
	sha256Hash := sha256.Sum256([]byte(currURL))
	base64Encoded := base64.URLEncoding.EncodeToString(sha256Hash[:])[:8]
	json.NewEncoder(w).Encode(URLResponse{Response: base64Encoded})
}

// convert a long url while inserting into storage
func createURL(w http.ResponseWriter, r *http.Request) {
	currURL := r.URL.Path[len("/create/"):]
	urlPattern := `^(?:http(?:s)?://)?(?:www\.)?[a-zA-Z0-9-]+\.[a-zA-Z]{2,}$`
	regex := regexp.MustCompile(urlPattern)
	if !regex.MatchString(currURL) {
		json.NewEncoder(w).Encode(URLResponse{Response: "url not valid"})
		return
	}
	initDB()
	initRedis()
	sha256Hash := sha256.Sum256([]byte(currURL))
	base64Encoded := base64.URLEncoding.EncodeToString(sha256Hash[:])[:8]
	cacheOperations(rdb, base64Encoded, currURL)
	sqll := fmt.Sprintf("INSERT INTO shortened_url (long_url, short_url) VALUES ('%s', '%s')", currURL, base64Encoded)
	_, sqlErr := db.Exec(sqll)
	if sqlErr != nil {
		json.NewEncoder(w).Encode(URLResponse{Response: sqlErr})
		panic(sqlErr)
	}

	json.NewEncoder(w).Encode(URLResponse{Response: "your tinyURL: " + base64Encoded,})
	db.Close()
}

// redirect the user to long url (need to access from browser)
func redirect(w http.ResponseWriter, r *http.Request) {
	
	initRedis()
	
	shortURL := r.URL.Path[len("/access/"):]
	cached, cacheErr := getCache(rdb, shortURL)
	
	if cacheErr == nil {
		http.Redirect(w, r, "http://" + cached, http.StatusFound)
	}else{
		initDB()
		sqll := fmt.Sprintf("SELECT long_url FROM shortened_url WHERE short_url = '%s'", shortURL)
		fmt.Println("Query:", sqll)
		row := db.QueryRow(sqll)
	
		var longURL string
		err := row.Scan(&longURL)
		if err != nil {
			if err == sql.ErrNoRows {
				http.Error(w, "URL not found", http.StatusNotFound)
			} else {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			panic(err)
		}
		log.Printf(longURL)
		http.Redirect(w, r, "http://" + longURL, http.StatusFound)
		db.Close()
	}
	
}

// cache operation entry
func cacheOperations(rdb *redis.Client, key string, value string) {
    err := setCache(rdb, key, value)
    if err != nil {
        log.Printf("Error setting value in LRU cache: %v", err)
    }
}

// cache setting main method
func setCache(rdb *redis.Client, key string, value string) error {
    err := rdb.Set(ctx, key, value, 0).Err()

    if err != nil {
        return fmt.Errorf("failed to set key %s in Redis: %v", key, err)
    }

    err = manageLRUEviction(rdb, key)
    if err != nil {
        return err
    }

    return nil
}

// cache extraction
func getCache(rdb *redis.Client, key string) (string, error) {

    val, err := rdb.Get(ctx, key).Result()
    if err != nil {
        return "", fmt.Errorf("failed to get key %s from Redis: %v", key, err)
    }
    return val, nil
}

// manage cache expiration
func manageLRUEviction(rdb *redis.Client, key string) error {
    err := rdb.ZAdd(ctx, CacheKey, &redis.Z{
        Score:  float64(time.Now().Unix()),
        Member: key,
    }).Err()
    if err != nil {
        return fmt.Errorf("failed to add key %s to LRU cache: %v", key, err)
    }

    length, err := rdb.ZCard(ctx, CacheKey).Result()
    if err != nil {
        return fmt.Errorf("failed to get LRU cache length: %v", err)
    }

    if length > MaxCacheSize {
        keys, err := rdb.ZRange(ctx, CacheKey, 0, 0).Result()
        if err != nil {
            return fmt.Errorf("failed to get least recently used key: %v", err)
        }

        if len(keys) > 0 {
            _, err := rdb.Del(ctx, keys[0]).Result()
            if err != nil {
                return fmt.Errorf("failed to delete key %s from LRU cache: %v", keys[0], err)
            }

            _, err = rdb.ZRem(ctx, CacheKey, keys[0]).Result()
            if err != nil {
                return fmt.Errorf("failed to remove key %s from LRU cache keys: %v", keys[0], err)
            }
        }
    }

    return nil
}

func main() {

	http.HandleFunc("/urls", getURLs)
	http.HandleFunc("/shorten/", getURL)
	http.HandleFunc("/create/", createURL)
	http.HandleFunc("/access/", redirect)

	configFile, err := os.Open("config.json")
    if err != nil {
        fmt.Println("Error opening config file:", err)
        return
    }
    decoder := json.NewDecoder(configFile)
    err = decoder.Decode(&config)
    if err != nil {
        fmt.Println("Error decoding config JSON:", err)
        return
    }
    defer configFile.Close()

	log.Fatal(http.ListenAndServe(":8080", nil))
}
