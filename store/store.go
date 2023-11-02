package store

import (
	"database/sql"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/zhifengle/rss2cloud/rsssite"
)

type Store struct {
	DBInstance *sql.DB
}

func New(db *sql.DB) *Store {
	if db == nil {
		db, _ = sql.Open("sqlite3", "db.sqlite")
	}
	db.Exec("CREATE TABLE if not exists `rss_items` (`id` INTEGER PRIMARY KEY AUTOINCREMENT, `link` VARCHAR(255), `title` VARCHAR(255), `guid` VARCHAR(255), `pubDate` DATETIME, `creator` VARCHAR(255), `summary` TEXT, `content` VARCHAR(255), `isoDate` DATETIME, `categories` VARCHAR(255), `contentSnippet` VARCHAR(255), `done` TINYINT(1) DEFAULT 0, `magnet` VARCHAR(255) NOT NULL, `createdAt` DATETIME NOT NULL, `updatedAt` DATETIME NOT NULL)")
	db.Exec("CREATE TABLE if not exists `sites_status` (`id` INTEGER PRIMARY KEY AUTOINCREMENT, `name` VARCHAR(255), `needLogin` TINYINT(1), `abnormalOp` TINYINT(1), `createdAt` DATETIME NOT NULL, `updatedAt` DATETIME NOT NULL)")
	return &Store{
		DBInstance: db,
	}
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
