package auction

import (
	"context"
	"fmt"
	"fullcycle-auction_go/configuration/logger"
	"fullcycle-auction_go/internal/entity/auction_entity"
	"fullcycle-auction_go/internal/internal_error"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type AuctionEntityMongo struct {
	Id          string                          `bson:"_id"`
	ProductName string                          `bson:"product_name"`
	Category    string                          `bson:"category"`
	Description string                          `bson:"description"`
	Condition   auction_entity.ProductCondition `bson:"condition"`
	Status      auction_entity.AuctionStatus    `bson:"status"`
	Timestamp   int64                           `bson:"timestamp"`
}
type AuctionRepository struct {
	Collection      *mongo.Collection
	auctionInterval time.Duration
}

func NewAuctionRepository(database *mongo.Database) *AuctionRepository {
	ar := &AuctionRepository{
		Collection:      database.Collection("auctions"),
		auctionInterval: getAuctionInterval(),
	}

	go ar.initExpiredAuctionsMonitoring()

	return ar
}

func (ar *AuctionRepository) CreateAuction(
	ctx context.Context,
	auctionEntity *auction_entity.Auction) *internal_error.InternalError {

	auctionEntityMongo := &AuctionEntityMongo{
		Id:          auctionEntity.Id,
		ProductName: auctionEntity.ProductName,
		Category:    auctionEntity.Category,
		Description: auctionEntity.Description,
		Condition:   auctionEntity.Condition,
		Status:      auctionEntity.Status,
		Timestamp:   auctionEntity.Timestamp.Unix(),
	}
	_, err := ar.Collection.InsertOne(ctx, auctionEntityMongo)
	if err != nil {
		logger.Error("Error trying to insert auction", err)
		return internal_error.NewInternalServerError("Error trying to insert auction")
	}

	return nil
}

func (ar *AuctionRepository) initExpiredAuctionsMonitoring() {
	ctx := context.Background()
	ticker := time.NewTicker(1 * time.Second)

	for range ticker.C {

		activeAuctions := ar.findActiveAuctions(ctx)

		if len(activeAuctions) == 0 {
			continue
		}

		expiredAuctions := ar.filterExpiredAuctions(activeAuctions)

		if len(expiredAuctions) == 0 {
			continue
		}

		ar.updateAuctionsToStatusCompleted(ctx, expiredAuctions)

	}

}

func (ar *AuctionRepository) filterExpiredAuctions(auctions []auction_entity.Auction) []auction_entity.Auction {
	expiredAuctions := make([]auction_entity.Auction, 0)

	for _, auction := range auctions {

		auctionEndTime := auction.Timestamp.Add(ar.auctionInterval)
		auctionExpired := time.Now().After(auctionEndTime)

		if auctionExpired {
			expiredAuctions = append(expiredAuctions, auction)
		}

	}

	return expiredAuctions
}

func (ar *AuctionRepository) findActiveAuctions(ctx context.Context) []auction_entity.Auction {
	auctions, _ := ar.FindAuctions(ctx, auction_entity.Active, "", "")
	return auctions
}

func (ar *AuctionRepository) updateAuctionsToStatusCompleted(ctx context.Context, auctions []auction_entity.Auction) {
	logger.Info(fmt.Sprintf("Updating %d expired auctions", len(auctions)))

	for _, auction := range auctions {

		_, err := ar.Collection.UpdateOne(
			ctx,
			bson.M{"_id": auction.Id},
			bson.D{
				{"$set", bson.D{
					{"status", auction_entity.Completed},
				}},
			},
		)

		if err != nil {
			logger.Error("Error trying to update auction status to Completed", err)
		}

	}
}

func getAuctionInterval() time.Duration {
	auctionInterval := os.Getenv("AUCTION_INTERVAL")
	duration, err := time.ParseDuration(auctionInterval)
	if err != nil {
		return time.Minute * 5
	}

	return duration
}
