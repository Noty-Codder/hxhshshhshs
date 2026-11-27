package db

import (
	"errors"
	"time"

	"istkharmusic/internal/cache"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// RestrictCfg holds per-chat restrictor / VC-logger settings.
type RestrictCfg struct {
	ID         int64   `bson:"_id"`
	LinkRemove bool    `bson:"link_remove"`
	BioRemove  bool    `bson:"bio_remove"`
	VcLogger   bool    `bson:"vc_logger"`
	Approved   []int64 `bson:"approved"`
}

var restrictCache = cache.NewCache[*RestrictCfg](10 * time.Minute)

// GetRestrict returns the chat's settings (never nil).
func (db *Database) GetRestrict(chatID int64) *RestrictCfg {
	if cfg, ok := restrictCache.Get(toKey(chatID)); ok && cfg != nil {
		return cfg
	}
	ctx, cancel := db.ctx()
	defer cancel()

	cfg := &RestrictCfg{ID: chatID}
	err := db.DB.Collection("restrict").FindOne(ctx, bson.M{"_id": chatID}).Decode(cfg)
	if err != nil && !errors.Is(err, mongo.ErrNoDocuments) {
		return &RestrictCfg{ID: chatID}
	}
	restrictCache.Set(toKey(chatID), cfg)
	return cfg
}

func (db *Database) updateRestrict(chatID int64, update bson.M) error {
	ctx, cancel := db.ctx()
	defer cancel()
	_, err := db.DB.Collection("restrict").UpdateOne(ctx, bson.M{"_id": chatID}, update, options.UpdateOne().SetUpsert(true))
	if err == nil {
		restrictCache.Delete(toKey(chatID))
	}
	return err
}

func (db *Database) SetLinkRemove(chatID int64, on bool) error {
	return db.updateRestrict(chatID, bson.M{"$set": bson.M{"link_remove": on}})
}

func (db *Database) SetBioRemove(chatID int64, on bool) error {
	return db.updateRestrict(chatID, bson.M{"$set": bson.M{"bio_remove": on}})
}

func (db *Database) SetVcLogger(chatID int64, on bool) error {
	return db.updateRestrict(chatID, bson.M{"$set": bson.M{"vc_logger": on}})
}

func (db *Database) ApproveUser(chatID, userID int64) error {
	return db.updateRestrict(chatID, bson.M{"$addToSet": bson.M{"approved": userID}})
}

func (db *Database) UnapproveUser(chatID, userID int64) error {
	return db.updateRestrict(chatID, bson.M{"$pull": bson.M{"approved": userID}})
}

// LogChatEvent records "add" / "remove" events for /bdata.
func (db *Database) LogChatEvent(chatID int64, kind string) {
	ctx, cancel := db.ctx()
	defer cancel()
	_, _ = db.DB.Collection("events").InsertOne(ctx, bson.M{"chat_id": chatID, "kind": kind, "ts": time.Now().UTC()})
}

// CountChatEvents counts events of a kind since the given time.
func (db *Database) CountChatEvents(kind string, since time.Time) int64 {
	ctx, cancel := db.ctx()
	defer cancel()
	n, err := db.DB.Collection("events").CountDocuments(ctx, bson.M{"kind": kind, "ts": bson.M{"$gte": since.UTC()}})
	if err != nil {
		return 0
	}
	return n
}
