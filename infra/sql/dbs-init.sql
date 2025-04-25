DROP USER IF EXISTS stocks_config;
CREATE USER stocks_config WITH PASSWORD 'stocks_config';
DROP DATABASE IF EXISTS stocks_config;
CREATE DATABASE stocks_config OWNER stocks_config ENCODING = 'UTF8';
