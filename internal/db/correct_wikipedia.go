package db

import "fmt"

// correctFlowerWikipedia fixes known-bad Wikipedia URLs.
// Safe to run repeatedly — only updates URLs that match the old value.
func correctFlowerWikipedia(d *DB) error {
	fixes := []struct{ old, new string }{
		{
			"https://en.wikipedia.org/wiki/Canna_x_generalis",
			"https://en.wikipedia.org/wiki/Canna_(plant)",
		},
		{
			"https://en.wikipedia.org/wiki/Lysimachia_nummularia_'Aurea'",
			"https://en.wikipedia.org/wiki/Lysimachia_nummularia",
		},
		{
			"https://en.wikipedia.org/wiki/Rosmarinus_officinalis_'Prostratus'",
			"https://en.wikipedia.org/wiki/Salvia_rosmarinus",
		},
	}

	for _, f := range fixes {
		res, err := d.Exec(`UPDATE flowers SET wikipedia_url = ? WHERE wikipedia_url = ?`, f.new, f.old)
		if err != nil {
			return fmt.Errorf("wikipedia correction: %w", err)
		}
		n, _ := res.RowsAffected()
		_ = n // debug: log rows affected if needed
	}
	return nil
}
