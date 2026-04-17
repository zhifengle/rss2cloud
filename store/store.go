package store

import (
	"database/sql"
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

func (s *Store) SaveMagnetItems(items []rsssite.MagnetItem) error {
	now := time.Now()
	for _, item := range items {
		sql := "INSERT INTO rss_items (`link`,`title`,`content`,`magnet`,`done`,`createdAt`,`updatedAt`) VALUES (?,?,?,?,?,?,?)"
		_, err := s.DBInstance.Exec(sql, item.Link, item.Title, item.Content, item.Magnet, 0, now, now)
		if err != nil {
			return err
		}
	}
	return nil
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

func (s *Store) Close() error {
	return s.DBInstance.Close()
}
