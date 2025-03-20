CREATE TABLE IF NOT EXISTS cache_geolocation (
	ip TEXT PRIMARY KEY,
	city TEXT,
	region TEXT,
	country TEXT,
	loc TEXT,
	org TEXT,
	postal TEXT,
	timezone TEXT
);
