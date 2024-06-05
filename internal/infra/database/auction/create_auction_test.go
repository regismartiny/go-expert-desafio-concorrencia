package auction

import (
	"context"
	"fullcycle-auction_go/configuration/logger"
	"fullcycle-auction_go/internal/entity/auction_entity"
	"log"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	mim "github.com/ONSdigital/dp-mongodb-in-memory"
)

type AuctionRepositoryTestSuite struct {
	suite.Suite
	context  context.Context
	DbServer *mim.Server
	Db       *mongo.Database
}

func (suite *AuctionRepositoryTestSuite) SetupSuite() {
	testCtx := context.Background()
	suite.context = testCtx

	server, err := mim.Start(testCtx, "6.0.4")

	if err != nil {
		log.Fatal(err.Error())
	}
	suite.NoError(err)
	suite.DbServer = server

	client, err := mongo.Connect(testCtx, options.Client().ApplyURI(server.URI()))
	suite.NoError(err)

	if err := client.Ping(testCtx, nil); err != nil {
		logger.Error("Error trying to ping test mongodb database", err)
	}
	suite.NoError(err)

	suite.Db = client.Database("testDb")

	os.Setenv("AUCTION_INTERVAL", "5s")
}

func (suite *AuctionRepositoryTestSuite) TearDownSuite() {
	suite.DbServer.Stop(suite.context)
}

func (suite *AuctionRepositoryTestSuite) TearDownTest() {
	err := suite.Db.Drop(suite.context)
	suite.NoError(err)
}

func TestSuite(t *testing.T) {
	suite.Run(t, new(AuctionRepositoryTestSuite))
}

func (suite *AuctionRepositoryTestSuite) TestGivenAnAction_WhenSave_ThenShouldSaveAuctionAndNotUpdateStatusToCompletedBeforeExpiration() {

	for range [1000]int{} {

		go func() {

			repo := NewAuctionRepository(suite.Db)

			auctionEntity, err := auction_entity.CreateAuction(
				"Produto",
				"Categoria",
				"Descrição",
				auction_entity.ProductCondition(1))

			suite.Nil(err)

			_ = repo.CreateAuction(suite.context, auctionEntity)

			time.Sleep(1 * time.Second)

			auction, _ := repo.FindAuctionById(suite.context, auctionEntity.Id)

			suite.Assertions.NotNil(auction)
			suite.Equal(auctionEntity.ProductName, auction.ProductName)
			suite.Equal(auctionEntity.Category, auction.Category)
			suite.Equal(auctionEntity.Description, auction.Description)
			suite.Equal(auction_entity.New, auction.Condition)
			suite.Equal(auction_entity.Active, auction.Status)

		}()
	}
}

func (suite *AuctionRepositoryTestSuite) TestGivenAnAction_WhenSave_ThenShouldSaveAuctionAndUpdateStatusToCompletedAfterExpiration() {

	for range [1000]int{} {

		go func() {

			repo := NewAuctionRepository(suite.Db)

			auctionEntity, err := auction_entity.CreateAuction(
				"Produto",
				"Categoria",
				"Descrição",
				auction_entity.ProductCondition(1))

			suite.Nil(err)

			_ = repo.CreateAuction(suite.context, auctionEntity)

			time.Sleep(5 * time.Second)

			auction, _ := repo.FindAuctionById(suite.context, auctionEntity.Id)

			suite.NotNil(auction)
			suite.Equal(auctionEntity.ProductName, auction.ProductName)
			suite.Equal(auctionEntity.Category, auction.Category)
			suite.Equal(auctionEntity.Description, auction.Description)
			suite.Equal(auction_entity.New, auction.Condition)
			suite.Equal(auction_entity.Completed, auction.Status)

		}()
	}
}
