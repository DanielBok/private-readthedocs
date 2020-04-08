CREATE TABLE account
(
    id       SERIAL PRIMARY KEY,
    username VARCHAR(255) UNIQUE CHECK ( length(username) >= 4 ) NOT NULL,
    password VARCHAR(255) CHECK ( length(password) >= 4 )        NOT NULL,
    is_admin BOOLEAN DEFAULT FALSE
);

CREATE TABLE document
(
    id          SERIAL PRIMARY KEY,
    name        VARCHAR(255) UNIQUE CHECK ( length(name) >= 1 ) NOT NULL,
    last_update TIMESTAMP DEFAULT NOW(),
    account_id  INT REFERENCES account (id) ON UPDATE CASCADE ON DELETE CASCADE
);
