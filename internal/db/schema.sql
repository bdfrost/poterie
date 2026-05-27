CREATE TABLE IF NOT EXISTS flowers (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT NOT NULL,
	botanical_name TEXT NOT NULL DEFAULT '',
	role TEXT NOT NULL CHECK(role IN ('thriller', 'filler', 'spiller')),
	zone_min INTEGER NOT NULL,
	zone_max INTEGER NOT NULL,
	sun TEXT NOT NULL CHECK(sun IN ('full_sun', 'part_shade', 'full_shade')),
	soils TEXT NOT NULL, -- comma-separated: clay,loam,sandy,well_drained
	color TEXT NOT NULL DEFAULT '',
	bloom_season TEXT NOT NULL DEFAULT 'summer',
	height TEXT NOT NULL DEFAULT '',
	spacing TEXT NOT NULL DEFAULT '',
	description TEXT NOT NULL DEFAULT '',
	image_url TEXT NOT NULL DEFAULT ''
);