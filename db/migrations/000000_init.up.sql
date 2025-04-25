-- create table progress with primary key symbol and column last_update
CREATE TABLE progress
(
    symbol      VARCHAR(255) PRIMARY KEY,
    last_update TIMESTAMP
);