package store

import (
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/zhifengle/rss2cloud/rsssite"
)

type Store struct {
	DBInstance *sql.DB
}

// New creates a new Store instance with the given database connection.
// If db is nil, it opens a SQLite database at the default path "db.sqlite".
// Deprecated: Use NewWithPath to specify a custom database path.
func New(db *sql.DB) *Store {
	if db == nil {
		store, err := NewWithPath("db.sqlite")
		if err != nil {
			panic(err)
		}
		return store
	}
	if err := initSchema(db); err != nil {
		panic(err)
	}
	if err := migrateSchema(db); err != nil {
		panic(err)
	}
	return &Store{
		DBInstance: db,
	}
}

// NewWithPath creates a new Store instance with a database at the specified path.
// If path is empty, it defaults to "db.sqlite".
func NewWithPath(path string) (*Store, error) {
	if path == "" {
		path = "db.sqlite"
	}
	if dir := filepath.Dir(path); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
	}
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}
	if err := initSchema(db); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := migrateSchema(db); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &Store{
		DBInstance: db,
	}, nil
}

func initSchema(db *sql.DB) error {
	if _, err := db.Exec("CREATE TABLE if not exists `rss_items` (`id` INTEGER PRIMARY KEY AUTOINCREMENT, `link` VARCHAR(255), `title` VARCHAR(255), `guid` VARCHAR(255), `pubDate` DATETIME, `creator` VARCHAR(255), `summary` TEXT, `content` VARCHAR(255), `isoDate` DATETIME, `categories` VARCHAR(255), `contentSnippet` VARCHAR(255), `done` TINYINT(1) DEFAULT 0, `magnet` VARCHAR(255) NOT NULL, `createdAt` DATETIME NOT NULL, `updatedAt` DATETIME NOT NULL)"); err != nil {
		return err
	}
	if _, err := db.Exec("CREATE TABLE if not exists `sites_status` (`id` INTEGER PRIMARY KEY AUTOINCREMENT, `name` VARCHAR(255), `needLogin` TINYINT(1), `abnormalOp` TINYINT(1), `createdAt` DATETIME NOT NULL, `updatedAt` DATETIME NOT NULL)"); err != nil {
		return err
	}
	return nil
}

// migrateSchema adds columns introduced after the initial schema to existing
// rss_items tables. Idempotent: skips columns that already exist.
func migrateSchema(db *sql.DB) error {
	cols, err := tableColumns(db, "rss_items")
	if err != nil {
		return err
	}
	additions := []struct {
		col string
		sql string
	}{
		{"configName", "ALTER TABLE `rss_items` ADD COLUMN `configName` VARCHAR(255)"},
		{"configUrl", "ALTER TABLE `rss_items` ADD COLUMN `configUrl` VARCHAR(255)"},
		{"taskHash", "ALTER TABLE `rss_items` ADD COLUMN `taskHash` VARCHAR(255)"},
	}
	for _, a := range additions {
		if cols[a.col] {
			continue
		}
		if _, err := db.Exec(a.sql); err != nil {
			return err
		}
	}
	return nil
}

// tableColumns returns the set of column names present in a table.
func tableColumns(db *sql.DB, table string) (map[string]bool, error) {
	rows, err := db.Query("PRAGMA table_info(`" + table + "`)")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	cols := make(map[string]bool)
	for rows.Next() {
		var cid, notNull, pk int
		var name, typ string
		var dflt any
		if err := rows.Scan(&cid, &name, &typ, &notNull, &dflt, &pk); err != nil {
			return nil, err
		}
		cols[name] = true
	}
	return cols, rows.Err()
}

// SaveMagnetItems records accepted magnet items with their RSS feed context
// and the non-empty 115 offline task hashes returned by OfflineAddUrl.
// Each item must have a corresponding hash.
func (s *Store) SaveMagnetItems(items []rsssite.MagnetItem, configName, configUrl string, hashes []string) error {
	if len(items) == 0 {
		return nil
	}
	if len(hashes) != len(items) {
		return fmt.Errorf("magnet item/hash count mismatch: %d items, %d hashes", len(items), len(hashes))
	}
	for i, hash := range hashes {
		if hash == "" {
			return fmt.Errorf("magnet item %d has an empty task hash", i)
		}
	}
	tx, err := s.DBInstance.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	now := time.Now()
	for i, item := range items {
		stmt := "INSERT INTO rss_items (`link`,`title`,`content`,`magnet`,`configName`,`configUrl`,`taskHash`,`done`,`createdAt`,`updatedAt`) VALUES (?,?,?,?,?,?,?,?,?,?)"
		_, err := tx.Exec(stmt, item.Link, item.Title, item.Content, item.Magnet, configName, configUrl, hashes[i], 0, now, now)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) HasItem(magnet string) bool {
	var count int
	s.DBInstance.QueryRow("SELECT count(*) AS num FROM rss_items WHERE magnet = ?", magnet).Scan(&count)
	return count > 0
}

// @TODO 替换 HasItem. 注意目前 magnet 存的长度是 VARCHAR(255)。有tracker的长URI会存不了.
func (s *Store) HasMagnetByXt(magnet string) bool {
	var count int
	u, err := url.Parse(magnet)
	if err != nil {
		return false
	}
	params := u.Query()
	xt := params.Get("xt")
	s.DBInstance.QueryRow("SELECT count(*) AS num FROM rss_items WHERE magnet LIKE ?", "%"+xt+"%").Scan(&count)
	return count > 0
}

// HistoryItem is a single submission record from the rss_items table.
type HistoryItem struct {
	ID         int64
	Title      string
	ConfigName string
	ConfigUrl  string
	CreatedAt  time.Time
	TaskHash   string
}

// ListHistory returns the most recent submission records, newest first.
// A non-empty name filters by RSS feed config name. limit <= 0 defaults to 20.
func (s *Store) ListHistory(limit int, name string) ([]HistoryItem, error) {
	if limit <= 0 {
		limit = 20
	}
	query := "SELECT id, COALESCE(title,''), COALESCE(configName,''), COALESCE(configUrl,''), createdAt, COALESCE(taskHash,'') FROM rss_items"
	var args []any
	if name != "" {
		query += " WHERE configName = ?"
		args = append(args, name)
	}
	query += " ORDER BY id DESC LIMIT ?"
	args = append(args, limit)
	rows, err := s.DBInstance.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []HistoryItem
	for rows.Next() {
		var h HistoryItem
		if err := rows.Scan(&h.ID, &h.Title, &h.ConfigName, &h.ConfigUrl, &h.CreatedAt, &h.TaskHash); err != nil {
			return nil, err
		}
		items = append(items, h)
	}
	return items, rows.Err()
}

func (s *Store) Close() error {
	return s.DBInstance.Close()
}
