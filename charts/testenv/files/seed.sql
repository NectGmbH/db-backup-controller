CREATE DATABASE IF NOT EXISTS {{ .Values.crdbDB }};

USE {{ .Values.crdbDB }};

CREATE TABLE IF NOT EXISTS kv (
  key string PRIMARY KEY,
  value string
);

UPSERT INTO kv (key, value) VALUES ('inserted', '{{ now }}');
