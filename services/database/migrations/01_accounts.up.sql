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
    last_update TIMESTAMP DEFAULT NOW()
);

CREATE TABLE account_document
(
    account_id  SERIAL REFERENCES account (id) ON UPDATE CASCADE ON DELETE CASCADE,
    document_id SERIAL REFERENCES document (id) ON UPDATE CASCADE ON DELETE CASCADE
);
