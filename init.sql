CREATE DATABASE IF NOT EXISTS url;

use url;

CREATE TABLE shortened_url (
    long_url VARCHAR(100) PRIMARY KEY,
    short_url VARCHAR(100) UNIQUE NOT NULL
);