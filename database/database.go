package database

import (
	"github.com/glebarez/sqlite"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

var Pool *gorm.DB

type KnownUsers struct {
	ID         uint     `gorm:"primaryKey"`
	Username   string   `gorm:"unique"`
	KnownNames []string `gorm:"type:text"`
	CreatedAt  int64
}

type IndexedMessages struct {
	ID          uint   `gorm:"primaryKey"`
	MessageID   string `gorm:"unique"`
	Content     string `gorm:"type:text"`
	ChannelID   string `gorm:"index"`
	ChannelName string `gorm:"index"`
	GuildID     string `gorm:"index"`
	GuildName   string `gorm:"index"`
	AuthorID    string `gorm:"index"`
	Username    string `gorm:"index"`
	CreatedAt   int64
}

func InitDatabase() error {
	db, err := gorm.Open(sqlite.Open("database.db"), &gorm.Config{})

	if err != nil {
		log.Errorf("error opening database: %v", err)
		return err
	}

	log.Info("Database opened successfully")
	db.AutoMigrate(&KnownUsers{}, &IndexedMessages{})
	log.Info("Database migrated successfully")

	Pool = db
	log.Info("Database initialized successfully")
	return nil
}
