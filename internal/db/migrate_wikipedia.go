package db

import "fmt"

// migrateFlowerWikipedia adds Wikipedia URLs to all seeded flowers.
// Safe to run repeatedly — skips rows that already have a URL.
func migrateFlowerWikipedia(d *DB) error {
	type entry struct{ name, url string }
	entries := []entry{
		{"Canna Lily", "https://en.wikipedia.org/wiki/Canna_x_generalis"},
		{"Angelonia", "https://en.wikipedia.org/wiki/Angelonia_angustifolia"},
		{"Salvia", "https://en.wikipedia.org/wiki/Salvia_splendens"},
		{"Celosia", "https://en.wikipedia.org/wiki/Celosia_argentea"},
		{"Dracaena Spike", "https://en.wikipedia.org/wiki/Cordyline_australis"},
		{"Ornamental Grass", "https://en.wikipedia.org/wiki/Pennisetum_setaceum"},
		{"Torenia", "https://en.wikipedia.org/wiki/Torenia_fournieri"},
		{"Castor Bean", "https://en.wikipedia.org/wiki/Ricinus_communis"},
		{"Astilbe", "https://en.wikipedia.org/wiki/Astilbe_x_arendsii"},
		{"Foxglove", "https://en.wikipedia.org/wiki/Digitalis_purpurea"},
		{"Delphinium", "https://en.wikipedia.org/wiki/Delphinium_elatum"},
		{"Banana Plant", "https://en.wikipedia.org/wiki/Musa_basjoo"},
		{"Hosta", "https://en.wikipedia.org/wiki/Hosta_spp."},
		{"Japanese Painted Fern", "https://en.wikipedia.org/wiki/Athyrium_niponicum"},
		{"Hollyhock", "https://en.wikipedia.org/wiki/Alcea_rosea"},
		{"Sunflower", "https://en.wikipedia.org/wiki/Helianthus_annuus"},
		{"Gladiolus", "https://en.wikipedia.org/wiki/Gladiolus_x_hortulanus"},
		{"Petunia", "https://en.wikipedia.org/wiki/Petunia_x_hybrida"},
		{"Geranium", "https://en.wikipedia.org/wiki/Pelargonium_x_hortorum"},
		{"Marigold", "https://en.wikipedia.org/wiki/Tagetes_erecta"},
		{"Begonia", "https://en.wikipedia.org/wiki/Begonia_x_semperflorens"},
		{"Coleus", "https://en.wikipedia.org/wiki/Plectranthus_scutellarioides"},
		{"Impatiens", "https://en.wikipedia.org/wiki/Impatiens_walleriana"},
		{"Zinnia", "https://en.wikipedia.org/wiki/Zinnia_elegans"},
		{"Dahlia", "https://en.wikipedia.org/wiki/Dahlia_x_hortensis"},
		{"Calibrachoa", "https://en.wikipedia.org/wiki/Calibrachoa_x_hybrida"},
		{"Vinca", "https://en.wikipedia.org/wiki/Catharanthus_roseus"},
		{"Nemesia", "https://en.wikipedia.org/wiki/Nemesia_strumosa"},
		{"Lobelia", "https://en.wikipedia.org/wiki/Lobelia_erinus"},
		{"Dusty Miller", "https://en.wikipedia.org/wiki/Senecio_cineraria"},
		{"Pansy", "https://en.wikipedia.org/wiki/Viola_x_wittrockiana"},
		{"Snapdragon", "https://en.wikipedia.org/wiki/Antirrhinum_majus"},
		{"Ornamental Pepper", "https://en.wikipedia.org/wiki/Capsicum_annuum"},
		{"Caladium", "https://en.wikipedia.org/wiki/Caladium_bicolor"},
		{"Lavender", "https://en.wikipedia.org/wiki/Lavandula_angustifolia"},
		{"Coneflower", "https://en.wikipedia.org/wiki/Echinacea_purpurea"},
		{"Black-eyed Susan", "https://en.wikipedia.org/wiki/Rudbeckia_hirta"},
		{"Creeping Jenny", "https://en.wikipedia.org/wiki/Lysimachia_nummularia"},
		{"Sweet Alyssum", "https://en.wikipedia.org/wiki/Lobularia_maritima"},
		{"Bacopa", "https://en.wikipedia.org/wiki/Sutera_cordata"},
		{"Trailing Verbena", "https://en.wikipedia.org/wiki/Verbena_x_hybrida"},
		{"Dichondra Silver Falls", "https://en.wikipedia.org/wiki/Dichondra_argentea"},
		{"Trailing Ivy", "https://en.wikipedia.org/wiki/Hedera_helix"},
		{"Mazus", "https://en.wikipedia.org/wiki/Mazus_reptans"},
		{"Nasturtium", "https://en.wikipedia.org/wiki/Tropaeolum_majus"},
		{"Lemon Thyme", "https://en.wikipedia.org/wiki/Thymus_citriodorus"},
		{"Creeping Thyme", "https://en.wikipedia.org/wiki/Thymus_serpyllum"},
		{"Scaevola", "https://en.wikipedia.org/wiki/Scaevola_aemula"},
		{"Sedum Angelina", "https://en.wikipedia.org/wiki/Sedum_rupestre"},
		{"Lotus Vine", "https://en.wikipedia.org/wiki/Lotus_berthelotii"},
		{"English Ivy", "https://en.wikipedia.org/wiki/Hedera_hibernica"},
		{"Creeping Jenny Gold", "https://en.wikipedia.org/wiki/Lysimachia_nummularia_'Aurea'"},
		{"Trailing Rosemary", "https://en.wikipedia.org/wiki/Rosmarinus_officinalis_'Prostratus'"},
		{"String of Pearls", "https://en.wikipedia.org/wiki/Senecio_rowleyanus"},
		{"Mint", "https://en.wikipedia.org/wiki/Mentha_spp."},
		{"Vinca Minor", "https://en.wikipedia.org/wiki/Vinca_minor"},
		{"Sweet Woodruff", "https://en.wikipedia.org/wiki/Galium_odoratum"},
		{"Ajuga", "https://en.wikipedia.org/wiki/Ajuga_reptans"},
	}

	for _, e := range entries {
		_, err := d.Exec(`UPDATE flowers SET wikipedia_url = ? WHERE name = ? AND (wikipedia_url IS NULL OR wikipedia_url = '')`, e.url, e.name)
		if err != nil {
			return fmt.Errorf("wikipedia migration for %s: %w", e.name, err)
		}
	}
	return nil
}
