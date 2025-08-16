package database

import (
	"github.com/glebarez/sqlite"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var Pool *gorm.DB

type KnownUsers struct {
	ID         uint64     `gorm:"primaryKey"`
	Username   string     `gorm:"unique"`
	KnownNames []UserName `gorm:"foreignKey:UserID"`
	Facts      []UserFact `gorm:"foreignKey:UserID"`
	CreatedAt  int64
}

type UserFact struct {
	Id     uint64 `gorm:"primaryKey;autoIncrement"`
	Fact   string
	UserID uint64
}

type UserName struct {
	Id     uint64 `gorm:"primaryKey;autoIncrement"`
	Name   string
	UserID uint64
}

type IndexedMessages struct {
	ID                 uint   `gorm:"primaryKey"`
	MessageID          string `gorm:"unique"`
	Content            string `gorm:"type:text"`
	ChannelID          string `gorm:"index"`
	ChannelName        string `gorm:"index"`
	GuildID            string `gorm:"index"`
	GuildName          string `gorm:"index"`
	AuthorID           string `gorm:"index"`
	Username           string `gorm:"index"`
	ReferenceMessageID string `gorm:"index"`
	CreatedAt          int64
}

func InitDatabase() error {
	db, err := gorm.Open(sqlite.Open("database.db"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})

	if err != nil {
		log.Errorf("error opening database: %v", err)
		return err
	}

	log.Info("Database opened successfully")
	db.AutoMigrate(&KnownUsers{}, &IndexedMessages{}, &UserName{}, &UserFact{})
	log.Info("Database migrated successfully")

	Pool = db
	log.Info("Database initialized successfully")
	return nil
}

func AddUsername(user uint64, username string) {
	var knownUser KnownUsers
	Pool.Where(&KnownUsers{ID: user}).Attrs(&KnownUsers{ID: user}).FirstOrCreate(&knownUser)
	Pool.Save(&knownUser)

	// knownUser.KnownNames = append(knownUser.KnownNames, UserName{Name: username})
	Pool.Create(&UserName{Name: username, UserID: user})

}

func AddFact(user uint64, fact string) {
	var knownUser KnownUsers
	Pool.Where(&KnownUsers{ID: user}).Attrs(&KnownUsers{ID: user}).FirstOrCreate(&knownUser)
	Pool.Save(&knownUser)

	k := Pool.Create(&UserFact{Fact: fact, UserID: user})
	log.Println(k.Statement.Error)
}

func RemoveFact(user uint64, fact string) {
	var knownUser KnownUsers
	Pool.Where(&KnownUsers{ID: user}).Attrs(&KnownUsers{ID: user}).FirstOrCreate(&knownUser)

	// index := slices.Index(knownUser.Facts, fact)
	// if index != -1 {
	// 	knownUser.Facts = append(knownUser.Facts[:index], knownUser.Facts[index+1:]...)
	// }
	var factFact UserFact

	Pool.First(&factFact, "fact = ? AND user_id = ?", fact, user)
	Pool.Delete(&factFact)
}

func NamestToStrings(names []UserName) []string {
	res := make([]string, len(names))

	for _, name := range names {
		res = append(res, name.Name)
	}

	return res
}

func FactsToStrings(names []UserFact) []string {
	res := make([]string, len(names))

	for _, name := range names {
		res = append(res, name.Fact)
	}

	return res
}
